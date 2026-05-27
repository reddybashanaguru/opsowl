package documents

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"go.uber.org/zap"
)

type aiProcessRequest struct {
	DocumentID       string `json:"document_id"`
	CaseID           string `json:"case_id"`
	StoredPath       string `json:"stored_path"`
	OriginalFilename string `json:"original_filename"`
	FileType         string `json:"file_type"`
	MimeType         string `json:"mime_type"`
}

type aiProcessResponse struct {
	Success      bool    `json:"success"`
	PageCount    *int32  `json:"page_count"`
	ChunkCount   *int32  `json:"chunk_count"`
	ErrorMessage *string `json:"error_message"`
}

// processBackoff defines the wait between automatic retry attempts.
// Attempt 1 → immediate, attempt 2 → 30s, attempt 3 → 5min.
var processBackoff = []time.Duration{0, 30 * time.Second, 5 * time.Minute}

// processAsync is called in a goroutine after upload succeeds.
// It marks the document as processing, calls the AI service with up to 3 attempts
// (backoff: 0s / 30s / 5min), then writes the final status.
func (s *Service) processAsync(docID, caseID uuid.UUID, storedPath, filename, fileType, mimeType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	if _, err := s.repo.UpdateStatus(ctx, docID, db.DocumentProcessingStatusProcessing); err != nil {
		s.logger.Error("failed to mark document processing", zap.String("doc_id", docID.String()), zap.Error(err))
		return
	}

	var lastErr error
	for attempt, delay := range s.retryBackoff {
		if delay > 0 {
			s.logger.Info("retrying document processing",
				zap.String("doc_id", docID.String()),
				zap.Int("attempt", attempt+1),
				zap.Duration("backoff", delay))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				errMsg := "processing cancelled"
				s.finalizeProcessing(ctx, docID, caseID, db.DocumentProcessingStatusFailed, nil, nil, &errMsg)
				return
			}
		}

		result, err := s.callAIService(ctx, docID, caseID, storedPath, filename, fileType, mimeType)
		if err == nil {
			status := db.DocumentProcessingStatusCompleted
			if !result.Success {
				status = db.DocumentProcessingStatusFailed
			}
			s.finalizeProcessing(ctx, docID, caseID, status, result.PageCount, result.ChunkCount, result.ErrorMessage)
			return
		}
		lastErr = err
		s.logger.Warn("AI service call failed — will retry",
			zap.String("doc_id", docID.String()),
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", len(s.retryBackoff)),
			zap.Error(err))
	}

	// All attempts exhausted.
	s.logger.Error("AI service permanently failed for document",
		zap.String("doc_id", docID.String()), zap.Error(lastErr))
	errMsg := lastErr.Error()
	s.finalizeProcessing(ctx, docID, caseID, db.DocumentProcessingStatusFailed, nil, nil, &errMsg)
}

func (s *Service) callAIService(ctx context.Context, docID, caseID uuid.UUID, storedPath, filename, fileType, mimeType string) (*aiProcessResponse, error) {
	body, _ := json.Marshal(aiProcessRequest{
		DocumentID:       docID.String(),
		CaseID:           caseID.String(),
		StoredPath:       storedPath,
		OriginalFilename: filename,
		FileType:         fileType,
		MimeType:         mimeType,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.aiServiceURL+"/process-document", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.aiServiceSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.aiServiceSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned %d", resp.StatusCode)
	}

	var result aiProcessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) finalizeProcessing(ctx context.Context, docID, caseID uuid.UUID, status db.DocumentProcessingStatus, pageCount, chunkCount *int32, errMsg *string) {
	nullErrMsg := sql.NullString{}
	if errMsg != nil {
		nullErrMsg = sql.NullString{String: *errMsg, Valid: true}
	}

	_, err := s.repo.UpdateProcessingResult(ctx, db.UpdateDocumentProcessingResultParams{
		ID:               docID,
		ProcessingStatus: status,
		PageCount:        nullInt32(pageCount),
		ChunkCount:       nullInt32(chunkCount),
		ErrorMessage:     nullErrMsg,
	})
	if err != nil {
		s.logger.Error("failed to update processing result", zap.String("doc_id", docID.String()), zap.Error(err))
		return
	}

	eventType := db.TimelineEventTypeDocumentProcessed
	metadata := map[string]interface{}{
		"document_id": docID.String(),
		"status":      string(status),
	}
	if pageCount != nil {
		metadata["page_count"] = *pageCount
	}
	if chunkCount != nil {
		metadata["chunk_count"] = *chunkCount
	}
	s.emitTimeline(ctx, caseID, eventType, metadata)
}
