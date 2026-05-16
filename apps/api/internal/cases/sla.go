package cases

import (
	"time"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var slaDurations = map[db.CasePriority]time.Duration{
	db.CasePriorityCritical: 4 * time.Hour,
	db.CasePriorityHigh:     24 * time.Hour,
	db.CasePriorityMedium:   72 * time.Hour,
	db.CasePriorityLow:      7 * 24 * time.Hour,
}

func slaBreachAt(priority db.CasePriority, from time.Time) time.Time {
	d, ok := slaDurations[priority]
	if !ok {
		d = 72 * time.Hour
	}
	return from.Add(d)
}
