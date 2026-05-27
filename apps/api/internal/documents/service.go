package documents

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/scanner"
	"github.com/reddybashanaguru/opsowl/internal/storage"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

type Service struct {
	repo            *Repository
	store           storage.Client
	emitter         timeline.Emitter
	logger          *zap.Logger
	httpClient      *http.Client
	aiServiceURL    string
	aiServiceSecret string
	av              scanner.Scanner
	retryBackoff    []time.Duration
	processingWg    sync.WaitGroup
}

func NewService(repo *Repository, store storage.Client, emitter timeline.Emitter, logger *zap.Logger, aiServiceURL string) *Service {
	return &Service{
		repo:         repo,
		store:        store,
		emitter:      emitter,
		logger:       logger,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		aiServiceURL: aiServiceURL,
		av:           scanner.NoopScanner{},
		retryBackoff: processBackoff,
	}
}

// WithServiceSecret sets the shared secret used to authenticate calls to the
// Python AI service. Must match the AI_SERVICE_SECRET environment variable.
func (s *Service) WithServiceSecret(secret string) *Service {
	s.aiServiceSecret = secret
	return s
}

// WithScanner injects an antivirus scanner used during Upload.
// Defaults to NoopScanner (pass-through) when not called.
func (s *Service) WithScanner(sc scanner.Scanner) *Service {
	s.av = sc
	return s
}

// WithRetryBackoff overrides the default processing retry delays.
// Intended for tests that need immediate retries without waiting 30s/5min.
func (s *Service) WithRetryBackoff(delays []time.Duration) *Service {
	s.retryBackoff = delays
	return s
}

// RecoverStuckDocuments marks any documents stuck in 'processing' for longer
// than 15 minutes as 'failed'. Safe to call multiple times.
func (s *Service) RecoverStuckDocuments(ctx context.Context) {
	n, err := s.repo.MarkStuckAsFailed(ctx, 15*time.Minute)
	if err != nil {
		s.logger.Error("failed to recover stuck documents", zap.Error(err))
		return
	}
	if n > 0 {
		s.logger.Warn("recovered stuck documents", zap.Int64("count", n))
	}
}

// StartPeriodicRecovery starts a background loop that calls RecoverStuckDocuments
// every [interval]. This handles documents that get stuck after startup (e.g. if
// the AI service hangs mid-processing without the container restarting).
// The loop exits when [ctx] is cancelled.
func (s *Service) StartPeriodicRecovery(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s.RecoverStuckDocuments(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// launchProcessAsync starts a tracked goroutine for document processing.
// Using processingWg allows DrainProcessing to wait for in-flight jobs during
// graceful shutdown instead of abandoning them.
func (s *Service) launchProcessAsync(docID, caseID uuid.UUID, storedPath, filename, fileType, mimeType string) {
	s.processingWg.Add(1)
	go func() {
		defer s.processingWg.Done()
		s.processAsync(docID, caseID, storedPath, filename, fileType, mimeType)
	}()
}

// DrainProcessing waits up to [timeout] for all in-flight processAsync goroutines
// to finish. Call during graceful shutdown after the HTTP server has stopped.
// Any goroutines still running after the timeout will be recovered as 'failed'
// by RecoverStuckDocuments on next startup.
func (s *Service) DrainProcessing(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		s.processingWg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// Upload validates, stores, and persists a document, then kicks off async processing.
func (s *Service) Upload(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, filename string, reader io.Reader, totalSize int64) (*DocumentResponse, error) {
	if totalSize > MaxFileSizeBytes {
		return nil, ErrFileTooLarge
	}

	// Read first 512 bytes for MIME detection while retaining them for upload.
	header := make([]byte, 512)
	n, err := io.ReadAtLeast(reader, header, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}
	header = header[:n]

	fi, err := Validate(filename, header)
	if err != nil {
		return nil, err
	}

	docID := uuid.New()
	filename = sanitizeFilename(filename)
	key := fmt.Sprintf("documents/%s/%s/%s", caseID.String(), docID.String(), filename)

	// Compute SHA-256 while streaming to storage — no extra memory allocation.
	// Simultaneously pipe a copy to the antivirus scanner.
	hasher := sha256.New()
	combined := newPrefixReader(header, io.TeeReader(reader, hasher))

	pr, pw := io.Pipe()
	var (
		wg      sync.WaitGroup
		virusErr error
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		virusErr = s.av.ScanReader(pr)
		_ = pr.Close()
	}()

	teeForScan := io.TeeReader(combined, pw)
	if uploadErr := s.store.Upload(ctx, key, teeForScan, totalSize, fi.MIMEType); uploadErr != nil {
		_ = pw.CloseWithError(uploadErr)
		wg.Wait()
		return nil, fmt.Errorf("storage upload: %w", uploadErr)
	}
	_ = pw.Close()
	wg.Wait()

	if errors.Is(virusErr, scanner.ErrVirusFound) {
		_ = s.store.Delete(ctx, key)
		return nil, ErrVirusFound
	}
	if virusErr != nil && !errors.Is(virusErr, scanner.ErrUnavailable) {
		s.logger.Warn("antivirus scan error — upload allowed (graceful degradation)", zap.Error(virusErr))
	} else if errors.Is(virusErr, scanner.ErrUnavailable) {
		s.logger.Warn("antivirus unavailable — upload allowed", zap.Error(virusErr))
	}

	fileHash := fmt.Sprintf("%x", hasher.Sum(nil))

	doc, err := s.repo.Create(ctx, db.CreateDocumentParams{
		ID:               docID,
		CaseID:           caseID,
		UploadedBy:       claims.UserID,
		OriginalFilename: filename,
		StoredPath:       key,
		FileType:         fi.FileType,
		MimeType:         fi.MIMEType,
		FileSizeBytes:    totalSize,
	})
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, err
	}

	// Store SHA-256 for chain-of-custody verification.
	go func() {
		_ = s.repo.q.UpdateDocumentHash(context.Background(), db.UpdateDocumentHashParams{
			ID:         doc.ID,
			FileSha256: sql.NullString{String: fileHash, Valid: true},
		})
	}()

	s.emitTimeline(ctx, caseID, db.TimelineEventTypeDocumentUploaded, map[string]interface{}{
		"document_id": docID.String(),
		"filename":    filename,
		"file_type":   fi.FileType,
		"size_bytes":  totalSize,
	})

	s.launchProcessAsync(doc.ID, caseID, key, filename, fi.FileType, fi.MIMEType)

	url, _ := s.store.PresignedURL(ctx, key, PresignExpiry)
	return &DocumentResponse{
		ID:               doc.ID,
		CaseID:           doc.CaseID,
		OriginalFilename: doc.OriginalFilename,
		FileType:         doc.FileType,
		MimeType:         doc.MimeType,
		FileSizeBytes:    doc.FileSizeBytes,
		ProcessingStatus: doc.ProcessingStatus,
		UploadedBy: UploaderRef{
			ID:    doc.UploadedBy,
			Email: claims.Email,
		},
		DownloadURL: url,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*DocumentResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, row)
}

func (s *Service) List(ctx context.Context, caseID uuid.UUID, f ListFilters) ([]*DocumentResponse, ListMeta, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var nullStatus db.NullDocumentProcessingStatus
	if f.Status != nil {
		nullStatus = db.NullDocumentProcessingStatus{DocumentProcessingStatus: *f.Status, Valid: true}
	}

	rows, err := s.repo.ListByCase(ctx, db.ListDocumentsByCaseParams{
		CaseID:     caseID,
		Status:     nullStatus,
		CursorTime: nullTime(f.CursorTime),
		CursorID:   nullUUID(f.CursorID),
		RowLimit:   limit + 1,
	})
	if err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.repo.Count(ctx, caseID)

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		c := encodeCursor(last.CreatedAt, last.ID)
		nextCursor = &c
	}

	out := make([]*DocumentResponse, 0, len(rows))
	for _, row := range rows {
		resp, err := s.buildResponse(ctx, rowToGetRow(row))
		if err != nil {
			s.logger.Warn("failed to build document response", zap.Error(err))
			continue
		}
		out = append(out, resp)
	}

	return out, ListMeta{TotalCount: total, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *Service) ListAll(ctx context.Context, status *db.DocumentProcessingStatus, cursorTime *time.Time, cursorID *uuid.UUID, limit int32) ([]*GlobalDocumentResponse, ListMeta, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var nullStatus db.NullDocumentProcessingStatus
	if status != nil {
		nullStatus = db.NullDocumentProcessingStatus{DocumentProcessingStatus: *status, Valid: true}
	}

	rows, err := s.repo.ListAll(ctx, db.ListAllDocumentsParams{
		Status:     nullStatus,
		CursorTime: nullTime(cursorTime),
		CursorID:   nullUUID(cursorID),
		RowLimit:   limit + 1,
	})
	if err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.repo.CountAll(ctx, nullStatus)

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		c := encodeCursor(last.CreatedAt, last.ID)
		nextCursor = &c
	}

	out := make([]*GlobalDocumentResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, &GlobalDocumentResponse{
			ID:               row.ID,
			CaseID:           row.CaseID,
			CaseNumber:       row.CaseNumber,
			CaseTitle:        row.CaseTitle,
			OriginalFilename: row.OriginalFilename,
			FileType:         row.FileType,
			MimeType:         row.MimeType,
			FileSizeBytes:    row.FileSizeBytes,
			ProcessingStatus: row.ProcessingStatus,
			PageCount:        int32Ptr(row.PageCount),
			ChunkCount:       int32Ptr(row.ChunkCount),
			ErrorMessage:     strPtr(row.ErrorMessage),
			UploadedBy: UploaderRef{
				ID:       row.UploadedByID,
				FullName: row.UploadedByName,
				Email:    row.UploadedByEmail,
			},
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}

	return out, ListMeta{TotalCount: total, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *Service) Delete(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, id uuid.UUID) error {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if row.CaseID != caseID {
		return ErrNotFound
	}
	if row.UploadedBy != claims.UserID &&
		claims.Role != db.UserRoleManager &&
		claims.Role != db.UserRoleAdmin &&
		claims.Role != db.UserRoleSuperAdmin {
		return ErrForbidden
	}
	storedPath := row.StoredPath
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	go func() { _ = s.store.Delete(context.Background(), storedPath) }()
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (s *Service) buildResponse(ctx context.Context, row db.GetDocumentByIDRow) (*DocumentResponse, error) {
	url, err := s.store.PresignedURL(ctx, row.StoredPath, PresignExpiry)
	if err != nil {
		s.logger.Warn("failed to generate presigned URL", zap.Error(err))
		url = ""
	}
	return &DocumentResponse{
		ID:               row.ID,
		CaseID:           row.CaseID,
		OriginalFilename: row.OriginalFilename,
		FileType:         row.FileType,
		MimeType:         row.MimeType,
		FileSizeBytes:    row.FileSizeBytes,
		ProcessingStatus: row.ProcessingStatus,
		PageCount:        int32Ptr(row.PageCount),
		ChunkCount:       int32Ptr(row.ChunkCount),
		ErrorMessage:     strPtr(row.ErrorMessage),
		UploadedBy: UploaderRef{
			ID:       row.UploadedBy,
			FullName: row.UploadedByName,
			Email:    row.UploadedByEmail,
		},
		DownloadURL: url,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (s *Service) emitTimeline(ctx context.Context, caseID uuid.UUID, evtType db.TimelineEventType, payload map[string]interface{}) {
	if err := s.emitter.Emit(ctx, timeline.Event{
		CaseID:     caseID,
		EventType:  evtType,
		Source:     db.TimelineEventSourceSystem,
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
	}); err != nil {
		s.logger.Warn("failed to emit timeline event", zap.Error(err))
	}
}

// sanitizeFilename strips directory traversal sequences and path separators so
// the filename is safe to embed in a MinIO object key.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" || name == "." {
		name = "upload"
	}
	return name
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

type docCursor struct {
	T  time.Time `json:"t"`
	ID uuid.UUID `json:"id"`
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	b, _ := json.Marshal(docCursor{T: t, ID: id})
	return base64.StdEncoding.EncodeToString(b)
}

func decodeCursor(s string) (*time.Time, *uuid.UUID, error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, nil, err
	}
	var p docCursor
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil, err
	}
	return &p.T, &p.ID, nil
}

func rowToGetRow(r db.ListDocumentsByCaseRow) db.GetDocumentByIDRow {
	return db.GetDocumentByIDRow{
		ID:               r.ID,
		CaseID:           r.CaseID,
		UploadedBy:       r.UploadedBy,
		OriginalFilename: r.OriginalFilename,
		StoredPath:       r.StoredPath,
		FileType:         r.FileType,
		MimeType:         r.MimeType,
		FileSizeBytes:    r.FileSizeBytes,
		ProcessingStatus: r.ProcessingStatus,
		PageCount:        r.PageCount,
		ChunkCount:       r.ChunkCount,
		ErrorMessage:     r.ErrorMessage,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		UploadedByName:   r.UploadedByName,
		UploadedByEmail:  r.UploadedByEmail,
	}
}

// Retry re-triggers async processing for a failed document.
func (s *Service) Retry(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, docID uuid.UUID) (*DocumentResponse, error) {
	row, err := s.repo.q.GetDocumentForRetry(ctx, docID)
	if err != nil {
		return nil, ErrNotFound
	}
	if row.CaseID != caseID {
		return nil, ErrNotFound
	}
	if row.ProcessingStatus == db.DocumentProcessingStatusProcessing {
		return nil, fmt.Errorf("document is already being processed")
	}
	if row.RetryCount >= 3 {
		return nil, fmt.Errorf("maximum retry attempts reached")
	}

	updated, err := s.repo.q.IncrementDocumentRetry(ctx, docID)
	if err != nil {
		return nil, err
	}

	s.launchProcessAsync(updated.ID, updated.CaseID, updated.StoredPath, updated.OriginalFilename, updated.FileType, updated.MimeType)

	return s.Get(ctx, docID)
}
