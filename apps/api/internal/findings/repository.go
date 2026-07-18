package findings

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Internal row types (not exported) ────────────────────────────────────────

type findingRow struct {
	ID                uuid.UUID
	CaseID            uuid.UUID
	Title             string
	Description       string
	Category          string
	Severity          string
	Status            string
	NarrativeWeight   string
	ConfidenceScore   sql.NullString
	AIGenerated       bool
	SourceAILogID     uuid.NullUUID
	EvidenceCount     int64
	CreatedBy         uuid.UUID
	CreatedByName     string
	ReviewedBy        uuid.NullUUID
	ReviewedByName    sql.NullString
	ApprovedBy        uuid.NullUUID
	ApprovedByName    sql.NullString
	ReviewedAt        sql.NullTime
	ApprovedAt        sql.NullTime
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type evidenceLinkRow struct {
	ID            uuid.UUID
	FindingID     uuid.UUID
	EvidenceType  string
	EvidenceID    uuid.UUID
	Citation      string
	PageNumber    sql.NullInt32
	ChunkID       uuid.NullUUID
	RelevanceNote string
	LinkedBy      uuid.UUID
	LinkedAt      time.Time
}

// ── Finding CRUD ──────────────────────────────────────────────────────────────

func (r *Repository) Create(ctx context.Context, caseID, createdBy uuid.UUID, req CreateFindingRequest) (findingRow, error) {
	category := req.Category
	if category == "" {
		category = "other"
	}
	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}
	weight := req.NarrativeWeight
	if weight == "" {
		weight = "supporting"
	}

	var sourceLogID uuid.NullUUID
	if req.SourceAILogID != nil {
		if id, err := uuid.Parse(*req.SourceAILogID); err == nil {
			sourceLogID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO case_findings
			(case_id, title, description, category, severity, status, narrative_weight,
			 ai_generated, source_ai_log_id, created_by)
		VALUES ($1,$2,$3,$4,$5,'draft',$6,$7,$8,$9)
		RETURNING id, created_at, updated_at`,
		caseID, req.Title, req.Description,
		category, severity, weight,
		req.AIGenerated, sourceLogID, createdBy,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return findingRow{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (findingRow, error) {
	row := findingRow{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			f.id, f.case_id, f.title, f.description,
			f.category::TEXT, f.severity::TEXT, f.status::TEXT, f.narrative_weight::TEXT,
			f.confidence_score::TEXT, f.ai_generated, f.source_ai_log_id,
			COALESCE((SELECT COUNT(*) FROM finding_evidence_links WHERE finding_id = f.id), 0) AS evidence_count,
			f.created_by, COALESCE(cu.full_name,'') AS created_by_name,
			f.reviewed_by, ru.full_name AS reviewed_by_name,
			f.approved_by, au.full_name AS approved_by_name,
			f.reviewed_at, f.approved_at,
			f.created_at, f.updated_at
		FROM case_findings f
		JOIN users cu ON cu.id = f.created_by
		LEFT JOIN users ru ON ru.id = f.reviewed_by
		LEFT JOIN users au ON au.id = f.approved_by
		WHERE f.id = $1 AND f.deleted_at IS NULL`, id,
	).Scan(
		&row.ID, &row.CaseID, &row.Title, &row.Description,
		&row.Category, &row.Severity, &row.Status, &row.NarrativeWeight,
		&row.ConfidenceScore, &row.AIGenerated, &row.SourceAILogID,
		&row.EvidenceCount,
		&row.CreatedBy, &row.CreatedByName,
		&row.ReviewedBy, &row.ReviewedByName,
		&row.ApprovedBy, &row.ApprovedByName,
		&row.ReviewedAt, &row.ApprovedAt,
		&row.CreatedAt, &row.UpdatedAt,
	)
	return row, err
}

func (r *Repository) List(ctx context.Context, caseID uuid.UUID, statusFilter, severityFilter string, limit int, cursorID uuid.UUID) ([]findingRow, error) {
	query := `
		SELECT
			f.id, f.case_id, f.title, f.description,
			f.category::TEXT, f.severity::TEXT, f.status::TEXT, f.narrative_weight::TEXT,
			f.confidence_score::TEXT, f.ai_generated, f.source_ai_log_id,
			COALESCE((SELECT COUNT(*) FROM finding_evidence_links WHERE finding_id = f.id), 0),
			f.created_by, COALESCE(cu.full_name,'') AS created_by_name,
			f.reviewed_by, ru.full_name,
			f.approved_by, au.full_name,
			f.reviewed_at, f.approved_at,
			f.created_at, f.updated_at
		FROM case_findings f
		JOIN users cu ON cu.id = f.created_by
		LEFT JOIN users ru ON ru.id = f.reviewed_by
		LEFT JOIN users au ON au.id = f.approved_by
		WHERE f.case_id = $1
		  AND f.deleted_at IS NULL
		  AND ($2 = '' OR f.status::TEXT = $2)
		  AND ($3 = '' OR f.severity::TEXT = $3)
		  AND ($4::uuid = '00000000-0000-0000-0000-000000000000' OR f.id < $4::uuid)
		ORDER BY
			CASE f.severity::TEXT
				WHEN 'critical' THEN 1 WHEN 'high' THEN 2
				WHEN 'medium' THEN 3 ELSE 4
			END,
			f.created_at DESC
		LIMIT $5`

	rows, err := r.db.QueryContext(ctx, query, caseID, statusFilter, severityFilter, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []findingRow
	for rows.Next() {
		var row findingRow
		if err := rows.Scan(
			&row.ID, &row.CaseID, &row.Title, &row.Description,
			&row.Category, &row.Severity, &row.Status, &row.NarrativeWeight,
			&row.ConfidenceScore, &row.AIGenerated, &row.SourceAILogID,
			&row.EvidenceCount,
			&row.CreatedBy, &row.CreatedByName,
			&row.ReviewedBy, &row.ReviewedByName,
			&row.ApprovedBy, &row.ApprovedByName,
			&row.ReviewedAt, &row.ApprovedAt,
			&row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) Count(ctx context.Context, caseID uuid.UUID, statusFilter, severityFilter string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM case_findings
		WHERE case_id = $1 AND deleted_at IS NULL
		  AND ($2 = '' OR status::TEXT = $2)
		  AND ($3 = '' OR severity::TEXT = $3)`,
		caseID, statusFilter, severityFilter,
	).Scan(&n)
	return n, err
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateFindingRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_findings SET
			title           = COALESCE($2, title),
			description     = COALESCE($3, description),
			category        = COALESCE($4::finding_category, category),
			severity        = COALESCE($5::finding_severity, severity),
			narrative_weight = COALESCE($6::finding_narrative_weight, narrative_weight)
		WHERE id = $1 AND deleted_at IS NULL`,
		id, req.Title, req.Description, req.Category, req.Severity, req.NarrativeWeight,
	)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, id, actorID uuid.UUID, newStatus string, reviewedAt, approvedAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_findings SET
			status      = $2::finding_status,
			reviewed_by = CASE WHEN $2 IN ('under_review','rejected','contested') THEN $3 ELSE reviewed_by END,
			approved_by = CASE WHEN $2 = 'approved' THEN $3 ELSE approved_by END,
			reviewed_at = CASE WHEN $4::timestamptz IS NOT NULL THEN $4::timestamptz ELSE reviewed_at END,
			approved_at = CASE WHEN $5::timestamptz IS NOT NULL THEN $5::timestamptz ELSE approved_at END
		WHERE id = $1 AND deleted_at IS NULL`,
		id, newStatus, actorID, reviewedAt, approvedAt,
	)
	return err
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE case_findings SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

// Summary returns quick counts for a case (used in case detail header).
func (r *Repository) Summary(ctx context.Context, caseID uuid.UUID) (FindingsSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT severity::TEXT, status::TEXT, COUNT(*)
		FROM case_findings
		WHERE case_id = $1 AND deleted_at IS NULL
		GROUP BY severity, status`, caseID)
	if err != nil {
		return FindingsSummary{}, err
	}
	defer rows.Close()

	var s FindingsSummary
	for rows.Next() {
		var sev, stat string
		var cnt int
		if err := rows.Scan(&sev, &stat, &cnt); err != nil {
			continue
		}
		s.Total += cnt
		if stat == "approved" {
			s.Approved += cnt
		}
		if sev == "critical" {
			s.Critical += cnt
		}
		if sev == "high" {
			s.High += cnt
		}
	}
	return s, rows.Err()
}

// ── Evidence links ────────────────────────────────────────────────────────────

func (r *Repository) LinkEvidence(ctx context.Context, findingID, linkedBy uuid.UUID, req LinkEvidenceRequest) (evidenceLinkRow, error) {
	evidenceID, err := uuid.Parse(req.EvidenceID)
	if err != nil {
		return evidenceLinkRow{}, err
	}

	var chunkID uuid.NullUUID
	if req.ChunkID != nil {
		if id, err := uuid.Parse(*req.ChunkID); err == nil {
			chunkID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	var pageNumber sql.NullInt32
	if req.PageNumber != nil {
		pageNumber = sql.NullInt32{Int32: int32(*req.PageNumber), Valid: true}
	}

	var row evidenceLinkRow
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO finding_evidence_links
			(finding_id, evidence_type, evidence_id, citation, page_number, chunk_id, relevance_note, linked_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (finding_id, evidence_type, evidence_id) DO UPDATE
			SET citation = EXCLUDED.citation,
			    relevance_note = EXCLUDED.relevance_note,
			    page_number = EXCLUDED.page_number,
			    chunk_id = EXCLUDED.chunk_id
		RETURNING id, finding_id, evidence_type, evidence_id, citation,
		          page_number, chunk_id, relevance_note, linked_by, linked_at`,
		findingID, req.EvidenceType, evidenceID, req.Citation,
		pageNumber, chunkID, req.RelevanceNote, linkedBy,
	).Scan(
		&row.ID, &row.FindingID, &row.EvidenceType, &row.EvidenceID,
		&row.Citation, &row.PageNumber, &row.ChunkID, &row.RelevanceNote,
		&row.LinkedBy, &row.LinkedAt,
	)
	return row, err
}

func (r *Repository) ListEvidence(ctx context.Context, findingID uuid.UUID) ([]evidenceLinkRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, finding_id, evidence_type, evidence_id, citation,
		       page_number, chunk_id, relevance_note, linked_by, linked_at
		FROM finding_evidence_links
		WHERE finding_id = $1
		ORDER BY linked_at ASC`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []evidenceLinkRow
	for rows.Next() {
		var row evidenceLinkRow
		if err := rows.Scan(
			&row.ID, &row.FindingID, &row.EvidenceType, &row.EvidenceID,
			&row.Citation, &row.PageNumber, &row.ChunkID, &row.RelevanceNote,
			&row.LinkedBy, &row.LinkedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) RemoveEvidence(ctx context.Context, linkID, findingID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM finding_evidence_links WHERE id = $1 AND finding_id = $2`, linkID, findingID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ── Evidence resolution helpers ────────────────────────────────────────────────
// These enrich evidence links with display metadata from their source tables.

type EvidenceDetail struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`       // doc filename, note title, etc.
	Preview     string `json:"preview"`     // short text excerpt
	URL         string `json:"url,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func (r *Repository) ResolveEvidenceDetails(ctx context.Context, links []evidenceLinkRow) map[string]EvidenceDetail {
	details := make(map[string]EvidenceDetail, len(links))
	for _, link := range links {
		var title, preview string
		switch link.EvidenceType {
		case "document":
			_ = r.db.QueryRowContext(ctx,
				`SELECT original_filename, '' FROM documents WHERE id = $1`, link.EvidenceID,
			).Scan(&title, &preview)
		case "note":
			_ = r.db.QueryRowContext(ctx,
				`SELECT title, LEFT(content_text, 120) FROM notes WHERE id = $1`, link.EvidenceID,
			).Scan(&title, &preview)
		case "ai_log":
			_ = r.db.QueryRowContext(ctx,
				`SELECT action_type, LEFT(response_text, 120) FROM ai_logs WHERE id = $1`, link.EvidenceID,
			).Scan(&title, &preview)
		case "subject":
			_ = r.db.QueryRowContext(ctx,
				`SELECT canonical_name, entity_type FROM subjects WHERE id = $1`, link.EvidenceID,
			).Scan(&title, &preview)
		}
		details[link.EvidenceID.String()] = EvidenceDetail{
			ID:      link.EvidenceID.String(),
			Type:    link.EvidenceType,
			Title:   title,
			Preview: preview,
		}
	}
	return details
}

// GetFindingByCaseAndID validates that the finding belongs to the case (IDOR guard).
func (r *Repository) GetFindingByCaseAndID(ctx context.Context, caseID, findingID uuid.UUID) (findingRow, error) {
	row, err := r.GetByID(ctx, findingID)
	if err != nil {
		return findingRow{}, err
	}
	if row.CaseID != caseID {
		return findingRow{}, sql.ErrNoRows
	}
	return row, nil
}

// GetFindingOwner returns the created_by UUID for permission checks.
func (r *Repository) GetFindingOwner(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var ownerID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT created_by FROM case_findings WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&ownerID)
	return ownerID, err
}

// ApprovedFindings returns all approved findings for a case (used by closure pre-check and STR).
func (r *Repository) ApprovedFindings(ctx context.Context, caseID uuid.UUID) ([]findingRow, error) {
	return r.List(ctx, caseID, "approved", "", 100, uuid.Nil)
}

// IsCaseClosed returns true when the case status is 'closed'.
func (r *Repository) IsCaseClosed(ctx context.Context, caseID uuid.UUID) bool {
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT status FROM cases WHERE id = $1`, caseID).Scan(&status)
	return err == nil && status == "closed"
}

