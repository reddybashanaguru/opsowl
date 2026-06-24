package subjects

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Create ────────────────────────────────────────────────────────────────────

const createSubjectSQL = `
INSERT INTO subjects (
  org_id, canonical_name, entity_type, id_type, id_number,
  secondary_ids, nationality, date_of_birth, gender,
  addresses, phones, emails,
  is_pep, is_sanctioned, is_watchlisted,
  kyc_tier, notes, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
RETURNING id, created_at, updated_at`

func (r *Repository) Create(ctx context.Context, orgID, createdBy uuid.UUID, req CreateSubjectRequest) (*SubjectResponse, error) {
	secIDs, _ := json.Marshal(req.SecondaryIDs)
	addrs, _ := json.Marshal(req.Addresses)
	entityType := req.EntityType
	if entityType == "" {
		entityType = "individual"
	}
	if req.Phones == nil {
		req.Phones = []string{}
	}
	if req.Emails == nil {
		req.Emails = []string{}
	}

	var dob *string
	if req.DateOfBirth != nil && *req.DateOfBirth != "" {
		dob = req.DateOfBirth
	}

	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, createSubjectSQL,
		orgID, req.CanonicalName, entityType, req.IDType, req.IDNumber,
		secIDs, req.Nationality, dob, req.Gender,
		addrs, pq.Array(req.Phones), pq.Array(req.Emails),
		req.IsPEP, req.IsSanctioned, req.IsWatchlisted,
		req.KYCTier, req.Notes, createdBy,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "idx_subjects_identity") {
			return nil, ErrDuplicateID
		}
		return nil, err
	}

	return r.GetByID(ctx, id, orgID)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

const getSubjectSQL = `
SELECT id, org_id, canonical_name, entity_type, id_type, id_number,
       secondary_ids, nationality, date_of_birth, gender,
       addresses, phones, emails,
       risk_score, risk_level, risk_factors,
       is_pep, is_sanctioned, is_watchlisted,
       kyc_tier, kyc_last_reviewed_at,
       total_cases, open_cases, total_str_filings,
       notes, first_seen_at, last_seen_at, created_at, updated_at
FROM subjects
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`

func (r *Repository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*SubjectResponse, error) {
	row := r.db.QueryRowContext(ctx, getSubjectSQL, id, orgID)
	return scanSubject(row)
}

func (r *Repository) GetByIdentity(ctx context.Context, orgID uuid.UUID, idType, idNumber string) (*SubjectResponse, error) {
	if idType == "" || idNumber == "" {
		return nil, ErrNotFound
	}
	const q = `SELECT id, org_id, canonical_name, entity_type, id_type, id_number,
       secondary_ids, nationality, date_of_birth, gender,
       addresses, phones, emails,
       risk_score, risk_level, risk_factors,
       is_pep, is_sanctioned, is_watchlisted,
       kyc_tier, kyc_last_reviewed_at,
       total_cases, open_cases, total_str_filings,
       notes, first_seen_at, last_seen_at, created_at, updated_at
FROM subjects WHERE org_id=$1 AND id_type=$2 AND id_number=$3 AND deleted_at IS NULL LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, orgID, idType, idNumber)
	sub, err := scanSubject(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return sub, err
}

// ── List ──────────────────────────────────────────────────────────────────────

func (r *Repository) List(ctx context.Context, orgID uuid.UUID, f ListFilters) ([]SubjectResponse, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 25
	}

	args := []any{orgID}
	where := []string{"org_id = $1", "deleted_at IS NULL"}
	n := 2

	if f.Search != "" {
		where = append(where, "canonical_name ILIKE $"+itoa(n))
		args = append(args, "%"+f.Search+"%")
		n++
	}
	if f.EntityType != "" {
		where = append(where, "entity_type = $"+itoa(n))
		args = append(args, f.EntityType)
		n++
	}
	if f.RiskLevel != "" {
		where = append(where, "risk_level = $"+itoa(n))
		args = append(args, f.RiskLevel)
		n++
	}
	if f.IsPEP != nil {
		where = append(where, "is_pep = $"+itoa(n))
		args = append(args, *f.IsPEP)
		n++
	}
	if f.IsSanctioned != nil {
		where = append(where, "is_sanctioned = $"+itoa(n))
		args = append(args, *f.IsSanctioned)
		n++
	}
	if f.Watchlisted != nil {
		where = append(where, "is_watchlisted = $"+itoa(n))
		args = append(args, *f.Watchlisted)
		n++
	}

	cond := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM subjects WHERE "+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.QueryContext(ctx, `
SELECT id, org_id, canonical_name, entity_type, id_type, id_number,
       secondary_ids, nationality, date_of_birth, gender,
       addresses, phones, emails,
       risk_score, risk_level, risk_factors,
       is_pep, is_sanctioned, is_watchlisted,
       kyc_tier, kyc_last_reviewed_at,
       total_cases, open_cases, total_str_filings,
       notes, first_seen_at, last_seen_at, created_at, updated_at
FROM subjects WHERE `+cond+` ORDER BY last_seen_at DESC LIMIT $`+itoa(n)+` OFFSET $`+itoa(n+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []SubjectResponse
	for rows.Next() {
		s, err := scanSubjectRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *s)
	}
	if out == nil {
		out = []SubjectResponse{}
	}
	return out, total, rows.Err()
}

// ── Update ────────────────────────────────────────────────────────────────────

func (r *Repository) Update(ctx context.Context, id, orgID uuid.UUID, req UpdateSubjectRequest) (*SubjectResponse, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1

	set := func(col string, val any) {
		sets = append(sets, col+" = $"+itoa(n))
		args = append(args, val)
		n++
	}

	if req.CanonicalName != nil {
		set("canonical_name", *req.CanonicalName)
	}
	if req.EntityType != nil {
		set("entity_type", *req.EntityType)
	}
	if req.IDType != nil {
		set("id_type", *req.IDType)
	}
	if req.IDNumber != nil {
		set("id_number", *req.IDNumber)
	}
	if req.SecondaryIDs != nil {
		b, _ := json.Marshal(req.SecondaryIDs)
		set("secondary_ids", b)
	}
	if req.Nationality != nil {
		set("nationality", *req.Nationality)
	}
	if req.Gender != nil {
		set("gender", *req.Gender)
	}
	if req.DateOfBirth != nil {
		set("date_of_birth", *req.DateOfBirth)
	}
	if req.Addresses != nil {
		b, _ := json.Marshal(req.Addresses)
		set("addresses", b)
	}
	if req.Phones != nil {
		set("phones", pq.Array(req.Phones))
	}
	if req.Emails != nil {
		set("emails", pq.Array(req.Emails))
	}
	if req.IsPEP != nil {
		set("is_pep", *req.IsPEP)
	}
	if req.IsSanctioned != nil {
		set("is_sanctioned", *req.IsSanctioned)
	}
	if req.IsWatchlisted != nil {
		set("is_watchlisted", *req.IsWatchlisted)
	}
	if req.KYCTier != nil {
		set("kyc_tier", *req.KYCTier)
	}
	if req.Notes != nil {
		set("notes", *req.Notes)
	}
	if req.RiskScore != nil {
		set("risk_score", *req.RiskScore)
	}
	if req.RiskLevel != nil {
		set("risk_level", *req.RiskLevel)
	}
	if req.RiskFactors != nil {
		set("risk_factors", pq.Array(req.RiskFactors))
	}

	args = append(args, id, orgID)
	q := "UPDATE subjects SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + itoa(n) + " AND org_id = $" + itoa(n+1) + " AND deleted_at IS NULL"

	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id, orgID)
}

// ── Delete (soft) ─────────────────────────────────────────────────────────────

func (r *Repository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		"UPDATE subjects SET deleted_at = NOW() WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL",
		id, orgID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Case ↔ Subject linking ────────────────────────────────────────────────────

func (r *Repository) LinkToCase(ctx context.Context, caseID, subjectID, linkedBy uuid.UUID, role string) error {
	if role == "" {
		role = "primary"
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO case_subjects (case_id, subject_id, role, linked_by)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (case_id, subject_id) DO NOTHING`,
		caseID, subjectID, role, linkedBy)
	if err != nil {
		return err
	}
	// update last_seen_at and stats
	_, _ = r.db.ExecContext(ctx,
		`UPDATE subjects SET last_seen_at = NOW(),
            total_cases = (SELECT COUNT(*) FROM case_subjects WHERE subject_id = $1),
            open_cases  = (SELECT COUNT(*) FROM case_subjects cs
                           JOIN cases c ON c.id = cs.case_id
                           WHERE cs.subject_id = $1 AND c.status NOT IN ('closed') AND c.deleted_at IS NULL)
         WHERE id = $1`, subjectID)
	return nil
}

func (r *Repository) UnlinkFromCase(ctx context.Context, caseID, subjectID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		"DELETE FROM case_subjects WHERE case_id = $1 AND subject_id = $2",
		caseID, subjectID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotLinked
	}
	_, _ = r.db.ExecContext(ctx,
		`UPDATE subjects SET
            total_cases = (SELECT COUNT(*) FROM case_subjects WHERE subject_id = $1),
            open_cases  = (SELECT COUNT(*) FROM case_subjects cs
                           JOIN cases c ON c.id = cs.case_id
                           WHERE cs.subject_id = $1 AND c.status NOT IN ('closed') AND c.deleted_at IS NULL)
         WHERE id = $1`, subjectID)
	return nil
}

func (r *Repository) ListForCase(ctx context.Context, caseID uuid.UUID) ([]CaseSubjectRef, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT cs.subject_id, s.canonical_name, s.id_type, s.id_number, s.entity_type,
       s.risk_level, s.risk_score, s.is_pep, s.is_sanctioned, cs.role, cs.linked_at
FROM case_subjects cs
JOIN subjects s ON s.id = cs.subject_id
WHERE cs.case_id = $1 AND s.deleted_at IS NULL
ORDER BY cs.linked_at`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CaseSubjectRef
	for rows.Next() {
		var ref CaseSubjectRef
		if err := rows.Scan(&ref.SubjectID, &ref.CanonicalName, &ref.IDType, &ref.IDNumber,
			&ref.EntityType, &ref.RiskLevel, &ref.RiskScore, &ref.IsPEP, &ref.IsSanctioned,
			&ref.Role, &ref.LinkedAt); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	if out == nil {
		out = []CaseSubjectRef{}
	}
	return out, rows.Err()
}

func (r *Repository) ListCasesForSubject(ctx context.Context, subjectID uuid.UUID) ([]SubjectCaseRef, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT c.id, c.case_number, c.title, c.status, c.priority, cs.role, cs.linked_at, c.created_at
FROM case_subjects cs
JOIN cases c ON c.id = cs.case_id
WHERE cs.subject_id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at DESC`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubjectCaseRef
	for rows.Next() {
		var ref SubjectCaseRef
		if err := rows.Scan(&ref.CaseID, &ref.CaseNumber, &ref.Title, &ref.Status,
			&ref.Priority, &ref.Role, &ref.LinkedAt, &ref.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	if out == nil {
		out = []SubjectCaseRef{}
	}
	return out, rows.Err()
}

// ── Risk history ──────────────────────────────────────────────────────────────

func (r *Repository) AppendRiskHistory(ctx context.Context, subjectID uuid.UUID, score int, level, reason, changedBy string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO subject_risk_history (subject_id, risk_score, risk_level, reason, changed_by) VALUES ($1,$2,$3,$4,$5)",
		subjectID, score, level, reason, changedBy)
	return err
}

func (r *Repository) RiskHistory(ctx context.Context, subjectID uuid.UUID) ([]RiskHistoryEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, risk_score, risk_level, reason, changed_by, created_at FROM subject_risk_history WHERE subject_id = $1 ORDER BY created_at DESC LIMIT 50",
		subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RiskHistoryEntry
	for rows.Next() {
		var e RiskHistoryEntry
		if err := rows.Scan(&e.ID, &e.RiskScore, &e.RiskLevel, &e.Reason, &e.ChangedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if out == nil {
		out = []RiskHistoryEntry{}
	}
	return out, rows.Err()
}

// ── STR filings ───────────────────────────────────────────────────────────────

func (r *Repository) ListFilingsForSubject(ctx context.Context, subjectID, orgID uuid.UUID) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT sf.id, sf.report_type, sf.status, sf.suspicion_basis, sf.created_at, c.id AS case_id, c.case_number
FROM str_filings sf
JOIN cases c ON c.id = sf.case_id
JOIN case_subjects cs ON cs.case_id = c.id
WHERE cs.subject_id = $1 AND c.organization_id = $2 AND c.deleted_at IS NULL
ORDER BY sf.created_at DESC`, subjectID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, caseID uuid.UUID
		var reportType, status, suspicionBasis, caseNumber string
		var createdAt time.Time
		if err := rows.Scan(&id, &reportType, &status, &suspicionBasis, &createdAt, &caseID, &caseNumber); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "report_type": reportType, "status": status,
			"suspicion_basis": suspicionBasis, "created_at": createdAt,
			"case_id": caseID, "case_number": caseNumber,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, rows.Err()
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanSubject(s scanner) (*SubjectResponse, error) {
	return scanSubjectRow(s)
}

func scanSubjectRow(s scanner) (*SubjectResponse, error) {
	var sub SubjectResponse
	var secIDsRaw, addrsRaw []byte
	var dob sql.NullTime
	var kycReviewed sql.NullTime
	var phones, emails, riskFactors pq.StringArray

	var orgID uuid.UUID
	if err := s.Scan(
		&sub.ID, &orgID, &sub.CanonicalName, &sub.EntityType, &sub.IDType, &sub.IDNumber,
		&secIDsRaw, &sub.Nationality, &dob, &sub.Gender,
		&addrsRaw, &phones, &emails,
		&sub.RiskScore, &sub.RiskLevel, &riskFactors,
		&sub.IsPEP, &sub.IsSanctioned, &sub.IsWatchlisted,
		&sub.KYCTier, &kycReviewed,
		&sub.TotalCases, &sub.OpenCases, &sub.TotalSTRFilings,
		&sub.Notes, &sub.FirstSeenAt, &sub.LastSeenAt, &sub.CreatedAt, &sub.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if dob.Valid {
		s := dob.Time.Format("2006-01-02")
		sub.DateOfBirth = &s
	}
	if kycReviewed.Valid {
		sub.KYCLastReviewed = &kycReviewed.Time
	}

	_ = json.Unmarshal(secIDsRaw, &sub.SecondaryIDs)
	_ = json.Unmarshal(addrsRaw, &sub.Addresses)
	if sub.SecondaryIDs == nil {
		sub.SecondaryIDs = []SecondaryID{}
	}
	if sub.Addresses == nil {
		sub.Addresses = []Address{}
	}

	sub.Phones = []string(phones)
	sub.Emails = []string(emails)
	sub.RiskFactors = []string(riskFactors)
	if sub.Phones == nil {
		sub.Phones = []string{}
	}
	if sub.Emails == nil {
		sub.Emails = []string{}
	}
	if sub.RiskFactors == nil {
		sub.RiskFactors = []string{}
	}

	return &sub, nil
}

func itoa(n int) string {
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}
	return itoa(n/10) + string(digits[n%10])
}

// ── Fuzzy Entity Resolution ───────────────────────────────────────────────────

type SimilarSubject struct {
	ID            string   `json:"id"`
	CanonicalName string   `json:"canonical_name"`
	EntityType    string   `json:"entity_type"`
	IDNumber      string   `json:"id_number"`
	Phones        []string `json:"phones"`
	RiskLevel     string   `json:"risk_level"`
	RiskScore     int      `json:"risk_score"`
	IsPEP         bool     `json:"is_pep"`
	IsSanctioned  bool     `json:"is_sanctioned"`
	DateOfBirth   *string  `json:"date_of_birth,omitempty"`
	// Scoring breakdown
	NameSimilarity float64 `json:"name_similarity"`
	PhoneMatch     bool    `json:"phone_match"`
	DOBMatch       bool    `json:"dob_match"`
	TotalScore     float64 `json:"total_score"`
	MatchReasons   []string `json:"match_reasons"`
}

// FindSimilar returns subjects similar to the given one, ranked by composite score.
// Uses PostgreSQL pg_trgm similarity on canonical_name and phone array overlap.
func (r *Repository) FindSimilar(ctx context.Context, subjectID uuid.UUID, threshold float64, limit int) ([]SimilarSubject, error) {
	// First fetch the source subject's name, phones, DOB for comparison
	var name, dob sql.NullString
	var phones pq.StringArray
	err := r.db.QueryRowContext(ctx,
		`SELECT canonical_name, phones, date_of_birth::text FROM subjects WHERE id = $1 AND deleted_at IS NULL`,
		subjectID,
	).Scan(&name, &phones, &dob)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 50 {
		limit = 10
	}
	if threshold <= 0 {
		threshold = 0.25
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			s.id,
			s.canonical_name,
			s.entity_type,
			s.id_number,
			s.phones,
			s.risk_level,
			s.risk_score,
			s.is_pep,
			s.is_sanctioned,
			s.date_of_birth::text,
			similarity(s.canonical_name, $2)                                       AS name_sim,
			(s.phones && $3::text[] AND cardinality($3::text[]) > 0)              AS phone_match,
			($4::text IS NOT NULL AND s.date_of_birth::text = $4::text)           AS dob_match,
			similarity(s.canonical_name, $2)
			  + CASE WHEN s.phones && $3::text[] AND cardinality($3::text[]) > 0 THEN 0.35 ELSE 0 END
			  + CASE WHEN $4::text IS NOT NULL AND s.date_of_birth::text = $4::text THEN 0.25 ELSE 0 END
			  AS total_score
		FROM subjects s
		WHERE
			s.id          != $1
			AND s.deleted_at IS NULL
			AND (
				similarity(s.canonical_name, $2) >= $5
				OR (s.phones && $3::text[] AND cardinality($3::text[]) > 0)
			)
		ORDER BY total_score DESC
		LIMIT $6`,
		subjectID, name.String, phones, dob, threshold, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []SimilarSubject{}
	for rows.Next() {
		var s SimilarSubject
		var dobStr sql.NullString
		var ph pq.StringArray
		if err := rows.Scan(
			&s.ID, &s.CanonicalName, &s.EntityType, &s.IDNumber,
			&ph, &s.RiskLevel, &s.RiskScore, &s.IsPEP, &s.IsSanctioned,
			&dobStr, &s.NameSimilarity, &s.PhoneMatch, &s.DOBMatch, &s.TotalScore,
		); err != nil {
			return nil, err
		}
		s.Phones = []string(ph)
		if s.Phones == nil {
			s.Phones = []string{}
		}
		if dobStr.Valid {
			s.DateOfBirth = &dobStr.String
		}
		// Build human-readable match reasons
		s.MatchReasons = []string{}
		if s.NameSimilarity >= 0.8 {
			s.MatchReasons = append(s.MatchReasons, "Near-identical name")
		} else if s.NameSimilarity >= 0.5 {
			s.MatchReasons = append(s.MatchReasons, "Similar name")
		} else if s.NameSimilarity >= 0.25 {
			s.MatchReasons = append(s.MatchReasons, "Partial name match")
		}
		if s.PhoneMatch {
			s.MatchReasons = append(s.MatchReasons, "Shared phone number")
		}
		if s.DOBMatch {
			s.MatchReasons = append(s.MatchReasons, "Same date of birth")
		}
		results = append(results, s)
	}
	return results, nil
}

// ConfirmMatch creates a SAME_ENTITY relationship between two subjects via entity_relationships.
func (r *Repository) ConfirmMatch(ctx context.Context, fromID, toID, userID uuid.UUID, notes string, relType string) error {
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO entity_relationships
		  (org_id, from_subject_id, to_subject_id, relationship_type, strength, source, metadata, created_by)
		VALUES ($1, $2, $3, $4, 1.0, 'fuzzy_resolution', '{}', $5)
		ON CONFLICT (org_id, from_subject_id, to_subject_id, relationship_type) DO NOTHING`,
		orgID, fromID, toID, relType, userID,
	)
	return err
}
