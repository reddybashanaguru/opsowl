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

// daysAgo returns a UTC timestamp N days before now at the given hour:minute.
func daysAgo(days int, hour, min int) time.Time {
	base := time.Now().UTC().AddDate(0, 0, -days)
	return time.Date(base.Year(), base.Month(), base.Day(), hour, min, 0, 0, time.UTC)
}

func jsonb(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

var demoOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://opsowl:opsowldev@localhost:5433/opsowl?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	must(err)
	defer pool.Close()

	sqlDB := pgxstdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	q := dbgen.New(sqlDB)

	fmt.Println("🌱 Seeding OpsOwl demo data (1-month bank-scale dataset)...")
	truncate(sqlDB)
	u := seedUsers(ctx, q)
	cases := seedCases(ctx, q, u)
	seedTimeline(ctx, q, cases, u)
	seedTasks(ctx, q, cases, u)
	seedDocuments(ctx, q, cases, u)
	seedAILogs(ctx, q, cases, u)
	seedAuditLogs(ctx, q, u, cases)
	fmt.Println("✅ Done.")
	fmt.Println("   Login:   admin@opsowl.dev / OpsOwl@2024")
	fmt.Printf("   Cases:   %d total\n", len(cases))
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
	admin, manager, analyst1, analyst2, analyst3 dbgen.CreateUserRow
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
			Email: email, FullName: name, PasswordHash: hash, Role: role,
		})
		must(err)
		return u
	}
	s := userSet{
		admin:    mk("admin@opsowl.dev", "Priya Sharma", dbgen.UserRoleSuperAdmin),
		manager:  mk("ravi.kumar@opsowl.dev", "Ravi Kumar", dbgen.UserRoleManager),
		analyst1: mk("divya.reddy@opsowl.dev", "Divya Reddy", dbgen.UserRoleAnalyst),
		analyst2: mk("arjun.mehta@opsowl.dev", "Arjun Mehta", dbgen.UserRoleAnalyst),
		analyst3: mk("sneha.patel@opsowl.dev", "Sneha Patel", dbgen.UserRoleAnalyst),
	}
	fmt.Println("  👤 5 users created (password: OpsOwl@2024)")
	return s
}

// ── Cases ─────────────────────────────────────────────────────────────────────

func mkCase(ctx context.Context, q *dbgen.Queries,
	title, desc string, status dbgen.CaseStatus, prio dbgen.CasePriority,
	createdBy uuid.UUID, assignedTo *uuid.UUID,
	tags []string, daysAgoCreated int) dbgen.Case {

	var assigned uuid.NullUUID
	if assignedTo != nil {
		assigned = uuid.NullUUID{UUID: *assignedTo, Valid: true}
	}

	// SLA: critical=3d, high=7d, medium=14d, low=30d from creation
	slaMap := map[dbgen.CasePriority]int{
		dbgen.CasePriorityCritical: 3,
		dbgen.CasePriorityHigh:     7,
		dbgen.CasePriorityMedium:   14,
		dbgen.CasePriorityLow:      30,
	}
	createdAt := daysAgo(daysAgoCreated, 9, 30)
	slaDays := slaMap[prio]
	slaAt := createdAt.AddDate(0, 0, slaDays)
	dueAt := createdAt.AddDate(0, 0, slaDays-1)

	c, err := q.CreateCase(ctx, dbgen.CreateCaseParams{
		Title: title, Description: desc, Status: status, Priority: prio,
		CreatedBy: createdBy, OrganizationID: demoOrgID, AssignedTo: assigned,
		DueDate:     sql.NullTime{Time: dueAt, Valid: true},
		SlaBreachAt: sql.NullTime{Time: slaAt, Valid: true},
		Tags:        tags,
	})
	must(err)
	return c
}

func seedCases(ctx context.Context, q *dbgen.Queries, u userSet) []dbgen.Case {
	a1, a2, a3 := &u.analyst1.ID, &u.analyst2.ID, &u.analyst3.ID
	mgr := &u.manager.ID

	cases := []dbgen.Case{
		// ── UPI / IMPS Fraud ──────────────────────────────────────────────────────
		mkCase(ctx, q,
			"Suspicious UPI Cluster — Account 9876543210",
			"47 high-value UPI transactions totalling ₹42.3 lakh in 72 hours from account 9876543210. "+
				"Beneficiaries span HDFC, ICICI, Axis Bank with no prior transaction history. "+
				"Structuring pattern below ₹1L CTR threshold. Possible PMLA Schedule II layering offence.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityCritical, u.analyst1.ID, a2,
			[]string{"upi", "fraud", "layering", "critical"}, 28),

		mkCase(ctx, q,
			"UPI Mule Network — 12-Account Chain Detected",
			"Transaction monitoring flagged a 12-account UPI mule chain originating from a single source "+
				"account. ₹1.8 crore passed through in 5 days. Each intermediate account held funds for "+
				"<6 hours before onward transfer. Network graph analysis shows spoke-and-hub topology.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityHigh, u.manager.ID, a1,
			[]string{"upi", "mule-network", "aml", "layering"}, 15),

		mkCase(ctx, q,
			"IMPS Velocity Breach — Corporate Account XYZ-4491",
			"Corporate current account XYZ-4491 triggered 340 IMPS transactions in a single business day, "+
				"exceeding the soft limit by 17x. Single-beneficiary pattern (₹50K each) suggests automated "+
				"scripting or API-level abuse of limits. Account under SLA freeze watch.",
			dbgen.CaseStatusPendingReview, dbgen.CasePriorityHigh, u.analyst2.ID, mgr,
			[]string{"imps", "velocity", "corporate", "api-abuse"}, 10),

		mkCase(ctx, q,
			"UPI Phishing Refund Fraud — Customer CUST-3318",
			"Customer CUST-3318 reports UPI refund fraud: received spoofed VPA QR claiming to issue refund, "+
				"instead initiated ₹68,000 debit. Merchant callback URL shows phishing domain registered 3 days ago. "+
				"Reverse lookup confirms payment went to a dormant savings account.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityMedium, u.analyst3.ID, nil,
			[]string{"upi", "phishing", "customer-grievance", "qr"}, 22),

		// ── Card Fraud / Skimming ─────────────────────────────────────────────────
		mkCase(ctx, q,
			"ATM Skimming Device — Branch ATM BLR-ATM-044",
			"Physical inspection of BLR-ATM-044 (Koramangala branch) detected skimming device and pinhole "+
				"camera installed on card reader slot. Device operational for estimated 72 hours. "+
				"84 compromised cards identified via CBS card-usage correlation. FIR filed with Bengaluru Cyber Crime.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityCritical, u.manager.ID, a2,
			[]string{"atm", "skimming", "physical", "card-fraud", "fir"}, 18),

		mkCase(ctx, q,
			"Card-Not-Present Fraud — International CNP Cluster",
			"22 international CNP transactions on 6 cards totalling ₹12.4 lakh in 4 hours. "+
				"All transactions originate from same IP subnet (203.0.113.0/27). Cards were issued <30 days ago "+
				"to customers in the same PIN code — suggests data breach at a common merchant terminal.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.analyst1.ID, a3,
			[]string{"cnp", "card-fraud", "international", "data-breach"}, 12),

		mkCase(ctx, q,
			"Contactless Tap Fraud — Repeat Victim Pattern",
			"3 separate customers report unauthorised contactless transactions at the same merchant terminal "+
				"TERM-HYD-2219. Terminal logs show NFC read events not preceded by dip/swipe. "+
				"Relay attack suspected. Terminal replacement issued; forensic image preserved.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityMedium, u.analyst3.ID, nil,
			[]string{"contactless", "nfc", "relay-attack", "terminal"}, 25),

		// ── KYC / Identity Fraud ─────────────────────────────────────────────────
		mkCase(ctx, q,
			"KYC Document Forgery — Customer CUST-2024-0891",
			"Branch staff flagged potential forgery in Aadhaar and PAN submitted by CUST-2024-0891 during "+
				"loan onboarding. Hologram positioning and font inconsistencies noted. "+
				"UIDAI eKYC verification pending. Loan disbursement frozen.",
			dbgen.CaseStatusPendingReview, dbgen.CasePriorityHigh, u.analyst2.ID, a1,
			[]string{"kyc", "forgery", "compliance", "loan-onboarding"}, 20),

		mkCase(ctx, q,
			"Synthetic Identity Fraud — Account Cluster (5 Accounts)",
			"Credit bureau mismatch detected on 5 accounts opened in the same week using Aadhaar numbers "+
				"with identical date-of-birth pattern. DeepID score <0.3 for all 5. "+
				"Accounts show coordinated small-value deposit-and-withdraw cycles (warming pattern). "+
				"Consistent with synthetic identity ring preparing for bust-out fraud.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.analyst1.ID, a2,
			[]string{"synthetic-identity", "kyc", "bust-out", "aml"}, 9),

		mkCase(ctx, q,
			"Video KYC Deepfake Attempt — Account Opening",
			"AI liveness detection flagged a deepfake during Video KYC for new account opening application. "+
				"Facial movement artifacts and inconsistent lighting gradients detected. "+
				"Application rejected. Mobile number associated with 3 prior rejected applications at other banks "+
				"(cross-bureau alert via C-KYC registry).",
			dbgen.CaseStatusClosed, dbgen.CasePriorityHigh, u.analyst3.ID, nil,
			[]string{"vkyc", "deepfake", "synthetic-identity", "ai-detection"}, 30),

		// ── Loan / Credit Fraud ───────────────────────────────────────────────────
		mkCase(ctx, q,
			"Loan Disbursement Anomaly — Application LAF-2024-4412",
			"Loan of ₹18.5 lakh disbursed within 4 hours of approval, bypassing mandatory 48-hour cooling period. "+
				"Disbursement account differs from application. No maker-checker audit trail in CBS. "+
				"Possible insider threat or API-level business rule bypass.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityHigh, u.manager.ID, a1,
			[]string{"loan", "disbursement", "insider-threat", "anomaly"}, 17),

		mkCase(ctx, q,
			"Gold Loan Over-valuation — Branch MUM-019",
			"Valuator report for gold loan GL-2024-9901 (₹4.2 lakh) shows purity/weight figures 30% above "+
				"market benchmark for the collateral quantity. Same valuator involved in 3 prior over-valued loans "+
				"at MUM-019 in the past 6 months. Suspected collusion between valuator and customer ring.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.analyst2.ID, a3,
			[]string{"gold-loan", "valuation-fraud", "collateral", "insider"}, 14),

		mkCase(ctx, q,
			"Micro-Finance Round-Tripping — SHG Cluster (18 accounts)",
			"A self-help group cluster of 18 accounts shows round-tripping: MFI disbursements returned "+
				"as deposits to primary JLG account within 48 hours, only to be re-disbursed as new loans. "+
				"Net loan book inflation: ₹38.4 lakh. Possible evergreening/NPA suppression scheme.",
			dbgen.CaseStatusEscalated, dbgen.CasePriorityCritical, u.manager.ID, mgr,
			[]string{"microfinance", "round-tripping", "npa", "evergreening", "escalated"}, 7),

		// ── NACH / ECS Mandate Fraud ──────────────────────────────────────────────
		mkCase(ctx, q,
			"NACH Mandate Fraud — 340 Accounts Affected",
			"Fraudulent NACH mandates registered for 340 accounts using forged physical mandate forms. "+
				"Source UMRN batch uploaded by aggregator ID AGG-0442 which was deactivated 2 months ago. "+
				"₹1.04 crore in unauthorized debits across April cycle. Sponsor bank notified.",
			dbgen.CaseStatusEscalated, dbgen.CasePriorityCritical, u.manager.ID, a1,
			[]string{"nach", "mandate-fraud", "batch", "aggregator"}, 8),

		mkCase(ctx, q,
			"ECS Bounce Fraud — Coordinated Return Manipulation",
			"16 ECS mandates for corporate loans show coordinated insufficient-funds returns on the same day, "+
				"triggering penalty waiver requests 48 hours later. Pattern repeated over 3 consecutive cycles. "+
				"Suspected manual orchestration by branch staff to earn commissions on penalty reversals.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityHigh, u.analyst3.ID, a2,
			[]string{"ecs", "bounce", "insider", "penalty-fraud"}, 5),

		// ── Account Takeover (ATO) ────────────────────────────────────────────────
		mkCase(ctx, q,
			"Account Takeover via SIM Swap — Customer CUST-7721",
			"Customer CUST-7721 reported mobile banking access lost. Investigation confirms SIM swap at "+
				"Telecom operator 11 hours before ₹4.8 lakh NEFT transfer. OTP-based 2FA bypassed. "+
				"Customer's registered mobile was ported without physical presence verification by operator.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityHigh, u.analyst1.ID, nil,
			[]string{"ato", "sim-swap", "neft", "otp-bypass"}, 26),

		mkCase(ctx, q,
			"Credential Stuffing Attack — 89 Accounts Compromised",
			"Security monitoring detected 89 successful logins from a credential-stuffing attack using "+
				"a leaked credential database (sourced from third-party breach). "+
				"Attacker IPs: 3 Tor exit nodes, 2 residential proxies. "+
				"12 accounts had password same as email provider; 89 had profile edits before detection.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.manager.ID, a3,
			[]string{"ato", "credential-stuffing", "cyber", "data-breach"}, 11),

		mkCase(ctx, q,
			"Remote Access Trojan — Business Banking Customer",
			"Business customer (SME segment) reports ₹11.2 lakh RTGS transfer they did not initiate. "+
				"Forensic review of session logs shows transaction initiated via legitimate banking session "+
				"with no VPN/proxy. Device fingerprint unchanged but mouse movement pattern anomalous (RAT signature). "+
				"CERT-In notification filed.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.analyst2.ID, a1,
			[]string{"ato", "rat", "rtgs", "business-banking", "cert-in"}, 6),

		// ── Insider Threat ────────────────────────────────────────────────────────
		mkCase(ctx, q,
			"Insider Data Exfiltration — Branch Operations Staff",
			"DLP alert: branch ops staff member downloaded complete customer PII CSV (42,000 records) "+
				"to a personal USB drive 3 days before resignation. Data includes mobile, DOB, account numbers. "+
				"No business justification. Correlated with spike in phishing complaints from same locality 2 weeks later.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityCritical, u.manager.ID, mgr,
			[]string{"insider", "data-exfiltration", "pii", "dlp", "hr"}, 13),

		mkCase(ctx, q,
			"Relationship Manager — Unauthorised Account Access",
			"RM (Employee ID EMP-2248) accessed 127 accounts not in their portfolio over 3 weeks. "+
				"Access pattern: evenings and weekends, bypassing normal callback verification. "+
				"23 of those accounts subsequently had address/mobile updates without customer-initiated requests.",
			dbgen.CaseStatusPendingReview, dbgen.CasePriorityHigh, u.analyst3.ID, mgr,
			[]string{"insider", "rm", "unauthorized-access", "privilege-abuse"}, 16),

		// ── AML / Trade-Based Money Laundering ────────────────────────────────────
		mkCase(ctx, q,
			"Trade-Based Money Laundering — Import Over-invoicing",
			"Trade finance review flagged 8 import LCs for electronics components at 240% of market price. "+
				"Beneficiary entity in Dubai registered 60 days ago with no operating history. "+
				"Exporter and importer share beneficial owner (disclosed in UBO filing gap). "+
				"ED/FEMA notification under consideration.",
			dbgen.CaseStatusEscalated, dbgen.CasePriorityCritical, u.manager.ID, a2,
			[]string{"tblm", "trade-finance", "lc", "fema", "over-invoicing"}, 21),

		mkCase(ctx, q,
			"Shell Company Structuring — RTGS Layering Chain",
			"Three shell companies (MSME-registered, zero employees) received ₹2.8 crore via RTGS "+
				"from 41 different remitters in 7 days — all individuals, all <₹1L per transaction. "+
				"Funds onward transferred to single beneficial account in T+0. Classic integration stage. "+
				"STR filed with FIU-IND. ED liaison initiated.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.analyst1.ID, a3,
			[]string{"aml", "shell-company", "layering", "rtgs", "fiu"}, 19),

		// ── Merchant / Settlement ─────────────────────────────────────────────────
		mkCase(ctx, q,
			"Merchant Settlement Dispute — Batch NOV-2024-B88",
			"Merchant MCHT-9021 (QuickMart Retail) disputes short-settlement of ₹2.1 lakh in November batch. "+
				"Root cause: compound rounding bug in settlement engine v2.8.1 for international card transactions. "+
				"Hotfix v2.8.2 deployed. Manual credit of ₹2,10,000 issued. Case closed.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityMedium, u.analyst1.ID, nil,
			[]string{"merchant", "settlement", "dispute", "reconciliation"}, 29),

		mkCase(ctx, q,
			"Payment Aggregator Dispute — MDR Clawback Demand",
			"Payment aggregator PA-0077 demanding MDR clawback of ₹88,000 citing their ToS §14.3(b). "+
				"Bank's contract terms supersede. Legal review in progress. PA threatened to suspend payment "+
				"processing for 3 premium merchant clients. Business impact: ₹4.2 lakh/day pending resolution.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityMedium, u.analyst3.ID, mgr,
			[]string{"merchant", "aggregator", "legal", "mdr", "dispute"}, 4),

		// ── Cybersecurity ─────────────────────────────────────────────────────────
		mkCase(ctx, q,
			"Phishing Campaign — Branded SMS Spoofing (OpsOwl-HDFC Style)",
			"Bulk SMS campaign spoofing bank sender ID reaching ~12,000 customers with phishing link. "+
				"Link leads to pixel-perfect net banking clone. 89 customers clicked; 14 entered credentials. "+
				"Phishing domain reported to CERT-In and taken down within 4 hours. Customers under watch.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityHigh, u.analyst2.ID, nil,
			[]string{"phishing", "sms-spoofing", "cert-in", "cyber"}, 24),

		mkCase(ctx, q,
			"API Rate Limit Bypass — OTP Brute Force Attempt",
			"Automated attack attempted 4,800 OTP guesses against 120 account numbers via mobile banking API. "+
				"Rate-limiter triggered at 40 attempts/min but attacker distributed across 120 IPs. "+
				"12 accounts had their OTP lock triggered. SIEM correlation identified attack origin: "+
				"compromised IoT device network in Tier-2 cities.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityHigh, u.analyst1.ID, nil,
			[]string{"api", "otp", "brute-force", "cyber", "siem"}, 27),

		// ── Cheque / Instrument Fraud ─────────────────────────────────────────────
		mkCase(ctx, q,
			"Cheque Kiting — Interbank Float Exploitation",
			"Customer C-9812 exploited interbank clearing float by depositing outstation cheques (T+2 clearing) "+
				"and immediately withdrawing against uncleared funds. Net exposure: ₹8.7 lakh across 4 cycles. "+
				"Cheque issuing bank (other PSU) has been notified. Account frozen pending recovery.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityMedium, u.analyst3.ID, a2,
			[]string{"cheque", "kiting", "float", "interbank"}, 3),

		// ── Crypto On-Ramp ────────────────────────────────────────────────────────
		mkCase(ctx, q,
			"Crypto On-Ramp Suspicious Activity — P2P Transfers",
			"Account C-7744 shows recurring round-number NEFT transfers (₹50K, ₹1L) to 14 different "+
				"counterparties identified as P2P crypto marketplace collectors. "+
				"Volume: ₹34.2 lakh in 30 days. Counterparties registered in jurisdictions with FATF grey-list status. "+
				"Enhanced Due Diligence initiated.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityMedium, u.analyst2.ID, a1,
			[]string{"crypto", "p2p", "neft", "edd", "fatf"}, 2),

		// ── Dormant Account Activation ────────────────────────────────────────────
		mkCase(ctx, q,
			"Dormant Account Activation — Bulk Pattern (220 Accounts)",
			"220 accounts inactive for 2+ years activated simultaneously over 4 days via mobile banking. "+
				"Post-activation: immediate UPI-enable and small inward credit (₹1-5K) followed by "+
				"large outward transfer. Pattern consistent with mule account farming using stolen credentials. "+
				"Liaison with telecom for SIM ownership verification ongoing.",
			dbgen.CaseStatusEscalated, dbgen.CasePriorityHigh, u.manager.ID, a3,
			[]string{"dormant", "mule", "upi", "account-farming"}, 1),
	}

	fmt.Printf("  📁 %d cases created (OWL-2026-00001 … OWL-2026-%05d)\n", len(cases), len(cases))
	return cases
}

// ── Timeline Events ───────────────────────────────────────────────────────────

func emit(ctx context.Context, q *dbgen.Queries,
	caseID, actorID uuid.UUID, evtType dbgen.TimelineEventType,
	src dbgen.TimelineEventSource, at time.Time, payload any) {

	_, err := q.InsertTimelineEvent(ctx, dbgen.InsertTimelineEventParams{
		CaseID: caseID, EventType: evtType, Source: src,
		ActorID:    uuid.NullUUID{UUID: actorID, Valid: true},
		OccurredAt: at, Payload: jsonb(payload), Metadata: jsonb(map[string]any{}),
	})
	must(err)
}

func seedTimeline(ctx context.Context, q *dbgen.Queries, cases []dbgen.Case, u userSet) {
	actors := []uuid.UUID{u.analyst1.ID, u.analyst2.ID, u.analyst3.ID, u.manager.ID}
	ai := dbgen.TimelineEventSourceAi
	usr := dbgen.TimelineEventSourceUser

	total := 0
	for i, c := range cases {
		base := daysAgo(29-i, 9, 0)
		actor := actors[i%len(actors)]
		other := actors[(i+1)%len(actors)]

		// Every case: created
		emit(ctx, q, c.ID, actor, dbgen.TimelineEventTypeCaseCreated, usr, base,
			map[string]any{"case_number": c.CaseNumber, "title": c.Title, "priority": string(c.Priority)})
		total++

		// Assign (all except last 2)
		if i < len(cases)-2 {
			emit(ctx, q, c.ID, u.manager.ID, dbgen.TimelineEventTypeCaseAssigned, usr, base.Add(20*time.Minute),
				map[string]any{"assigned_to_name": "Divya Reddy", "assigned_by_name": "Ravi Kumar"})
			total++
		}

		// Document uploaded (2 per case)
		emit(ctx, q, c.ID, other, dbgen.TimelineEventTypeDocumentUploaded, usr, base.Add(2*time.Hour),
			map[string]any{"filename": fmt.Sprintf("evidence_%s.pdf", c.CaseNumber), "size_bytes": 512000 + int64(i)*12000})
		total++
		emit(ctx, q, c.ID, other, dbgen.TimelineEventTypeDocumentProcessed, ai, base.Add(2*time.Hour+4*time.Minute),
			map[string]any{"filename": fmt.Sprintf("evidence_%s.pdf", c.CaseNumber), "chunks": 40 + i*2, "status": "completed"})
		total++

		// AI query
		emit(ctx, q, c.ID, actor, dbgen.TimelineEventTypeAiQuery, ai, base.Add(3*time.Hour),
			map[string]any{"action_type": "ask", "model": "claude-opus-4-5", "latency_ms": 1800 + i*50})
		total++

		// Status change (open → in_progress or further)
		if c.Status != dbgen.CaseStatusDraft && c.Status != dbgen.CaseStatusOpen {
			emit(ctx, q, c.ID, actor, dbgen.TimelineEventTypeCaseStatusChanged, usr, base.Add(5*time.Hour),
				map[string]any{"from": "open", "to": string(c.Status)})
			total++
		}

		// Task created
		emit(ctx, q, c.ID, other, dbgen.TimelineEventTypeTaskCreated, usr, base.Add(6*time.Hour),
			map[string]any{"title": "Initial evidence collection and CBS log review"})
		total++

		// AI summary (for non-draft, non-open cases)
		if c.Status != dbgen.CaseStatusDraft && c.Status != dbgen.CaseStatusOpen {
			emit(ctx, q, c.ID, u.manager.ID, dbgen.TimelineEventTypeAiSummaryGenerated, ai, base.Add(8*time.Hour),
				map[string]any{"action_type": "summarize", "review_status": "approved", "model": "claude-opus-4-5"})
			total++
		}

		// Second document
		emit(ctx, q, c.ID, actor, dbgen.TimelineEventTypeDocumentUploaded, usr, base.Add(24*time.Hour),
			map[string]any{"filename": fmt.Sprintf("supplementary_%s.pdf", c.CaseNumber), "size_bytes": 245760})
		total++
		emit(ctx, q, c.ID, actor, dbgen.TimelineEventTypeDocumentProcessed, ai, base.Add(24*time.Hour+3*time.Minute),
			map[string]any{"filename": fmt.Sprintf("supplementary_%s.pdf", c.CaseNumber), "chunks": 22 + i, "status": "completed"})
		total++

		// Closed cases get a resolution event
		if c.Status == dbgen.CaseStatusClosed {
			emit(ctx, q, c.ID, u.manager.ID, dbgen.TimelineEventTypeCaseStatusChanged, usr, base.AddDate(0, 0, 3),
				map[string]any{"from": "in_progress", "to": "closed", "resolution": "Investigation complete, remediation actions taken"})
			total++
		}

		// Escalated cases get escalation note
		if c.Status == dbgen.CaseStatusEscalated {
			emit(ctx, q, c.ID, u.manager.ID, dbgen.TimelineEventTypeCaseStatusChanged, usr, base.Add(12*time.Hour),
				map[string]any{"from": "in_progress", "to": "escalated", "escalated_to": "Chief Risk Officer", "severity": "high"})
			total++
		}
	}

	// Add extra events for the first 3 flagship cases (more detailed trail)
	flagship := cases[0] // UPI cluster
	flagshipBase := daysAgo(28, 9, 0)
	extraEvents := []struct {
		evt     dbgen.TimelineEventType
		src     dbgen.TimelineEventSource
		offset  time.Duration
		actor   uuid.UUID
		payload map[string]any
	}{
		{dbgen.TimelineEventTypeAiQuery, ai, 4 * time.Hour, u.analyst2.ID,
			map[string]any{"action_type": "extract_actions", "suggestion_count": 5, "review_status": "approved"}},
		{dbgen.TimelineEventTypeTaskCreated, usr, 6*time.Hour + 30*time.Minute, u.analyst1.ID,
			map[string]any{"title": "File STR with FIU-IND (Form A)", "priority": "high"}},
		{dbgen.TimelineEventTypeTaskCreated, usr, 7 * time.Hour, u.analyst1.ID,
			map[string]any{"title": "Issue Section 12A PMLA freeze on 3 beneficiary accounts"}},
		{dbgen.TimelineEventTypeDocumentUploaded, usr, 30 * time.Hour, u.analyst2.ID,
			map[string]any{"filename": "str_form_a_draft.pdf", "size_bytes": 184320}},
		{dbgen.TimelineEventTypeAiQuery, ai, 32 * time.Hour, u.analyst2.ID,
			map[string]any{"action_type": "ask", "question": "STR narrative review", "latency_ms": 2100}},
	}
	for _, e := range extraEvents {
		emit(ctx, q, flagship.ID, e.actor, e.evt, e.src, flagshipBase.Add(e.offset), e.payload)
		total++
	}

	// Add today's events for the AI queries today metric
	todayBase := time.Now().UTC().Add(-3 * time.Hour)
	for i := 0; i < 3; i++ {
		c := cases[i%len(cases)]
		emit(ctx, q, c.ID, u.analyst1.ID, dbgen.TimelineEventTypeAiQuery, ai,
			todayBase.Add(time.Duration(i)*45*time.Minute),
			map[string]any{"action_type": "ask", "model": "claude-opus-4-5", "latency_ms": 1900})
		total++
	}

	fmt.Printf("  📅 %d timeline events created\n", total)
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

var taskTemplates = []struct {
	title, desc string
	prio        dbgen.TaskPriority
	dueHours    int
	isAI        bool
}{
	{"Pull CBS transaction logs and flag velocity anomalies",
		"Export from CBS_TXN_AUDIT table for the past 90 days. Apply velocity filter: >10 txns/hour or >₹1L/day.",
		dbgen.TaskPriorityCritical, 24, false},
	{"Send Letter of Cooperation to counterparty banks",
		"LOC format per RBI Master Direction. Addressed to all identified counterparty branches.",
		dbgen.TaskPriorityHigh, 72, false},
	{"Prepare Suspicious Transaction Report (STR) — FIU-IND Form A",
		"Deadline: 7 working days from detection. Attach CBS export and account profile.",
		dbgen.TaskPriorityHigh, 168, true},
	{"Conduct customer interview and collect statement",
		"Schedule via branch manager. Two-officer presence required. Record per policy.",
		dbgen.TaskPriorityMedium, 120, false},
	{"Cross-reference with CKYC registry and CIBIL alerts",
		"Check for prior fraud flags across industry-shared databases.",
		dbgen.TaskPriorityMedium, 96, true},
	{"Escalate to Compliance HOD for regulatory notification",
		"Prepare management summary. Attach evidence bundle. Regulatory deadline: 7 days.",
		dbgen.TaskPriorityCritical, 48, true},
	{"Preserve digital forensic evidence (disk image, logs)",
		"Use approved forensic toolkit. Chain of custody form required. Copy to secure evidence vault.",
		dbgen.TaskPriorityHigh, 48, false},
	{"Obtain independent valuation for collateral dispute",
		"Engage empanelled valuer. Separate from original branch valuator.",
		dbgen.TaskPriorityMedium, 72, false},
	{"File complaint with Cyber Crime portal and obtain FIR number",
		"Portal: cybercrime.gov.in. Attach all digital evidence. FIR number required for insurance claim.",
		dbgen.TaskPriorityHigh, 48, false},
	{"Notify affected customers and arrange credit reversal",
		"Customer communication per RBI grievance redressal timeline. Initiate credit within 24h of confirmation.",
		dbgen.TaskPriorityCritical, 24, false},
}

func seedTasks(ctx context.Context, q *dbgen.Queries, cases []dbgen.Case, u userSet) {
	actors := []uuid.UUID{u.analyst1.ID, u.analyst2.ID, u.analyst3.ID, u.manager.ID}
	total := 0

	for i, c := range cases {
		// 3–5 tasks per case
		numTasks := 3 + (i % 3)
		for j := 0; j < numTasks; j++ {
			tmpl := taskTemplates[(i*3+j)%len(taskTemplates)]
			creator := actors[i%len(actors)]
			assignee := actors[(i+j+1)%len(actors)]

			var due sql.NullTime
			if tmpl.dueHours > 0 {
				due = sql.NullTime{
					Time:  time.Now().UTC().Add(time.Duration(tmpl.dueHours-i*2) * time.Hour),
					Valid: true,
				}
			}

			var desc sql.NullString
			if tmpl.desc != "" {
				desc = sql.NullString{String: tmpl.desc, Valid: true}
			}

			_, err := q.CreateTask(ctx, dbgen.CreateTaskParams{
				CaseID:        c.ID,
				CreatedBy:     creator,
				AssignedTo:    uuid.NullUUID{UUID: assignee, Valid: true},
				Title:         tmpl.title,
				Description:   desc,
				Priority:      tmpl.prio,
				DueDate:       due,
				IsAiSuggested: tmpl.isAI,
			})
			must(err)
			total++
		}
	}
	fmt.Printf("  ✅ %d tasks created\n", total)
}

// ── Documents ─────────────────────────────────────────────────────────────────

var docTemplates = []struct {
	name     string
	fileType string
	mime     string
	size     int64
	status   dbgen.DocumentProcessingStatus
	pages    int
	chunks   int
}{
	{"cbs_transaction_export.pdf", "pdf", "application/pdf", 1245184, dbgen.DocumentProcessingStatusCompleted, 18, 92},
	{"account_statement_90days.pdf", "pdf", "application/pdf", 987136, dbgen.DocumentProcessingStatusCompleted, 12, 62},
	{"kyc_aadhaar_copy.pdf", "pdf", "application/pdf", 204800, dbgen.DocumentProcessingStatusCompleted, 2, 8},
	{"network_analysis_report.xlsx", "xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", 819200, dbgen.DocumentProcessingStatusCompleted, 4, 35},
	{"str_form_a_draft.pdf", "pdf", "application/pdf", 358400, dbgen.DocumentProcessingStatusCompleted, 6, 28},
	{"forensic_disk_image_manifest.txt", "txt", "text/plain", 4096, dbgen.DocumentProcessingStatusCompleted, 1, 2},
	{"cctv_stills_evidence.zip", "zip", "application/zip", 8388608, dbgen.DocumentProcessingStatusFailed, 0, 0},
	{"settlement_batch_reconciliation.csv", "csv", "text/csv", 65536, dbgen.DocumentProcessingStatusCompleted, 1, 18},
	{"customer_complaint_form.pdf", "pdf", "application/pdf", 102400, dbgen.DocumentProcessingStatusCompleted, 2, 6},
	{"valuation_report_independent.pdf", "pdf", "application/pdf", 614400, dbgen.DocumentProcessingStatusCompleted, 8, 40},
}

func seedDocuments(ctx context.Context, q *dbgen.Queries, cases []dbgen.Case, u userSet) {
	uploaders := []uuid.UUID{u.analyst1.ID, u.analyst2.ID, u.analyst3.ID}
	total := 0

	for i, c := range cases {
		numDocs := 2 + (i % 2)
		for j := 0; j < numDocs; j++ {
			tmpl := docTemplates[(i*2+j)%len(docTemplates)]
			uploader := uploaders[i%len(uploaders)]
			docID := uuid.New()

			// Customize filename with case number
			filename := fmt.Sprintf("%s_%s", c.CaseNumber, tmpl.name)

			d, err := q.CreateDocument(ctx, dbgen.CreateDocumentParams{
				ID:               docID,
				CaseID:           c.ID,
				UploadedBy:       uploader,
				OriginalFilename: filename,
				StoredPath:       fmt.Sprintf("cases/%s/docs/%s/%s", c.ID, docID, filename),
				FileType:         tmpl.fileType,
				MimeType:         tmpl.mime,
				FileSizeBytes:    tmpl.size,
			})
			must(err)

			if tmpl.status != dbgen.DocumentProcessingStatusPending {
				var pageCount, chunkCount sql.NullInt32
				var errMsg sql.NullString
				if tmpl.pages > 0 {
					pageCount = sql.NullInt32{Int32: int32(tmpl.pages), Valid: true}
					chunkCount = sql.NullInt32{Int32: int32(tmpl.chunks), Valid: true}
				}
				if tmpl.status == dbgen.DocumentProcessingStatusFailed {
					errMsg = sql.NullString{String: "OCR failed: unsupported binary format or corrupted file", Valid: true}
				}
				_, err = q.UpdateDocumentProcessingResult(ctx, dbgen.UpdateDocumentProcessingResultParams{
					ID: d.ID, ProcessingStatus: tmpl.status,
					PageCount: pageCount, ChunkCount: chunkCount, ErrorMessage: errMsg,
				})
				must(err)
			}
			total++
		}
	}
	fmt.Printf("  📄 %d documents created\n", total)
}

// ── AI Logs ───────────────────────────────────────────────────────────────────

var aiLogTemplates = []struct {
	actionType   string
	prompt       string
	response     string
	tokIn, tokOut int32
	latency      int32
	confidence   float64
	reviewStatus string // pending, approved, rejected, edited
}{
	{
		"ask",
		"What are the suspicious patterns in the transactions? Are there any AML red flags?",
		"Based on the transaction data analysed, three primary AML red flags are identified:\n\n" +
			"**1. Structuring:** Multiple transactions just below reporting thresholds (₹50K–₹95K range).\n" +
			"**2. Velocity:** Transaction frequency 12x above account's 90-day baseline.\n" +
			"**3. Mule Indicators:** Beneficiary accounts opened <60 days ago with minimum KYC.\n\n" +
			"**Recommendation:** File STR within 7 working days. Consider account freeze pending investigation.",
		1820, 340, 2340, 0.87, "approved",
	},
	{
		"summarize",
		"[SYSTEM: Generate a comprehensive investigation case summary from all available evidence]",
		"**Case Summary**\n\nThis case involves potential money laundering activity. " +
			"Transaction pattern analysis confirms structuring behaviour with high confidence. " +
			"Account shows dormant-to-active pattern consistent with mule account farming. " +
			"Regulatory reporting timelines must be adhered to strictly.",
		2100, 280, 3100, 0.91, "approved",
	},
	{
		"extract_actions",
		"[SYSTEM: Extract actionable investigation steps from the case documents and context]",
		"Recommended investigation actions:\n\n" +
			"1. **File STR with FIU-IND** (Critical) — 7 working day deadline\n" +
			"2. **Obtain bank statements** from counterparty banks via LOC (High)\n" +
			"3. **Verify KYC documents** via UIDAI eKYC API (High)\n" +
			"4. **Preserve digital evidence** — CBS logs, CCTV, device fingerprints (High)\n" +
			"5. **Escalate to Compliance HOD** with management summary (Medium)",
		1680, 310, 2050, 0.83, "pending",
	},
	{
		"ask",
		"Is there any evidence of insider involvement? What indicators should I look for?",
		"Based on the access logs reviewed, two patterns suggest possible insider facilitation:\n\n" +
			"**1. Off-hours access:** Critical transactions processed 11 PM – 2 AM IST without override audit.\n" +
			"**2. Permission escalation:** Normal-role user accessed elevated transaction approval APIs " +
			"3 hours before the suspicious transfer — this requires maker-checker bypass authorization.\n\n" +
			"**Note:** These are indicators only. HR and legal must be involved before any personnel action.",
		1540, 220, 1890, 0.68, "pending",
	},
	{
		"summarize",
		"[SYSTEM: Generate case summary focusing on regulatory compliance requirements]",
		"**Compliance Summary**\n\nThis case triggers the following regulatory reporting obligations:\n" +
			"• FIU-IND STR (PMLA §12) — 7 working day deadline from date of knowledge\n" +
			"• RBI Cyber Incident Report (if cyber element present) — 2 hours initial, 6 hours detailed\n" +
			"• CERT-In Notification — within 6 hours of detection\n\nAll documentation requirements met per compliance checklist.",
		2300, 290, 3450, 0.89, "edited",
	},
}

func seedAILogs(ctx context.Context, q *dbgen.Queries, cases []dbgen.Case, u userSet) {
	actors := []uuid.UUID{u.analyst1.ID, u.analyst2.ID, u.analyst3.ID}
	reviewers := []uuid.UUID{u.manager.ID, u.admin.ID}
	total := 0

	for i, c := range cases {
		numLogs := 1 + (i % 2) // 1 or 2 logs per case
		for j := 0; j < numLogs; j++ {
			tmpl := aiLogTemplates[(i+j)%len(aiLogTemplates)]
			actor := actors[(i+j)%len(actors)]
			reviewer := reviewers[i%len(reviewers)]

			confScore := sql.NullString{}
			if tmpl.confidence > 0 {
				confScore = sql.NullString{String: fmt.Sprintf("%.2f", tmpl.confidence), Valid: true}
			}

			citations := jsonb([]map[string]any{{
				"doc_id": "seed-doc", "doc_name": fmt.Sprintf("evidence_%s.pdf", c.CaseNumber),
				"chunk_index": 3, "page_number": 4,
				"excerpt":    "Transaction pattern shows velocity anomaly exceeding 10x baseline.",
				"similarity": 0.88 + float64(i%3)*0.02,
			}})

			row, err := q.CreateAILog(ctx, dbgen.CreateAILogParams{
				CaseID: c.ID, TriggeredBy: actor,
				ActionType: tmpl.actionType, PromptText: tmpl.prompt,
				ResponseText: tmpl.response, ModelUsed: "claude-opus-4-5",
				Provider: "anthropic", TokensInput: tmpl.tokIn, TokensOutput: tmpl.tokOut,
				LatencyMs: tmpl.latency, ConfidenceScore: confScore, Citations: citations,
			})
			must(err)
			total++

			switch tmpl.reviewStatus {
			case "approved":
				_, err = q.ApproveAILog(ctx, dbgen.ApproveAILogParams{
					ID:         row.ID,
					ReviewedBy: uuid.NullUUID{UUID: reviewer, Valid: true},
				})
				must(err)
			case "edited":
				_, err = q.EditAILog(ctx, dbgen.EditAILogParams{
					ID:         row.ID,
					ReviewedBy: uuid.NullUUID{UUID: reviewer, Valid: true},
					HumanEditedResponse: sql.NullString{
						String: tmpl.response + "\n\n[Edited by reviewer: Added regulatory deadline specifics and cross-referenced with compliance checklist v2.4]",
						Valid:  true,
					},
				})
				must(err)
			}
			// "pending" and "rejected" left as-is (pending = no further action needed)
		}
	}

	// Add today's AI logs specifically for the "AI Queries Today" dashboard metric
	todayCases := cases[:5]
	for _, c := range todayCases {
		_, err := q.CreateAILog(ctx, dbgen.CreateAILogParams{
			CaseID:      c.ID,
			TriggeredBy: u.analyst1.ID,
			ActionType:  "ask",
			PromptText:  "Summarise latest developments in this case.",
			ResponseText: "Based on the most recent evidence:\n\n" +
				"1. CBS logs confirm the suspicious transaction pattern.\n" +
				"2. Counterparty bank has responded to the LOC request.\n" +
				"3. STR draft is ready for final review and submission.",
			ModelUsed:       "claude-opus-4-5",
			Provider:        "anthropic",
			TokensInput:     1200,
			TokensOutput:    180,
			LatencyMs:       1650,
			ConfidenceScore: sql.NullString{String: "0.85", Valid: true},
			Citations:       jsonb([]map[string]any{}),
		})
		must(err)
		total++
	}

	fmt.Printf("  🤖 %d AI log entries created (%d today)\n", total, len(todayCases))
}

// ── Audit Logs ────────────────────────────────────────────────────────────────

func seedAuditLogs(ctx context.Context, q *dbgen.Queries, u userSet, cases []dbgen.Case) {
	write := func(code string, actor *uuid.UUID, ip, sev string, meta map[string]any) {
		var actorNull uuid.NullUUID
		if actor != nil {
			actorNull = uuid.NullUUID{UUID: *actor, Valid: true}
		}
		err := q.CreateAuditLog(ctx, dbgen.CreateAuditLogParams{
			EventCode: code, ActorID: actorNull,
			IpAddress: sql.NullString{String: ip, Valid: true},
			Metadata:  jsonb(meta), Severity: sev,
		})
		must(err)
	}

	ips := []string{"203.192.18.44", "103.21.58.91", "103.21.58.92", "10.0.1.15", "198.51.100.5"}

	// Auth events
	write("auth.login.success", &u.admin.ID, ips[0], "info",
		map[string]any{"email": "admin@opsowl.dev", "user_agent": "Mozilla/5.0 Chrome/124", "session_id": "sess-001"})
	write("auth.login.success", &u.manager.ID, ips[3], "info",
		map[string]any{"email": "ravi.kumar@opsowl.dev", "session_id": "sess-002"})
	write("auth.login.success", &u.analyst1.ID, ips[1], "info",
		map[string]any{"email": "divya.reddy@opsowl.dev", "session_id": "sess-003"})
	write("auth.login.success", &u.analyst2.ID, ips[2], "info",
		map[string]any{"email": "arjun.mehta@opsowl.dev", "session_id": "sess-004"})
	write("auth.login.success", &u.analyst3.ID, ips[1], "info",
		map[string]any{"email": "sneha.patel@opsowl.dev", "session_id": "sess-005"})

	// Failed logins (security events)
	write("auth.login.failed", nil, "185.220.101.42", "warn",
		map[string]any{"email": "admin@opsowl.dev", "reason": "bad_password", "attempt": 1})
	write("auth.login.failed", nil, "185.220.101.42", "warn",
		map[string]any{"email": "admin@opsowl.dev", "reason": "bad_password", "attempt": 2})
	write("auth.token.reuse_detected", &u.analyst2.ID, "198.51.100.5", "critical",
		map[string]any{"email": "arjun.mehta@opsowl.dev", "rt_hash": "a3f9b1c2...", "security_action": "session_revoked"})
	write("auth.login.failed", nil, "91.108.4.11", "warn",
		map[string]any{"email": "sneha.patel@opsowl.dev", "reason": "account_locked", "attempts": 5})

	// Case mutations
	for i := 0; i < len(cases) && i < 10; i++ {
		c := cases[i]
		actor := []uuid.UUID{u.analyst1.ID, u.analyst2.ID, u.analyst3.ID, u.manager.ID}[i%4]
		write("api.cases.post", &actor, ips[i%len(ips)], "info",
			map[string]any{"method": "POST", "path": "/api/v1/cases", "status": 201, "case_id": c.ID, "case_number": c.CaseNumber})
	}

	// Document uploads
	write("api.cases.documents.post", &u.analyst1.ID, ips[1], "info",
		map[string]any{"method": "POST", "status": 201, "filename": "cbs_transaction_export.pdf", "size_bytes": 1245184})
	write("api.cases.documents.post", &u.analyst2.ID, ips[2], "info",
		map[string]any{"method": "POST", "status": 201, "filename": "network_analysis_report.xlsx", "size_bytes": 819200})
	write("api.cases.documents.post", &u.analyst3.ID, ips[1], "info",
		map[string]any{"method": "POST", "status": 201, "filename": "forensic_disk_image_manifest.txt", "size_bytes": 4096})

	// AI interactions
	write("api.cases.ai.ask.post", &u.analyst1.ID, ips[1], "info",
		map[string]any{"method": "POST", "status": 200, "action_type": "ask", "latency_ms": 2340})
	write("api.cases.ai.ask.post", &u.analyst2.ID, ips[2], "info",
		map[string]any{"method": "POST", "status": 200, "action_type": "summarize", "latency_ms": 3100})
	write("api.ai.logs.approve.post", &u.manager.ID, ips[3], "info",
		map[string]any{"method": "POST", "status": 200, "action": "approved", "reviewer": "ravi.kumar@opsowl.dev"})
	write("api.ai.logs.reject.post", &u.manager.ID, ips[3], "warn",
		map[string]any{"method": "POST", "status": 200, "action": "rejected", "reason": "hallucinated measurements"})
	write("api.ai.logs.edit.post", &u.manager.ID, ips[3], "info",
		map[string]any{"method": "POST", "status": 200, "action": "edited", "reason": "added regulatory context"})

	// Task mutations
	write("api.cases.tasks.post", &u.analyst1.ID, ips[1], "info",
		map[string]any{"method": "POST", "status": 201, "task_title": "File STR with FIU-IND (Form A)"})
	write("api.cases.tasks.patch", &u.analyst2.ID, ips[2], "info",
		map[string]any{"method": "PATCH", "status": 200, "action": "status_update", "to": "done"})
	write("api.cases.tasks.patch", &u.analyst1.ID, ips[1], "info",
		map[string]any{"method": "PATCH", "status": 200, "action": "status_update", "to": "in_progress"})

	// User management
	write("api.users.patch", &u.admin.ID, ips[0], "info",
		map[string]any{"method": "PATCH", "status": 200, "action": "role_change", "target_email": "sneha.patel@opsowl.dev", "new_role": "analyst"})

	// Status changes
	write("api.cases.status.patch", &u.manager.ID, ips[3], "info",
		map[string]any{"method": "PATCH", "status": 200, "from": "open", "to": "escalated", "case_number": cases[0].CaseNumber})
	write("api.cases.status.patch", &u.manager.ID, ips[3], "info",
		map[string]any{"method": "PATCH", "status": 200, "from": "in_progress", "to": "closed", "case_number": cases[3].CaseNumber})

	// Error events
	write("api.cases.ai.ask.post", &u.analyst3.ID, ips[4], "warn",
		map[string]any{"method": "POST", "status": 429, "error": "rate_limited", "retry_after": 60})
	write("api.cases.documents.post", &u.analyst2.ID, ips[2], "warn",
		map[string]any{"method": "POST", "status": 413, "error": "file_too_large", "size_bytes": 52428800})

	fmt.Println("  🔍 35 audit log entries created")
}
