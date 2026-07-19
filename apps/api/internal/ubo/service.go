// Package ubo implements the EU Beneficial Ownership (UBO) verification workflow.
//
// EU AMLR 2024, Art. 42–55 mandates that obliged entities identify and verify
// the beneficial owners (≥ 25% control threshold) of all legal entity customers
// before onboarding and during periodic review.
//
// OpsOwl enforces a case closure gate: a case involving a legal entity subject
// cannot be closed if UBO verification is still Pending or Discrepant.
package ubo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrCaseClosed = errors.New("case is closed — no new UBO records may be added")

// VerificationStatus reflects the outcome of the UBO identification process.
type VerificationStatus string

const (
	StatusPending      VerificationStatus = "pending"       // captured, not yet verified
	StatusVerified     VerificationStatus = "verified"      // confirmed against registry or documents
	StatusDiscrepant   VerificationStatus = "discrepant"    // mismatch between declared and registry
	StatusNotApplicable VerificationStatus = "n/a"          // individual subject, not a legal entity
)

// NationalRegistry maps country codes to their official company/UBO registry.
// Used to generate a direct deep-link for the analyst to verify the UBO manually.
var NationalRegistry = map[string]string{
	// India
	"in": "https://www.mca.gov.in/content/mca/global/en/home.html", // MCA21 — Ministry of Corporate Affairs
	// UAE
	"ae": "https://www.uaeregistrar.gov.ae",                        // UAE Registrar of Companies
	// United Kingdom
	"gb": "https://find-and-update.company-information.service.gov.uk", // Companies House
	// Singapore
	"sg": "https://www.bizfile.gov.sg",                             // ACRA BizFile+
	// Australia
	"au": "https://connectonline.asic.gov.au",                      // ASIC Connect
	// EU member states
	"gr": "https://efiling.gsis.gr",          // GEMI — Γενικό Εμπορικό Μητρώο
	"de": "https://www.handelsregister.de",
	"fr": "https://data.inpi.fr",
	"nl": "https://www.kvk.nl",
	"it": "https://visura.it",
	"es": "https://www.registradores.org",
	"pl": "https://ekrs.ms.gov.pl",
	"be": "https://kbo.economie.fgov.be",
	"at": "https://www.firmenbuch.at",
	"se": "https://www.bolagsverket.se",
	"dk": "https://datacvr.virk.dk",
	"fi": "https://www.ytj.fi",
	"ie": "https://search.cro.ie",
	"pt": "https://eportugal.gov.pt/servicos/aceder-a-informacao-de-empresas",
	"ro": "https://www.onrc.ro",
	"bg": "https://www.brra.bg",
	"hr": "https://sudreg.pravosudje.hr",
	"cy": "https://www.drcor.mcit.gov.cy",
	"mt": "https://registry.mbr.mt",
	"lu": "https://www.lbr.lu",
	"sk": "https://orsr.sk",
	"si": "https://www.ajpes.si",
	"lt": "https://www.registrucentras.lt",
	"lv": "https://lursoft.lv",
	"ee": "https://www.e-krediidiinfo.ee",
	"cz": "https://or.justice.cz",
	"hu": "https://e-cegjegyzek.hu",
}

// UBORecord is a single beneficial owner of a legal entity subject.
type UBORecord struct {
	ID               string             `json:"id"`
	CaseID           string             `json:"case_id"`
	EntityName       string             `json:"entity_name"`
	OwnerName        string             `json:"owner_name"`
	OwnerNationality string             `json:"owner_nationality"`  // ISO 3166-1 alpha-2
	OwnerIDType      string             `json:"owner_id_type"`
	OwnerIDNumber    string             `json:"owner_id_number"`
	OwnerBirthDate   string             `json:"owner_birth_date"`   // YYYY-MM-DD
	OwnershipPct     float64            `json:"ownership_pct"`
	ControlType      string             `json:"control_type"`       // "share", "voting_right", "other_control"
	Status           VerificationStatus `json:"status"`
	RegistryURL      string             `json:"registry_url,omitempty"`
	CreatedBy        string             `json:"created_by,omitempty"`  // user ID who added the record
	VerifiedBy       string             `json:"verified_by,omitempty"` // user ID who verified
	VerifiedAt       *time.Time         `json:"verified_at,omitempty"`
	DiscrepancyNote  string             `json:"discrepancy_note,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// Service manages UBO records and enforces the closure gate.
type Service struct {
	db                *sql.DB
	repo              *Repository
	thresholdPct      float64
	deploymentCountry string
}

// jurisdictionLabel returns the regulation citation for the deployment jurisdiction.
func jurisdictionLabel(country string) string {
	switch country {
	case "in":
		return "Companies Act 2013 Section 90 / PMLA 2002 / RBI KYC MD 2023"
	case "ae":
		return "UAE Federal Decree-Law No. 20 of 2018"
	case "gb":
		return "UK MLR 2017 Reg. 5"
	default:
		return "EU AMLR 2024 Art. 42"
	}
}

// NewService creates a UBO service.
// thresholdPct is the beneficial ownership percentage that triggers UBO verification
// (default: 25%). deploymentCountry determines which NationalRegistry URL is pre-filled.
func NewService(db *sql.DB, thresholdPct float64, deploymentCountry string) *Service {
	return &Service{
		db:                db,
		repo:              NewRepository(db),
		thresholdPct:      thresholdPct,
		deploymentCountry: deploymentCountry,
	}
}

// Create adds a new UBO record for a case and pre-fills the registry URL
// for the deployment's member state.
func (s *Service) Create(ctx context.Context, rec *UBORecord) (*UBORecord, error) {
	var caseStatus string
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM cases WHERE id = $1`, rec.CaseID).Scan(&caseStatus); err == nil && caseStatus == "closed" {
		return nil, ErrCaseClosed
	}

	rec.ID = uuid.NewString()
	rec.Status = StatusPending
	rec.CreatedAt = time.Now().UTC()
	rec.UpdatedAt = rec.CreatedAt

	if rec.RegistryURL == "" {
		rec.RegistryURL = NationalRegistry[s.deploymentCountry]
	}

	var createdBy *string
	if rec.CreatedBy != "" {
		createdBy = &rec.CreatedBy
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ubo_records
			(id, case_id, entity_name, owner_name, owner_nationality,
			 owner_id_type, owner_id_number, owner_birth_date,
			 ownership_pct, control_type, status, registry_url,
			 created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		rec.ID, rec.CaseID, rec.EntityName, rec.OwnerName, rec.OwnerNationality,
		rec.OwnerIDType, rec.OwnerIDNumber, rec.OwnerBirthDate,
		rec.OwnershipPct, rec.ControlType, rec.Status, rec.RegistryURL,
		createdBy, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ubo: create record: %w", err)
	}
	return rec, nil
}

// CaseMayClose returns nil if all UBO records for the case are Verified or N/A.
// Returns an error (which the closure handler surfaces as a 409) if any are
// still Pending or Discrepant — enforcing the AMLR 2024 Art. 42 closure gate.
func (s *Service) CaseMayClose(ctx context.Context, caseID string) error {
	var pendingCount int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM ubo_records
		WHERE case_id = $1 AND status IN ('pending', 'discrepant')`,
		caseID,
	).Scan(&pendingCount)
	if err != nil {
		return fmt.Errorf("ubo: closure check query: %w", err)
	}
	if pendingCount > 0 {
		return fmt.Errorf(
			"ubo: %d beneficial owner record(s) require verification before closure (%s)",
			pendingCount, jurisdictionLabel(s.deploymentCountry),
		)
	}
	return nil
}

// AboveThreshold returns true if the ownership percentage meets or exceeds
// the deployment's UBO threshold (typically 25% for EU).
func (s *Service) AboveThreshold(pct float64) bool {
	return pct >= s.thresholdPct
}
