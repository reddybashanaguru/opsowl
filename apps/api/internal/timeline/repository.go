package timeline

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) List(ctx context.Context, caseID uuid.UUID, f ListFilters) ([]db.ListTimelineEventsRow, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	p := db.ListTimelineEventsParams{
		CaseID:   caseID,
		RowLimit: limit + 1, // fetch one extra to detect hasMore
	}
	if f.EventType != nil {
		p.EventType = db.NullTimelineEventType{TimelineEventType: *f.EventType, Valid: true}
	}
	if f.Source != nil {
		p.Source = db.NullTimelineEventSource{TimelineEventSource: *f.Source, Valid: true}
	}
	if f.ActorID != nil {
		p.ActorID = uuid.NullUUID{UUID: *f.ActorID, Valid: true}
	}
	if f.From != nil {
		p.FromTime = sql.NullTime{Time: *f.From, Valid: true}
	}
	if f.To != nil {
		p.ToTime = sql.NullTime{Time: *f.To, Valid: true}
	}
	if f.CursorTime != nil {
		p.CursorTime = sql.NullTime{Time: *f.CursorTime, Valid: true}
		if f.CursorID != nil {
			p.CursorID = uuid.NullUUID{UUID: *f.CursorID, Valid: true}
		}
	}
	return r.q.ListTimelineEvents(ctx, p)
}

func (r *Repository) Count(ctx context.Context, caseID uuid.UUID, f ListFilters) (int64, error) {
	p := db.CountTimelineEventsParams{CaseID: caseID}
	if f.EventType != nil {
		p.EventType = db.NullTimelineEventType{TimelineEventType: *f.EventType, Valid: true}
	}
	if f.Source != nil {
		p.Source = db.NullTimelineEventSource{TimelineEventSource: *f.Source, Valid: true}
	}
	if f.ActorID != nil {
		p.ActorID = uuid.NullUUID{UUID: *f.ActorID, Valid: true}
	}
	if f.From != nil {
		p.FromTime = sql.NullTime{Time: *f.From, Valid: true}
	}
	if f.To != nil {
		p.ToTime = sql.NullTime{Time: *f.To, Valid: true}
	}
	return r.q.CountTimelineEvents(ctx, p)
}

func (r *Repository) Insert(ctx context.Context, p db.InsertTimelineEventParams) (db.TimelineEvent, error) {
	return r.q.InsertTimelineEvent(ctx, p)
}
