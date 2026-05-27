package scheduled_reports

import (
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrNotFound           = errors.New("report schedule not found")
	ErrNoRecipients       = errors.New("at least one recipient email is required")
	ErrInvalidFrequency   = errors.New("frequency must be 'daily' or 'weekly'")
	ErrInvalidDayOfWeek   = errors.New("day_of_week must be 0–6 for weekly schedules")
)

const (
	FreqDaily  = "daily"
	FreqWeekly = "weekly"
)

type Filters struct {
	Status     *string `json:"status,omitempty"`
	Priority   *string `json:"priority,omitempty"`
	SLABreached *bool  `json:"sla_breached,omitempty"`
}

type Schedule struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Frequency  string     `json:"frequency"`
	Recipients []string   `json:"recipients"`
	Filters    Filters    `json:"filters"`
	SendHour   int32      `json:"send_hour"`
	DayOfWeek  *int32     `json:"day_of_week,omitempty"`
	IsActive   bool       `json:"is_active"`
	LastSentAt *time.Time `json:"last_sent_at,omitempty"`
	NextRunAt  time.Time  `json:"next_run_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateScheduleRequest struct {
	Name       string   `json:"name"`
	Frequency  string   `json:"frequency"`
	Recipients []string `json:"recipients"`
	Filters    Filters  `json:"filters"`
	SendHour   *int32   `json:"send_hour"`
	DayOfWeek  *int32   `json:"day_of_week"`
	IsActive   *bool    `json:"is_active"`
}

type UpdateScheduleRequest = CreateScheduleRequest

func scheduleFromRow(r db.ReportSchedule) *Schedule {
	s := &Schedule{
		ID:         r.ID,
		Name:       r.Name,
		Frequency:  r.Frequency,
		Recipients: r.Recipients,
		SendHour:   r.SendHour,
		IsActive:   r.IsActive,
		NextRunAt:  r.NextRunAt,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
	if r.DayOfWeek.Valid {
		v := r.DayOfWeek.Int32
		s.DayOfWeek = &v
	}
	if r.LastSentAt.Valid {
		t := r.LastSentAt.Time
		s.LastSentAt = &t
	}
	// Decode filters JSON
	_ = jsonUnmarshal(r.Filters, &s.Filters)
	return s
}
