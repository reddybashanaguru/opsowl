package notes

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrBadRequest = errors.New("bad request")
	ErrCaseClosed = errors.New("case is closed — no new notes may be added")
)

type Service struct {
	q       *db.Queries
	broker  *Broker
	emitter timeline.Emitter
}

func NewService(q *db.Queries, broker *Broker, emitter timeline.Emitter) *Service {
	return &Service{q: q, broker: broker, emitter: emitter}
}

// ── Notes ────────────────────────────────────────────────────────────────────

type CreateNoteRequest struct {
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	ContentText      string   `json:"content_text"`
	NoteType         string   `json:"note_type"` // "internal" | "formal"
	AIGenerated      bool     `json:"ai_generated"`
	MentionedUserIDs []string `json:"mentioned_user_ids"`
}

type UpdateNoteRequest struct {
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	ContentText      string   `json:"content_text"`
	NoteType         string   `json:"note_type"`
	IsPinned         *bool    `json:"is_pinned"`
	IsHumanEdit      bool     `json:"is_human_edit"` // true when human edits AI-generated note
	MentionedUserIDs []string `json:"mentioned_user_ids"`
}

type NoteResponse struct {
	ID           string    `json:"id"`
	CaseID       string    `json:"case_id"`
	AuthorID     string    `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	ContentText  string    `json:"content_text"`
	NoteType     string    `json:"note_type"`
	IsPinned     bool      `json:"is_pinned"`
	AIGenerated  bool      `json:"ai_generated"`
	AIAssisted   bool      `json:"ai_assisted"`
	CommentCount int64     `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ListMeta struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	TotalCount int64   `json:"total_count"`
}

func (s *Service) Create(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CreateNoteRequest) (NoteResponse, error) {
	if strings.TrimSpace(req.Content) == "" {
		return NoteResponse{}, ErrBadRequest
	}
	if c, err := s.q.GetCaseByID(ctx, caseID); err == nil && c.Status == db.CaseStatusClosed {
		return NoteResponse{}, ErrCaseClosed
	}
	nt := db.NoteTypeEnumInternal
	if req.NoteType == "formal" {
		nt = db.NoteTypeEnumFormal
	}

	note, err := s.q.CreateNote(ctx, db.CreateNoteParams{
		CaseID:      caseID,
		AuthorID:    claims.UserID,
		Title:       req.Title,
		Content:     req.Content,
		ContentText: req.ContentText,
		NoteType:    nt,
		IsPinned:    false,
		AiGenerated: req.AIGenerated,
	})
	if err != nil {
		return NoteResponse{}, err
	}

	// Insert mentions
	for _, uid := range req.MentionedUserIDs {
		mid, err := uuid.Parse(uid)
		if err != nil {
			continue
		}
		_, _ = s.q.CreateMention(ctx, db.CreateMentionParams{
			NoteID:          uuid.NullUUID{UUID: note.ID, Valid: true},
			MentionedUserID: mid,
			MentionedBy:     claims.UserID,
		})
	}

	full, err := s.q.GetNoteByID(ctx, note.ID)
	if err != nil {
		return NoteResponse{}, err
	}

	s.broker.Publish(NoteEvent{
		Type:   "note_added",
		CaseID: caseID.String(),
		Payload: map[string]any{
			"note_id":    note.ID.String(),
			"author_id":  claims.UserID.String(),
			"title":      note.Title,
			"note_type":  string(note.NoteType),
			"created_at": note.CreatedAt,
		},
	})

	actorID := claims.UserID
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:     caseID,
		EventType:  db.TimelineEventTypeNoteAdded,
		Source:     db.TimelineEventSourceUser,
		ActorID:    &actorID,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"note_id":   note.ID.String(),
			"title":     note.Title,
			"note_type": string(note.NoteType),
		},
	})

	return toNoteResponse(full), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (NoteResponse, error) {
	n, err := s.q.GetNoteByID(ctx, id)
	if err != nil {
		return NoteResponse{}, ErrNotFound
	}
	return toNoteResponse(n), nil
}

func (s *Service) List(ctx context.Context, caseID uuid.UUID, noteType string, cursor uuid.UUID, limit int32) ([]NoteResponse, ListMeta, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.q.ListNotesByCase(ctx, db.ListNotesByCaseParams{
		CaseID:  caseID,
		Column2: noteType,
		Column3: cursor,
		Limit:   limit + 1, // fetch one extra to detect has_more
	})
	if err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.q.CountNotesByCase(ctx, db.CountNotesByCaseParams{
		CaseID:  caseID,
		Column2: noteType,
	})

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID.String()
		nextCursor = &last
	}

	out := make([]NoteResponse, len(rows))
	for i, r := range rows {
		out[i] = toNoteResponseFromList(r)
	}

	return out, ListMeta{NextCursor: nextCursor, HasMore: hasMore, TotalCount: total}, nil
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, noteID uuid.UUID, req UpdateNoteRequest) (NoteResponse, error) {
	existing, err := s.q.GetNoteByID(ctx, noteID)
	if err != nil {
		return NoteResponse{}, ErrNotFound
	}
	if existing.AuthorID != claims.UserID && claims.Role != "supervisor" &&
		claims.Role != "manager" && claims.Role != "admin" && claims.Role != "super_admin" {
		return NoteResponse{}, ErrForbidden
	}

	isPinned := existing.IsPinned
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}

	_, err = s.q.UpdateNote(ctx, db.UpdateNoteParams{
		ID:          noteID,
		Column2:     req.Title,
		Column3:     req.Content,
		Column4:     req.ContentText,
		Column5:     req.NoteType,
		IsPinned:    isPinned,
		AiGenerated: req.IsHumanEdit,
	})
	if err != nil {
		return NoteResponse{}, err
	}

	// Handle new mentions
	for _, uid := range req.MentionedUserIDs {
		mid, err := uuid.Parse(uid)
		if err != nil {
			continue
		}
		_, _ = s.q.CreateMention(ctx, db.CreateMentionParams{
			NoteID:          uuid.NullUUID{UUID: noteID, Valid: true},
			MentionedUserID: mid,
			MentionedBy:     claims.UserID,
		})
	}

	full, err := s.q.GetNoteByID(ctx, noteID)
	if err != nil {
		return NoteResponse{}, err
	}

	actorID := claims.UserID
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:     full.CaseID,
		EventType:  db.TimelineEventTypeNoteEdited,
		Source:     db.TimelineEventSourceUser,
		ActorID:    &actorID,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"note_id": noteID.String(),
			"title":   full.Title,
		},
	})

	return toNoteResponse(full), nil
}

func (s *Service) Delete(ctx context.Context, claims *auth.Claims, noteID uuid.UUID) error {
	existing, err := s.q.GetNoteByID(ctx, noteID)
	if err != nil {
		return ErrNotFound
	}
	if existing.AuthorID != claims.UserID && claims.Role != "supervisor" &&
		claims.Role != "manager" && claims.Role != "admin" && claims.Role != "super_admin" {
		return ErrForbidden
	}
	_, err = s.q.SoftDeleteNote(ctx, noteID)
	return err
}

// ── Comments ─────────────────────────────────────────────────────────────────

type CreateCommentRequest struct {
	Content          string  `json:"content"`
	NoteID           *string `json:"note_id"`
	TimelineEventID  *string `json:"timeline_event_id"`
	ParentID         *string `json:"parent_id"`
	MentionedUserIDs []string `json:"mentioned_user_ids"`
}

type CommentResponse struct {
	ID              string    `json:"id"`
	CaseID          string    `json:"case_id"`
	NoteID          *string   `json:"note_id,omitempty"`
	TimelineEventID *string   `json:"timeline_event_id,omitempty"`
	AuthorID        string    `json:"author_id"`
	AuthorName      string    `json:"author_name"`
	ParentID        *string   `json:"parent_id,omitempty"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s *Service) CreateComment(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CreateCommentRequest) (CommentResponse, error) {
	if strings.TrimSpace(req.Content) == "" {
		return CommentResponse{}, ErrBadRequest
	}

	params := db.CreateCommentParams{
		CaseID:   caseID,
		AuthorID: claims.UserID,
		Content:  req.Content,
	}
	if req.NoteID != nil {
		if nid, err := uuid.Parse(*req.NoteID); err == nil {
			params.NoteID = uuid.NullUUID{UUID: nid, Valid: true}
		}
	}
	if req.TimelineEventID != nil {
		if tid, err := uuid.Parse(*req.TimelineEventID); err == nil {
			params.TimelineEventID = uuid.NullUUID{UUID: tid, Valid: true}
		}
	}
	if req.ParentID != nil {
		if pid, err := uuid.Parse(*req.ParentID); err == nil {
			params.ParentID = uuid.NullUUID{UUID: pid, Valid: true}
		}
	}

	c, err := s.q.CreateComment(ctx, params)
	if err != nil {
		return CommentResponse{}, err
	}

	// Insert mentions
	for _, uid := range req.MentionedUserIDs {
		mid, err := uuid.Parse(uid)
		if err != nil {
			continue
		}
		_, _ = s.q.CreateMention(ctx, db.CreateMentionParams{
			CommentID:       uuid.NullUUID{UUID: c.ID, Valid: true},
			MentionedUserID: mid,
			MentionedBy:     claims.UserID,
		})
	}

	s.broker.Publish(NoteEvent{
		Type:   "comment_added",
		CaseID: caseID.String(),
		Payload: map[string]any{
			"comment_id": c.ID.String(),
			"author_id":  claims.UserID.String(),
			"excerpt":    truncate(req.Content, 80),
			"created_at": c.CreatedAt,
		},
	})

	return commentToResponse(c, claims.Email), nil
}

func (s *Service) ListCommentsByNote(ctx context.Context, noteID uuid.UUID) ([]CommentResponse, error) {
	rows, err := s.q.ListCommentsByNote(ctx, uuid.NullUUID{UUID: noteID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]CommentResponse, len(rows))
	for i, r := range rows {
		out[i] = listCommentRowToResponse(r)
	}
	return out, nil
}

// ── Mentions ─────────────────────────────────────────────────────────────────

func (s *Service) MarkMentionRead(ctx context.Context, claims *auth.Claims, mentionID uuid.UUID) error {
	_, err := s.q.MarkMentionRead(ctx, db.MarkMentionReadParams{
		ID:              mentionID,
		MentionedUserID: claims.UserID,
	})
	return err
}

func (s *Service) UnreadCount(ctx context.Context, claims *auth.Claims) (int64, error) {
	return s.q.GetUnreadMentionCount(ctx, claims.UserID)
}

// ── Activity ─────────────────────────────────────────────────────────────────

type ActivityItem struct {
	ID             string    `json:"id"`
	CaseID         string    `json:"case_id"`
	ActivityType   string    `json:"activity_type"`
	AuthorID       string    `json:"author_id"`
	ContentPreview string    `json:"content_preview"`
	SubType        string    `json:"sub_type"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *Service) ListActivity(ctx context.Context, caseID uuid.UUID, cursor uuid.UUID, limit int32) ([]ActivityItem, ListMeta, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	rows, err := s.q.ListCaseActivity(ctx, db.ListCaseActivityParams{
		CaseID:  caseID,
		Column2: cursor,
		Limit:   limit + 1,
	})
	if err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID.String()
		nextCursor = &last
	}

	out := make([]ActivityItem, len(rows))
	for i, r := range rows {
		out[i] = ActivityItem{
			ID:             r.ID.String(),
			CaseID:         r.CaseID.String(),
			ActivityType:   r.ActivityType,
			AuthorID:       r.AuthorID.String(),
			ContentPreview: r.ContentPreview,
			SubType:        r.SubType,
			CreatedAt:      r.CreatedAt,
		}
	}

	return out, ListMeta{NextCursor: nextCursor, HasMore: hasMore}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func toNoteResponse(n db.GetNoteByIDRow) NoteResponse {
	return NoteResponse{
		ID:           n.ID.String(),
		CaseID:       n.CaseID.String(),
		AuthorID:     n.AuthorID.String(),
		AuthorName:   n.AuthorName,
		Title:        n.Title,
		Content:      n.Content,
		ContentText:  n.ContentText,
		NoteType:     string(n.NoteType),
		IsPinned:     n.IsPinned,
		AIGenerated:  n.AiGenerated,
		AIAssisted:   n.AiAssisted,
		CommentCount: n.CommentCount,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}
}

func toNoteResponseFromList(n db.ListNotesByCaseRow) NoteResponse {
	return NoteResponse{
		ID:           n.ID.String(),
		CaseID:       n.CaseID.String(),
		AuthorID:     n.AuthorID.String(),
		AuthorName:   n.AuthorName,
		Title:        n.Title,
		Content:      n.Content,
		ContentText:  n.ContentText,
		NoteType:     string(n.NoteType),
		IsPinned:     n.IsPinned,
		AIGenerated:  n.AiGenerated,
		AIAssisted:   n.AiAssisted,
		CommentCount: n.CommentCount,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}
}

func commentToResponse(c db.Comment, authorName string) CommentResponse {
	r := CommentResponse{
		ID:        c.ID.String(),
		CaseID:    c.CaseID.String(),
		AuthorID:  c.AuthorID.String(),
		AuthorName: authorName,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
	if c.NoteID.Valid {
		s := c.NoteID.UUID.String()
		r.NoteID = &s
	}
	if c.TimelineEventID.Valid {
		s := c.TimelineEventID.UUID.String()
		r.TimelineEventID = &s
	}
	if c.ParentID.Valid {
		s := c.ParentID.UUID.String()
		r.ParentID = &s
	}
	return r
}

func listCommentRowToResponse(r db.ListCommentsByNoteRow) CommentResponse {
	cr := CommentResponse{
		ID:         r.ID.String(),
		AuthorID:   r.AuthorID.String(),
		AuthorName: r.AuthorName,
		Content:    r.Content,
		CreatedAt:  r.CreatedAt,
	}
	if r.NoteID.Valid {
		s := r.NoteID.UUID.String()
		cr.NoteID = &s
	}
	if r.TimelineEventID.Valid {
		s := r.TimelineEventID.UUID.String()
		cr.TimelineEventID = &s
	}
	if r.ParentID.Valid {
		s := r.ParentID.UUID.String()
		cr.ParentID = &s
	}
	return cr
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
