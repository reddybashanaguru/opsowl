package sanctions

import "time"

// ListSource identifies a sanctions list provider.
type ListSource string

const (
	SourceEUConsolidated ListSource = "eu_consolidated"
	SourceUNConsolidated ListSource = "un_consolidated"
	SourceOFACSDN        ListSource = "ofac_sdn"
)

// SanctionsList is the metadata record for a loaded list.
type SanctionsList struct {
	ID         string     `json:"id"`
	Source     ListSource `json:"source"`
	LoadedAt   time.Time  `json:"loaded_at"`
	EntryCount int        `json:"entry_count"`
	Checksum   string     `json:"checksum"`
}

// Entry is a single person/entity on a sanctions list.
type Entry struct {
	ID          string     `json:"id"`
	ListSource  ListSource `json:"list_source"`
	ExternalID  string     `json:"external_id"`
	EntityType  string     `json:"entity_type"`
	PrimaryName string     `json:"primary_name"`
	Aliases     []string   `json:"aliases"`
	Nationality string     `json:"nationality"`
	BirthDate   string     `json:"birth_date"`
	ListedOn    *time.Time `json:"listed_on,omitempty"`
	Programs    []string   `json:"programs"`
	Remarks     string     `json:"remarks,omitempty"`
}

// Hit is a screening result: a subject matched against a sanctions entry.
type Hit struct {
	ID          string     `json:"id"`
	SubjectID   string     `json:"subject_id"`
	EntryID     string     `json:"entry_id"`
	Entry       *Entry     `json:"entry,omitempty"`
	MatchType   string     `json:"match_type"`
	MatchScore  float64    `json:"match_score"`
	MatchedName string     `json:"matched_name"`
	ScreenedAt  time.Time  `json:"screened_at"`
	ScreenedBy  *string    `json:"screened_by,omitempty"`
	Status      string     `json:"status"`
	ReviewedBy  *string    `json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

// HitStatus values
const (
	HitStatusOpen         = "open"
	HitStatusFalsePos     = "false_positive"
	HitStatusConfirmed    = "confirmed"
	HitStatusEscalated    = "escalated"
)

// ScreenResult summarises a screening run for one subject.
type ScreenResult struct {
	SubjectID   string    `json:"subject_id"`
	ScreenedAt  time.Time `json:"screened_at"`
	HitCount    int       `json:"hit_count"`
	IsSanctioned bool     `json:"is_sanctioned"`
	Hits        []Hit     `json:"hits"`
}
