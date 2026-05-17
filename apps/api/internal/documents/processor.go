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

// processAsync is called in a goroutine after upload succeeds.
// It marks the document as processing, calls the AI service, then updates the final result.
func (s *Service) processAsync(docID, caseID uuid.UUID, storedPath, filename, fileType, mimeType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if _, err := s.repo.UpdateStatus(ctx, docID, db.DocumentProcessingStatusProcessing); err != nil {
		s.logger.Error("failed to mark document processing", zap.String("doc_id", docID.String()), zap.Error(err))
		return
	}

	result, err := s.callAIService(ctx, docID, caseID, storedPath, filename, fileType, mimeType)
	if err != nil {
		s.logger.Error("AI service call failed", zap.String("doc_id", docID.String()), zap.Error(err))
		errMsg := err.Error()
		s.finalizeProcessing(ctx, docID, caseID, db.DocumentProcessingStatusFailed, nil, nil, &errMsg)
		return
	}

	status := db.DocumentProcessingStatusCompleted
	if !result.Success {
		status = db.DocumentProcessingStatusFailed
	}
	s.finalizeProcessing(ctx, docID, caseID, status, result.PageCount, result.ChunkCount, result.ErrorMessage)
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
