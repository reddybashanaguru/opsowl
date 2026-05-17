// cmd/seed/main.go — populate the dev database with realistic OpsOwl demo data.
// Usage: DATABASE_URL=... go run ./cmd/seed  (from apps/api directory)
// Idempotent: truncates seeded tables then re-inserts.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxstdlib "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed error:", err)
		os.Exit(1)
	}
}

func ptr[T any](v T) *T { return &v }

func ago(d time.Duration) time.Time { return time.Now().UTC().Add(-d) }

func jsonb(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

var demoOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://opsowl:opsowl@localhost:5433/opsowl?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	must(err)
	defer pool.Close()

	sqlDB := pgxstdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	q := dbgen.New(sqlDB)

	fmt.Println("🌱 Seeding OpsOwl demo data...")
	truncate(sqlDB)
	users := seedUsers(ctx, q)
	cases := seedCases(ctx, q, users)
	seedTimeline(ctx, q, cases, users)
	seedTasks(ctx, q, cases, users)
	seedDocuments(ctx, q, cases, users)
	seedAILogs(ctx, q, cases, users)
	seedAuditLogs(ctx, q, users, cases)
	fmt.Println("✅ Done. Login: admin@opsowl.dev / OpsOwl@2024")
}

// ── Truncate ──────────────────────────────────────────────────────────────────

func truncate(db *sql.DB) {
	stmts := []string{
		"TRUNCATE audit_logs, ai_logs, document_chunks, documents, task_comments, tasks, timeline_events, case_activities, case_watchers, case_assignments, cases, refresh_tokens, sessions, users CASCADE",
		"ALTER SEQUENCE case_number_seq RESTART WITH 1",
	}
	for _, s := range stmts {
		_, err := db.Exec(s)
		must(err)
	}
	fmt.Println("  ↩  tables truncated")
}

// ── Users ─────────────────────────────────────────────────────────────────────

type userSet struct {
	admin, manager, analyst1, analyst2, viewer dbgen.CreateUserRow
}

func hashPwd(pwd string) string {
	b, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	must(err)
	return string(b)
}

func seedUsers(ctx context.Context, q *dbgen.Queries) userSet {
	hash := hashPwd("OpsOwl@2024")
	mk := func(email, name string, role dbgen.UserRole) dbgen.CreateUserRow {
		u, err := q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        email,
			FullName:     name,
			PasswordHash: hash,
			Role:         role,
		})
		must(err)
		return u
	}
	s := userSet{
		admin:    mk("admin@opsowl.dev", "Priya Sharma", dbgen.UserRoleSuperAdmin),
		manager:  mk("ravi.kumar@opsowl.dev", "Ravi Kumar", dbgen.UserRoleManager),
		analyst1: mk("divya.reddy@opsowl.dev", "Divya Reddy", dbgen.UserRoleAnalyst),
		analyst2: mk("arjun.mehta@opsowl.dev", "Arjun Mehta", dbgen.UserRoleAnalyst),
		viewer:   mk("sneha.patel@opsowl.dev", "Sneha Patel", dbgen.UserRoleViewer),
	}
	fmt.Println("  👤 5 users created (all password: OpsOwl@2024)")
	return s
}

// ── Cases ─────────────────────────────────────────────────────────────────────

type caseSet struct {
	upi, kyc, loan, merchant dbgen.Case
}

func seedCases(ctx context.Context, q *dbgen.Queries, u userSet) caseSet {
	mk := func(title, desc string, status dbgen.CaseStatus, prio dbgen.CasePriority,
		createdBy uuid.UUID, assignedTo *uuid.UUID, tags []string, dueDaysFromNow int) dbgen.Case {

		var dueDate, slaAt sql.NullTime
		if dueDaysFromNow != 0 {
			dueDate = sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, dueDaysFromNow), Valid: true}
			slaAt = sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, dueDaysFromNow+2), Valid: true}
		}
		var assigned uuid.NullUUID
		if assignedTo != nil {
			assigned = uuid.NullUUID{UUID: *assignedTo, Valid: true}
		}
		c, err := q.CreateCase(ctx, dbgen.CreateCaseParams{
			Title:          title,
			Description:    desc,
			Status:         status,
			Priority:       prio,
			CreatedBy:      createdBy,
			OrganizationID: demoOrgID,
			AssignedTo:     assigned,
			DueDate:        dueDate,
			SlaBreachAt:    slaAt,
			Tags:           tags,
		})
		must(err)
		return c
	}

	s := caseSet{
		upi: mk(
			"Suspicious UPI Cluster — Account 9876543210",
			"A cluster of 47 high-value UPI transactions totalling ₹42.3 lakh were detected "+
				"originating from account 9876543210 within a 72-hour window. Beneficiary accounts "+
				"span 3 different banks with no prior transaction history. Possible layering activity "+
				"consistent with PMLA Schedule II predicate offence.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityCritical,
			u.analyst1.ID, ptr(u.analyst2.ID),
			[]string{"upi", "fraud", "layering", "high-value"}, 3),

		kyc: mk(
			"KYC Document Forgery — Customer CUST-2024-0891",
			"Branch staff flagged potential forgery in Aadhaar and PAN documents submitted by customer "+
				"CUST-2024-0891 during loan onboarding. Hologram positioning and font inconsistencies noted. "+
				"UIDAI eKYC verification pending. Loan disbursement frozen pending investigation.",
			dbgen.CaseStatusPendingReview, dbgen.CasePriorityHigh,
			u.analyst2.ID, ptr(u.analyst1.ID),
			[]string{"kyc", "forgery", "compliance", "loan-onboarding"}, 5),

		loan: mk(
			"Loan Disbursement Anomaly — Application LAF-2024-4412",
			"Loan of ₹18.5 lakh disbursed to LAF-2024-4412 within 4 hours of approval — significantly "+
				"below the mandatory 48-hour cooling period. Disbursement account differs from the one "+
				"in the application. No maker-checker audit trail found in CBS logs. Possible insider threat "+
				"or API-level bypass of the business rule engine.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityHigh,
			u.manager.ID, ptr(u.analyst1.ID),
			[]string{"loan", "disbursement", "insider-threat", "anomaly"}, 7),

		merchant: mk(
			"Merchant Settlement Dispute — Batch NOV-2024-B88",
			"Merchant MCHT-9021 (QuickMart Retail) disputes short-settlement of ₹2.1 lakh in November "+
				"batch NOV-2024-B88. Root cause: fee calculation bug in settlement engine v2.8.1 incorrectly "+
				"applied compound rounding to international card transactions. Hotfix v2.8.2 deployed. "+
				"Manual credit of ₹2,10,000 issued to merchant. Case closed.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityMedium,
			u.analyst1.ID, nil,
			[]string{"merchant", "settlement", "dispute", "reconciliation"}, 0),
	}
	fmt.Println("  📁 4 cases created (OWL-2026-00001 … OWL-2026-00004)")
	return s
}

// ── Timeline Events ───────────────────────────────────────────────────────────

func emit(ctx context.Context, q *dbgen.Queries,
	caseID, actorID uuid.UUID,
	evtType dbgen.TimelineEventType, src dbgen.TimelineEventSource,
	at time.Time, payload any) {

	_, err := q.InsertTimelineEvent(ctx, dbgen.InsertTimelineEventParams{
		CaseID:     caseID,
		EventType:  evtType,
		Source:     src,
		ActorID:    uuid.NullUUID{UUID: actorID, Valid: true},
		OccurredAt: at,
		Payload:    jsonb(payload),
		Metadata:   jsonb(map[string]any{}),
	})
	must(err)
}

func seedTimeline(ctx context.Context, q *dbgen.Queries, c caseSet, u userSet) {
	// ── UPI Case (7 days ago → ongoing)
	t0 := ago(7 * 24 * time.Hour)
	emit(ctx, q, c.upi.ID, u.analyst1.ID, dbgen.TimelineEventTypeCaseCreated, dbgen.TimelineEventSourceUser, t0,
		map[string]any{"case_number": c.upi.CaseNumber, "title": c.upi.Title, "priority": "critical"})
	emit(ctx, q, c.upi.ID, u.manager.ID, dbgen.TimelineEventTypeCaseAssigned, dbgen.TimelineEventSourceUser, t0.Add(30*time.Minute),
		map[string]any{"assigned_to_name": "Arjun Mehta", "assigned_by_name": "Ravi Kumar"})
	emit(ctx, q, c.upi.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentUploaded, dbgen.TimelineEventSourceUser, t0.Add(2*time.Hour),
		map[string]any{"filename": "upi_transaction_export_nov2024.pdf", "size_bytes": 1245184, "mime": "application/pdf"})
	emit(ctx, q, c.upi.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentProcessed, dbgen.TimelineEventSourceAi, t0.Add(2*time.Hour+5*time.Minute),
		map[string]any{"filename": "upi_transaction_export_nov2024.pdf", "chunks": 80, "pages": 15, "status": "completed"})
	emit(ctx, q, c.upi.ID, u.analyst2.ID, dbgen.TimelineEventTypeAiQuery, dbgen.TimelineEventSourceAi, t0.Add(3*time.Hour),
		map[string]any{"action_type": "ask", "model": "claude-opus-4-5", "latency_ms": 2340})
	emit(ctx, q, c.upi.ID, u.analyst1.ID, dbgen.TimelineEventTypeCaseStatusChanged, dbgen.TimelineEventSourceUser, t0.Add(5*time.Hour),
		map[string]any{"from": "open", "to": "in_progress"})
	emit(ctx, q, c.upi.ID, u.analyst2.ID, dbgen.TimelineEventTypeTaskCreated, dbgen.TimelineEventSourceUser, t0.Add(6*time.Hour),
		map[string]any{"title": "Pull all UPI transactions for account 9876543210"})
	emit(ctx, q, c.upi.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentUploaded, dbgen.TimelineEventSourceUser, t0.Add(8*time.Hour),
		map[string]any{"filename": "account_statement_q3_q4_2024.pdf", "size_bytes": 987136, "mime": "application/pdf"})
	emit(ctx, q, c.upi.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentProcessed, dbgen.TimelineEventSourceAi, t0.Add(8*time.Hour+6*time.Minute),
		map[string]any{"filename": "account_statement_q3_q4_2024.pdf", "chunks": 62, "pages": 12, "status": "completed"})
	emit(ctx, q, c.upi.ID, u.manager.ID, dbgen.TimelineEventTypeAiSummaryGenerated, dbgen.TimelineEventSourceAi, t0.Add(9*time.Hour),
		map[string]any{"action_type": "summarize", "review_status": "approved", "model": "claude-opus-4-5"})

	// ── KYC Case (5 days ago → pending review)
	t1 := ago(5 * 24 * time.Hour)
	emit(ctx, q, c.kyc.ID, u.analyst2.ID, dbgen.TimelineEventTypeCaseCreated, dbgen.TimelineEventSourceUser, t1,
		map[string]any{"case_number": c.kyc.CaseNumber, "priority": "high"})
	emit(ctx, q, c.kyc.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentUploaded, dbgen.TimelineEventSourceUser, t1.Add(1*time.Hour),
		map[string]any{"filename": "kyc_aadhaar_copy.pdf", "size_bytes": 204800})
	emit(ctx, q, c.kyc.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentUploaded, dbgen.TimelineEventSourceUser, t1.Add(1*time.Hour+5*time.Minute),
		map[string]any{"filename": "pan_card_scan.jpg", "size_bytes": 98304, "processing_failed": true})
	emit(ctx, q, c.kyc.ID, u.analyst1.ID, dbgen.TimelineEventTypeAiQuery, dbgen.TimelineEventSourceAi, t1.Add(2*time.Hour),
		map[string]any{"action_type": "ask", "question_summary": "Compare uploaded Aadhaar with standard format indicators"})
	emit(ctx, q, c.kyc.ID, u.manager.ID, dbgen.TimelineEventTypeAiActionSuggested, dbgen.TimelineEventSourceAi, t1.Add(2*time.Hour+30*time.Minute),
		map[string]any{"action_type": "extract_actions", "suggestion_count": 4, "review_status": "approved"})
	emit(ctx, q, c.kyc.ID, u.analyst1.ID, dbgen.TimelineEventTypeCaseStatusChanged, dbgen.TimelineEventSourceUser, t1.Add(4*time.Hour),
		map[string]any{"from": "open", "to": "pending_review"})

	// ── Loan Case (3 days ago → open)
	t2 := ago(3 * 24 * time.Hour)
	emit(ctx, q, c.loan.ID, u.manager.ID, dbgen.TimelineEventTypeCaseCreated, dbgen.TimelineEventSourceUser, t2,
		map[string]any{"case_number": c.loan.CaseNumber, "priority": "high"})
	emit(ctx, q, c.loan.ID, u.manager.ID, dbgen.TimelineEventTypeCaseAssigned, dbgen.TimelineEventSourceUser, t2.Add(15*time.Minute),
		map[string]any{"assigned_to_name": "Divya Reddy"})
	emit(ctx, q, c.loan.ID, u.analyst1.ID, dbgen.TimelineEventTypeDocumentUploaded, dbgen.TimelineEventSourceUser, t2.Add(2*time.Hour),
		map[string]any{"filename": "loan_application_form_4412.pdf", "size_bytes": 512000, "status": "processing"})
	emit(ctx, q, c.loan.ID, u.analyst1.ID, dbgen.TimelineEventTypeAiSummaryGenerated, dbgen.TimelineEventSourceAi, t2.Add(3*time.Hour),
		map[string]any{"action_type": "summarize", "review_status": "pending"})

	// ── Merchant Case (14 days ago → closed)
	t3 := ago(14 * 24 * time.Hour)
	emit(ctx, q, c.merchant.ID, u.analyst1.ID, dbgen.TimelineEventTypeCaseCreated, dbgen.TimelineEventSourceUser, t3,
		map[string]any{"case_number": c.merchant.CaseNumber, "priority": "medium"})
	emit(ctx, q, c.merchant.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentUploaded, dbgen.TimelineEventSourceUser, t3.Add(30*time.Minute),
		map[string]any{"filename": "merchant_settlement_batch_nov88.csv", "size_bytes": 45056})
	emit(ctx, q, c.merchant.ID, u.analyst2.ID, dbgen.TimelineEventTypeDocumentProcessed, dbgen.TimelineEventSourceAi, t3.Add(45*time.Minute),
		map[string]any{"filename": "merchant_settlement_batch_nov88.csv", "chunks": 20, "pages": 1, "status": "completed"})
	emit(ctx, q, c.merchant.ID, u.analyst2.ID, dbgen.TimelineEventTypeAiQuery, dbgen.TimelineEventSourceAi, t3.Add(2*time.Hour),
		map[string]any{"action_type": "summarize", "review_status": "edited"})
	emit(ctx, q, c.merchant.ID, u.manager.ID, dbgen.TimelineEventTypeCaseStatusChanged, dbgen.TimelineEventSourceUser, t3.Add(3*24*time.Hour),
		map[string]any{"from": "in_progress", "to": "closed",
			"resolution": "Fee calculation bug fixed in v2.8.2, ₹2,10,000 credit issued"})

	fmt.Println("  📅 25 timeline events created")
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

func mkTask(ctx context.Context, q *dbgen.Queries,
	caseID, createdBy uuid.UUID, assignedTo *uuid.UUID,
	title, desc string, prio dbgen.TaskPriority,
	dueOffset time.Duration, isAI bool) dbgen.Task {

	var due sql.NullTime
	if dueOffset != 0 {
		due = sql.NullTime{Time: time.Now().UTC().Add(dueOffset), Valid: true}
	}
	var assigned uuid.NullUUID
	if assignedTo != nil {
		assigned = uuid.NullUUID{UUID: *assignedTo, Valid: true}
	}
	var descNull sql.NullString
	if desc != "" {
		descNull = sql.NullString{String: desc, Valid: true}
	}
	t, err := q.CreateTask(ctx, dbgen.CreateTaskParams{
		CaseID:        caseID,
		CreatedBy:     createdBy,
		AssignedTo:    assigned,
		Title:         title,
		Description:   descNull,
		Priority:      prio,
		DueDate:       due,
		IsAiSuggested: isAI,
	})
	must(err)
	return t
}

func seedTasks(ctx context.Context, q *dbgen.Queries, c caseSet, u userSet) {
	// ── UPI Case tasks
	mkTask(ctx, q, c.upi.ID, u.analyst1.ID, ptr(u.analyst2.ID),
		"Pull all UPI transactions for account 9876543210 (last 6 months)",
		"Export via CBS template UPI-TXN-EXPORT-01. Include all debit/credit legs. Flag sub-₹1L transactions.",
		dbgen.TaskPriorityCritical, 48*time.Hour, false)
	mkTask(ctx, q, c.upi.ID, u.analyst1.ID, ptr(u.analyst1.ID),
		"Send Letter of Cooperation to 3 beneficiary banks",
		"LOC format per RBI Master Direction on KYC. Addressed to HDFC BLR-Central, ICICI KOR, Axis SEC.",
		dbgen.TaskPriorityHigh, 72*time.Hour, false)
	mkTask(ctx, q, c.upi.ID, u.analyst2.ID, ptr(u.analyst2.ID),
		"Prepare Suspicious Transaction Report (STR) — FIU-IND Form A",
		"Deadline: 7 working days from initial detection. Attach CBS export and account profile.",
		dbgen.TaskPriorityHigh, 5*24*time.Hour, true)
	mkTask(ctx, q, c.upi.ID, u.analyst2.ID, ptr(u.analyst1.ID),
		"Initiate Section 12A PMLA freeze on 3 beneficiary accounts",
		"Coordinate with Compliance HOD. Freeze order must be served within 24 hours.",
		dbgen.TaskPriorityCritical, 24*time.Hour, true)

	// ── KYC Case tasks
	mkTask(ctx, q, c.kyc.ID, u.analyst2.ID, ptr(u.analyst1.ID),
		"Verify Aadhaar via UIDAI eKYC API (UID 7319 **** ****)",
		"Use UIDAI eKYC v2.5 OTP flow. Screenshot API response for audit trail.",
		dbgen.TaskPriorityHigh, 0, false)
	mkTask(ctx, q, c.kyc.ID, u.analyst2.ID, ptr(u.analyst2.ID),
		"Request physical KYC dossier from branch BLR-049 (Koramangala)",
		"Courier request via branch ops portal. Expected TAT: 3 working days.",
		dbgen.TaskPriorityMedium, 3*24*time.Hour, false)
	mkTask(ctx, q, c.kyc.ID, u.analyst1.ID, ptr(u.analyst1.ID),
		"Cross-check PAN BXQPR4421L with ITD database",
		"ITD e-filing portal → PAN verification tab. Screenshot as evidence.",
		dbgen.TaskPriorityHigh, 0, false)
	mkTask(ctx, q, c.kyc.ID, u.manager.ID, ptr(u.manager.ID),
		"Escalate to CKYC registry if forgery confirmed",
		"File CKYC modification request. Regulatory notification to RBI within 7 days per PMLA Schedule II.",
		dbgen.TaskPriorityCritical, 7*24*time.Hour, true)

	// ── Loan Case tasks
	mkTask(ctx, q, c.loan.ID, u.manager.ID, ptr(u.analyst1.ID),
		"Audit CBS disbursement logs for LAF-2024-4412",
		"Pull from LOAN_DISBURSE_AUDIT table. Check maker-checker bypass flags and approval timestamps.",
		dbgen.TaskPriorityCritical, 24*time.Hour, false)
	mkTask(ctx, q, c.loan.ID, u.analyst1.ID, ptr(u.analyst1.ID),
		"Obtain CIBIL TransUnion Score v2 for applicant",
		"Consumer portal OTP auth required. Consent form timestamp must be documented.",
		dbgen.TaskPriorityHigh, 48*time.Hour, false)
	mkTask(ctx, q, c.loan.ID, u.analyst1.ID, ptr(u.analyst1.ID),
		"Review CCTV footage from branch BLR-011 on disbursement day",
		"10:00–16:00 IST window. Request via Security Ops portal with case reference.",
		dbgen.TaskPriorityMedium, 5*24*time.Hour, true)

	// ── Merchant Case tasks (all closed)
	mkTask(ctx, q, c.merchant.ID, u.analyst1.ID, ptr(u.analyst2.ID),
		"Reconcile settlement batch NOV-2024-B88 line by line",
		"Compare gateway capture CSV with settlement engine output. Log discrepancy per txn.",
		dbgen.TaskPriorityHigh, 0, false)
	mkTask(ctx, q, c.merchant.ID, u.analyst2.ID, ptr(u.analyst2.ID),
		"Process manual credit of ₹2,10,000 to QuickMart Retail (MCHT-9021)",
		"Via Settlement Ops portal. Attach reconciliation evidence and approval from manager.",
		dbgen.TaskPriorityHigh, 0, false)
	mkTask(ctx, q, c.merchant.ID, u.manager.ID, ptr(u.manager.ID),
		"Draft post-incident report for MD/CEO review",
		"Root cause, impact (847 txns, ₹2.1L), timeline, hotfix details, preventive controls added.",
		dbgen.TaskPriorityMedium, 0, false)

	fmt.Println("  ✅ 14 tasks created")
}

// ── Documents ─────────────────────────────────────────────────────────────────

func mkDoc(ctx context.Context, q *dbgen.Queries,
	caseID, uploader uuid.UUID,
	filename, fileType, mimeType string,
	sizeBytes int64,
	status dbgen.DocumentProcessingStatus,
	pages, chunks int) dbgen.Document {

	docID := uuid.New()
	d, err := q.CreateDocument(ctx, dbgen.CreateDocumentParams{
		ID:               docID,
		CaseID:           caseID,
		UploadedBy:       uploader,
		OriginalFilename: filename,
		StoredPath:       fmt.Sprintf("cases/%s/docs/%s/%s", caseID, docID, filename),
		FileType:         fileType,
		MimeType:         mimeType,
		FileSizeBytes:    sizeBytes,
	})
	must(err)

	if status != dbgen.DocumentProcessingStatusPending {
		var pageCount, chunkCount sql.NullInt32
		var errMsg sql.NullString
		if pages > 0 {
			pageCount = sql.NullInt32{Int32: int32(pages), Valid: true}
			chunkCount = sql.NullInt32{Int32: int32(chunks), Valid: true}
		}
		if status == dbgen.DocumentProcessingStatusFailed {
			errMsg = sql.NullString{String: "OCR failed: unreadable scan quality (possible digital reconstruction)", Valid: true}
		}
		_, err = q.UpdateDocumentProcessingResult(ctx, dbgen.UpdateDocumentProcessingResultParams{
			ID:               d.ID,
			ProcessingStatus: status,
			PageCount:        pageCount,
			ChunkCount:       chunkCount,
			ErrorMessage:     errMsg,
		})
		must(err)
	}
	return d
}

func seedDocuments(ctx context.Context, q *dbgen.Queries, c caseSet, u userSet) {
	mkDoc(ctx, q, c.upi.ID, u.analyst2.ID,
		"upi_transaction_export_nov2024.pdf", "pdf", "application/pdf", 1245184,
		dbgen.DocumentProcessingStatusCompleted, 15, 80)
	mkDoc(ctx, q, c.upi.ID, u.analyst2.ID,
		"account_statement_q3_q4_2024.pdf", "pdf", "application/pdf", 987136,
		dbgen.DocumentProcessingStatusCompleted, 12, 62)
	mkDoc(ctx, q, c.kyc.ID, u.analyst2.ID,
		"kyc_aadhaar_copy.pdf", "pdf", "application/pdf", 204800,
		dbgen.DocumentProcessingStatusCompleted, 2, 8)
	mkDoc(ctx, q, c.kyc.ID, u.analyst2.ID,
		"pan_card_scan.jpg", "jpg", "image/jpeg", 98304,
		dbgen.DocumentProcessingStatusFailed, 0, 0) // forgery — can't process
	mkDoc(ctx, q, c.loan.ID, u.analyst1.ID,
		"loan_application_form_4412.pdf", "pdf", "application/pdf", 512000,
		dbgen.DocumentProcessingStatusProcessing, 0, 0)
	mkDoc(ctx, q, c.merchant.ID, u.analyst2.ID,
		"merchant_settlement_batch_nov88.csv", "csv", "text/csv", 45056,
		dbgen.DocumentProcessingStatusCompleted, 1, 20)

	fmt.Println("  📄 6 documents seeded (completed/processing/failed states)")
}

// ── AI Logs ───────────────────────────────────────────────────────────────────

func mkAILog(ctx context.Context, q *dbgen.Queries,
	caseID, triggeredBy uuid.UUID,
	actionType, promptText, responseText, model string,
	tokIn, tokOut, latency int32,
	conf float64,
	reviewStatus string,
	reviewedBy *uuid.UUID,
	humanEdited *string) {

	citations := jsonb([]map[string]any{
		{
			"doc_id": "doc-seed-1", "doc_name": "upi_transaction_export_nov2024.pdf",
			"chunk_index": 3, "page_number": 4,
			"excerpt":    "47 outward UPI transactions totalling INR 42,30,000 detected within 72 hours.",
			"similarity": 0.91,
		},
	})
	var confScore sql.NullString
	if conf > 0 {
		confScore = sql.NullString{String: fmt.Sprintf("%.2f", conf), Valid: true}
	}

	row, err := q.CreateAILog(ctx, dbgen.CreateAILogParams{
		CaseID:          caseID,
		TriggeredBy:     triggeredBy,
		ActionType:      actionType,
		PromptText:      promptText,
		ResponseText:    responseText,
		ModelUsed:       model,
		Provider:        "anthropic",
		TokensInput:     tokIn,
		TokensOutput:    tokOut,
		LatencyMs:       latency,
		ConfidenceScore: confScore,
		Citations:       citations,
	})
	must(err)

	switch reviewStatus {
	case "approved":
		_, err = q.ApproveAILog(ctx, dbgen.ApproveAILogParams{
			ID:         row.ID,
			ReviewedBy: uuid.NullUUID{UUID: *reviewedBy, Valid: true},
		})
		must(err)
	case "rejected":
		reason := ""
		if humanEdited != nil {
			reason = *humanEdited
		}
		_, err = q.RejectAILog(ctx, dbgen.RejectAILogParams{
			ID:                  row.ID,
			ReviewedBy:          uuid.NullUUID{UUID: *reviewedBy, Valid: true},
			HumanEditedResponse: sql.NullString{String: reason, Valid: reason != ""},
		})
		must(err)
	case "edited":
		edit := ""
		if humanEdited != nil {
			edit = *humanEdited
		}
		_, err = q.EditAILog(ctx, dbgen.EditAILogParams{
			ID:                  row.ID,
			ReviewedBy:          uuid.NullUUID{UUID: *reviewedBy, Valid: true},
			HumanEditedResponse: sql.NullString{String: edit, Valid: edit != ""},
		})
		must(err)
	}
}

func seedAILogs(ctx context.Context, q *dbgen.Queries, c caseSet, u userSet) {
	// UPI — Ask (pending review)
	mkAILog(ctx, q, c.upi.ID, u.analyst2.ID, "ask",
		"What are the suspicious patterns in the UPI transactions? Are there any layering indicators?",
		"Based on the 47 UPI transactions analysed, three suspicious patterns emerge:\n\n"+
			"**1. Structuring (Smurfing):** 23 of 47 transactions are just below ₹1 lakh — "+
			"the mandatory CTR reporting threshold — suggesting intentional fragmentation to avoid detection.\n\n"+
			"**2. Dormant Account Activation:** The account had zero outward UPI activity for 6 months "+
			"before the cluster. Sudden high-velocity activity is a hallmark of mule account activation.\n\n"+
			"**3. Mule-Profile Beneficiaries:** ₹31.2 lakh (73.8%) flowed to 3 accounts opened within "+
			"90 days of the first transaction, all with minimum KYC (Aadhaar OTP only, no income proof).\n\n"+
			"**Recommendation:** File STR with FIU-IND within 7 working days. Consider Section 12A PMLA freeze.",
		"claude-opus-4-5", 1820, 340, 2340, 0.87,
		"pending", nil, nil)

	// UPI — Summarize (approved by manager)
	mkAILog(ctx, q, c.upi.ID, u.analyst2.ID, "summarize",
		"[SYSTEM: Summarize the investigation case based on all available documents and context]",
		"**Case Summary — OWL-2026-00001**\n\n"+
			"Account 9876543210 executed 47 outward UPI transactions totalling ₹42.3 lakh in 72 hours. "+
			"Analysis indicates layering-phase money laundering: structuring below CTR threshold, "+
			"dormant account activation, and 3 mule-profile beneficiary accounts. "+
			"STR preparation in progress. LOC letters sent to HDFC, ICICI, Axis Bank.",
		"claude-opus-4-5", 2100, 280, 3100, 0.92,
		"approved", &u.manager.ID, nil)

	// KYC — Ask (rejected — model hallucinated measurements)
	mkAILog(ctx, q, c.kyc.ID, u.analyst1.ID, "ask",
		"Compare the uploaded Aadhaar scan with standard UIDAI document format. Identify anomalies.",
		"The uploaded Aadhaar shows anomalies: QR code positioned 2mm below standard, "+
			"hologram serial in 10-digit format (expected 12), Arial Narrow font used instead of "+
			"UIDAI's proprietary font, microprint border shows pixelation.",
		"claude-opus-4-5", 1540, 210, 1890, 0.61,
		"rejected", &u.manager.ID,
		ptr("Response rejected: model hallucinated specific measurements and font identifications. "+
			"Physical document inspection required for hologram/microprint analysis — AI cannot verify from a scan."))

	// KYC — Extract Actions (approved)
	mkAILog(ctx, q, c.kyc.ID, u.analyst1.ID, "extract_actions",
		"[SYSTEM: Extract actionable investigation steps from the KYC documents and case context]",
		"Recommended investigation actions:\n\n"+
			"1. **Verify Aadhaar via UIDAI eKYC API** (Critical) — OTP-based verification for UID 7319 ****\n"+
			"2. **Cross-check PAN with ITD database** (High) — e-filing portal verification\n"+
			"3. **Request physical KYC from branch BLR-049** (Medium) — 3 working day TAT\n"+
			"4. **Escalate to CKYC registry if forgery confirmed** (Critical) — RBI notification within 7 days",
		"claude-opus-4-5", 1680, 290, 2050, 0.78,
		"approved", &u.manager.ID, nil)

	// Loan — Summarize (pending)
	mkAILog(ctx, q, c.loan.ID, u.analyst1.ID, "summarize",
		"[SYSTEM: Summarize the investigation case based on all available documents]",
		"**Case Summary — OWL-2026-00003**\n\n"+
			"Loan ₹18.5L (LAF-2024-4412) disbursed in 4 hours vs. mandatory 48-hour cooling period. "+
			"Disbursement destination account differs from loan application. No CBS maker-checker trail. "+
			"Red flags: cooling period bypass, account mismatch, missing audit trail. "+
			"Hypothesis: insider access or API-level rule engine bypass. Investigation ongoing.",
		"claude-opus-4-5", 2300, 310, 3450, 0.83,
		"pending", nil, nil)

	// Merchant — Summarize (edited by manager for accuracy)
	mkAILog(ctx, q, c.merchant.ID, u.analyst1.ID, "summarize",
		"[SYSTEM: Summarize the investigation case based on all available documents]",
		"Merchant settlement dispute from a fee calculation error in settlement engine v2.8.1.",
		"claude-opus-4-5", 1920, 190, 2100, 0.95,
		"edited", &u.manager.ID,
		ptr("**Case Summary — OWL-2026-00004 (Closed)**\n\n"+
			"QuickMart Retail (MCHT-9021) was short-settled ₹2,10,000 across 847 transactions "+
			"in November batch NOV-2024-B88. Root cause: compound rounding bug in fee calculation "+
			"module v2.8.1 affecting international card transactions. Resolution: hotfix v2.8.2 deployed "+
			"2026-05-03; manual credit processed 2026-05-05. Preventive control: fee calculation "+
			"unit tests added to CI pipeline. Case closed with merchant confirmation."))

	fmt.Println("  🤖 6 AI logs created (pending × 2, approved × 2, rejected × 1, edited × 1)")
}

// ── Audit Logs ────────────────────────────────────────────────────────────────

func seedAuditLogs(ctx context.Context, q *dbgen.Queries, u userSet, c caseSet) {
	write := func(code string, actor *uuid.UUID, ip string, meta map[string]any, sev string) {
		var actorNull uuid.NullUUID
		if actor != nil {
			actorNull = uuid.NullUUID{UUID: *actor, Valid: true}
		}
		err := q.CreateAuditLog(ctx, dbgen.CreateAuditLogParams{
			EventCode: code,
			ActorID:   actorNull,
			IpAddress: sql.NullString{String: ip, Valid: true},
			Metadata:  jsonb(meta),
			Severity:  sev,
		})
		must(err)
	}

	// Auth events
	write("auth.login.success", &u.admin.ID, "203.192.18.44",
		map[string]any{"email": "admin@opsowl.dev", "user_agent": "Mozilla/5.0 Chrome/124"}, "info")
	write("auth.login.success", &u.analyst1.ID, "103.21.58.91",
		map[string]any{"email": "divya.reddy@opsowl.dev"}, "info")
	write("auth.login.success", &u.analyst2.ID, "103.21.58.92",
		map[string]any{"email": "arjun.mehta@opsowl.dev"}, "info")
	write("auth.login.success", &u.manager.ID, "10.0.1.15",
		map[string]any{"email": "ravi.kumar@opsowl.dev"}, "info")
	write("auth.login.failed", nil, "185.220.101.42",
		map[string]any{"email": "admin@opsowl.dev", "reason": "bad_password", "attempt": 1}, "warn")
	write("auth.login.failed", nil, "185.220.101.42",
		map[string]any{"email": "admin@opsowl.dev", "reason": "bad_password", "attempt": 2}, "warn")
	write("auth.token.reuse_detected", &u.analyst2.ID, "198.51.100.5",
		map[string]any{"email": "arjun.mehta@opsowl.dev", "rt_hash": "a3f9b1c2..."}, "critical")

	// DML mutations (representative entries — middleware captures these in real-time)
	write("api.cases.post", &u.analyst1.ID, "103.21.58.91",
		map[string]any{"method": "POST", "path": "/api/v1/cases", "status": 201, "case_id": c.upi.ID}, "info")
	write("api.cases.post", &u.analyst2.ID, "103.21.58.92",
		map[string]any{"method": "POST", "path": "/api/v1/cases", "status": 201, "case_id": c.kyc.ID}, "info")
	write("api.cases.documents.post", &u.analyst2.ID, "103.21.58.92",
		map[string]any{"method": "POST", "status": 201, "filename": "upi_transaction_export_nov2024.pdf", "case_id": c.upi.ID}, "info")
	write("api.cases.documents.post", &u.analyst2.ID, "103.21.58.92",
		map[string]any{"method": "POST", "status": 201, "filename": "pan_card_scan.jpg", "case_id": c.kyc.ID}, "info")
	write("api.cases.ai.ask.post", &u.analyst2.ID, "103.21.58.92",
		map[string]any{"method": "POST", "status": 200, "action_type": "ask", "case_id": c.upi.ID}, "info")
	write("api.ai.logs.approve.post", &u.manager.ID, "10.0.1.15",
		map[string]any{"method": "POST", "status": 200, "action": "approved", "reviewer": "ravi.kumar@opsowl.dev"}, "info")
	write("api.ai.logs.reject.post", &u.manager.ID, "10.0.1.15",
		map[string]any{"method": "POST", "status": 200, "action": "rejected", "reason": "model hallucinated"}, "warn")
	write("api.cases.tasks.post", &u.analyst1.ID, "103.21.58.91",
		map[string]any{"method": "POST", "status": 201, "case_id": c.kyc.ID, "task_title": "Verify Aadhaar via UIDAI eKYC API"}, "info")

	fmt.Println("  🔍 15 audit log entries created")
}
