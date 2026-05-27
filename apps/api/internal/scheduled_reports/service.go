package scheduled_reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/mail"
)

type Service struct {
	q      *db.Queries
	mailer *mail.Sender
	logger *zap.Logger
}

func NewService(q *db.Queries, mailer *mail.Sender, logger *zap.Logger) *Service {
	return &Service{q: q, mailer: mailer, logger: logger}
}

// ─── CRUD ──────────────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, req CreateScheduleRequest, claims *auth.Claims) (*Schedule, error) {
	if err := validateReq(req); err != nil {
		return nil, err
	}
	sendHour := int32(7)
	if req.SendHour != nil {
		sendHour = *req.SendHour
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	filtersJSON, _ := json.Marshal(req.Filters)
	var createdBy uuid.NullUUID
	if claims != nil {
		createdBy = uuid.NullUUID{UUID: claims.UserID, Valid: true}
	}
	var dow sql.NullInt32
	if req.Frequency == FreqWeekly && req.DayOfWeek != nil {
		dow = sql.NullInt32{Int32: *req.DayOfWeek, Valid: true}
	}
	nextRun := CalcNextRun(req.Frequency, sendHour, req.DayOfWeek)

	row, err := s.q.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name:       req.Name,
		Frequency:  req.Frequency,
		Recipients: req.Recipients,
		Filters:    filtersJSON,
		SendHour:   sendHour,
		DayOfWeek:  dow,
		IsActive:   active,
		NextRunAt:  nextRun,
		CreatedBy:  createdBy,
	})
	if err != nil {
		return nil, err
	}
	return scheduleFromRow(row), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Schedule, error) {
	row, err := s.q.GetReportSchedule(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return scheduleFromRow(row), nil
}

func (s *Service) List(ctx context.Context) ([]*Schedule, error) {
	rows, err := s.q.ListReportSchedules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*Schedule, len(rows))
	for i, r := range rows {
		out[i] = scheduleFromRow(r)
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateScheduleRequest) (*Schedule, error) {
	if _, err := s.q.GetReportSchedule(ctx, id); err != nil {
		return nil, ErrNotFound
	}
	if err := validateReq(req); err != nil {
		return nil, err
	}
	sendHour := int32(7)
	if req.SendHour != nil {
		sendHour = *req.SendHour
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	filtersJSON, _ := json.Marshal(req.Filters)
	var dow sql.NullInt32
	if req.Frequency == FreqWeekly && req.DayOfWeek != nil {
		dow = sql.NullInt32{Int32: *req.DayOfWeek, Valid: true}
	}
	nextRun := CalcNextRun(req.Frequency, sendHour, req.DayOfWeek)

	row, err := s.q.UpdateReportSchedule(ctx, db.UpdateReportScheduleParams{
		ID:         id,
		Name:       req.Name,
		Frequency:  req.Frequency,
		Recipients: req.Recipients,
		Filters:    filtersJSON,
		SendHour:   sendHour,
		DayOfWeek:  dow,
		IsActive:   active,
		NextRunAt:  nextRun,
	})
	if err != nil {
		return nil, err
	}
	return scheduleFromRow(row), nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.q.GetReportSchedule(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.q.DeleteReportSchedule(ctx, id)
}

// SendNow immediately sends the report for the given schedule (for testing).
func (s *Service) SendNow(ctx context.Context, id uuid.UUID) error {
	row, err := s.q.GetReportSchedule(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	return s.sendReport(ctx, row)
}

// ─── Delivery ──────────────────────────────────────────────────────────────────

func (s *Service) sendReport(ctx context.Context, row db.ReportSchedule) error {
	stats, err := s.q.GetDashboardStats(ctx)
	if err != nil {
		return err
	}

	var since time.Time
	var periodDesc string
	switch row.Frequency {
	case FreqWeekly:
		since = time.Now().UTC().AddDate(0, 0, -7)
		periodDesc = "last 7 days"
	default:
		since = time.Now().UTC().Add(-24 * time.Hour)
		periodDesc = "last 24 hours"
	}
	newCases, _ := s.q.CountNewCasesInPeriod(ctx, since)

	sched := scheduleFromRow(row)
	subject, plainText, htmlBody := ComposeEmail(ReportData{
		Schedule:    sched,
		Stats:       stats,
		NewCases:    newCases,
		PeriodDesc:  periodDesc,
		GeneratedAt: time.Now().UTC(),
	})

	return s.mailer.Send(mail.Message{
		To:      row.Recipients,
		Subject: subject,
		Body:    plainText,
		HTML:    htmlBody,
	})
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func validateReq(req CreateScheduleRequest) error {
	if req.Frequency != FreqDaily && req.Frequency != FreqWeekly {
		return ErrInvalidFrequency
	}
	if len(req.Recipients) == 0 {
		return ErrNoRecipients
	}
	if req.Frequency == FreqWeekly {
		if req.DayOfWeek == nil || *req.DayOfWeek < 0 || *req.DayOfWeek > 6 {
			return ErrInvalidDayOfWeek
		}
	}
	return nil
}

// CalcNextRun returns the next UTC wall-clock time the schedule should fire.
func CalcNextRun(frequency string, sendHour int32, dow *int32) time.Time {
	now := time.Now().UTC()
	target := time.Date(now.Year(), now.Month(), now.Day(), int(sendHour), 0, 0, 0, time.UTC)
	if target.Before(now) {
		target = target.Add(24 * time.Hour)
	}
	if frequency == FreqWeekly && dow != nil {
		for int(target.Weekday()) != int(*dow) {
			target = target.Add(24 * time.Hour)
		}
	}
	return target
}

// jsonUnmarshal is a nil-safe json.Unmarshal.
func jsonUnmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// EmailsFromString splits a comma-separated email list.
func EmailsFromString(s string) []string {
	var out []string
	for _, e := range strings.Split(s, ",") {
		if e := strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}
