package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// seedEUUsers creates a EU-named team (same emails/password as India seed for demo consistency).
func seedEUUsers(ctx context.Context, q *dbgen.Queries) userSet {
	hash := hashPwd("OpsOwl@2024")
	mk := func(email, name string, role dbgen.UserRole) dbgen.CreateUserRow {
		u, err := q.CreateUser(ctx, dbgen.CreateUserParams{
			Email: email, FullName: name, PasswordHash: hash, Role: role,
		})
		must(err)
		return u
	}
	s := userSet{
		admin:    mk("admin@opsowl.dev", "Aistė Bernotienė", dbgen.UserRoleSuperAdmin),
		manager:  mk("ravi.kumar@opsowl.dev", "Elena Stylianou", dbgen.UserRoleManager),
		analyst1: mk("divya.reddy@opsowl.dev", "Mantas Žukauskas", dbgen.UserRoleAnalyst),
		analyst2: mk("arjun.mehta@opsowl.dev", "Rūta Petrauskaitė", dbgen.UserRoleAnalyst),
		analyst3: mk("sneha.patel@opsowl.dev", "Andreas Papadopoulos", dbgen.UserRoleAnalyst),
	}
	fmt.Println("  👤 5 EU users created (password: OpsOwl@2024)")
	return s
}

// seedEUCases creates EU-context cases: SEPA/SWIFT, EUR amounts, Lithuanian EMI + Cypriot FX broker scenarios.
func seedEUCases(ctx context.Context, q *dbgen.Queries, u userSet) []dbgen.Case {
	a1, a2, a3 := &u.analyst1.ID, &u.analyst2.ID, &u.analyst3.ID
	mgr := &u.manager.ID

	cases := []dbgen.Case{
		// ── SEPA / EMI Fraud (Lithuania — FNTT) ───────────────────────────────────
		mkCase(ctx, q,
			"Suspicious SEPA Instant Cluster — EMI Account LT-EMI-7741",
			"23 SEPA Instant transactions totalling €85,000 in 48 hours from EMI account LT-EMI-7741. "+
				"Beneficiaries span IBANs in Latvia (LV), Estonia (EE), and Poland (PL) with no prior "+
				"transaction history. All transactions structured below the €10,000 EDD threshold. "+
				"Pattern consistent with layering via EU e-money accounts. FNTT STR deadline: 3 working days.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityCritical, u.analyst1.ID, a2,
			[]string{"sepa", "emi", "layering", "fntt", "critical"}, 2),

		mkCase(ctx, q,
			"SEPA Mule Network — 8 Coordinated EMI Accounts",
			"Transaction monitoring flagged 8 e-money accounts at the same Lithuanian EMI showing "+
				"coordinated SEPA Instant transfers. €320,000 passed through in 48 hours — each account "+
				"held funds for under 4 hours before onward transfer to a single Cypriot IBAN. "+
				"All 8 accounts opened within the same 2-week window with minimum CDD documentation.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityHigh, u.manager.ID, a1,
			[]string{"sepa", "mule-network", "emi", "layering", "fntt"}, 10),

		mkCase(ctx, q,
			"SEPA Velocity Breach — Corporate IBAN LT-CORP-9921",
			"Lithuanian corporate account LT-CORP-9921 triggered 94 SEPA Instant transactions totalling "+
				"€180,000 in 6 hours — exceeding velocity limits by 22x. All transfers to a single "+
				"beneficiary IBAN in Malta (MT). No invoicing or contract documentation on file. "+
				"Account under SLA watch. EDD required under AMLR 2024 Art. 21.",
			dbgen.CaseStatusPendingReview, dbgen.CasePriorityHigh, u.analyst2.ID, mgr,
			[]string{"sepa", "velocity", "edd", "amlr", "corporate"}, 8),

		// ── UBO / Offshore Structure (Cyprus — MOKAS) ─────────────────────────────
		mkCase(ctx, q,
			"UBO Non-Disclosure — Offshore Holding Structure (FX Broker)",
			"Cypriot FX broker client declared nominee director as UBO. Corporate structure shows: "+
				"operating entity (Cyprus) → BVI holding company → Seychelles trust → beneficial owner "+
				"(Russian national). AMLR 2024 Art. 42 requires verification to natural person level. "+
				"Nominee arrangement does not satisfy Art. 42(3). MOKAS notification under review. "+
				"CySEC may require licence suspension pending UBO resolution.",
			dbgen.CaseStatusEscalated, dbgen.CasePriorityCritical, u.manager.ID, mgr,
			[]string{"ubo", "offshore", "fx-broker", "mokas", "cysec", "amlr", "escalated"}, 15),

		mkCase(ctx, q,
			"UBO Discrepancy — Registry vs Declared Ownership",
			"Subject Costas Georgiou declared 20% ownership in onboarding CDD. Cyprus Registrar of "+
				"Companies shows 47% direct shareholding. Discrepancy of 27 percentage points — UBO "+
				"threshold breach. AMLR 2024 Art. 42 requires institutions to report discrepancies to "+
				"the relevant authority. Enhanced due diligence initiated. Business relationship suspended.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.analyst3.ID, a1,
			[]string{"ubo", "discrepancy", "registry", "mokas", "edd"}, 12),

		// ── Sanctions / PEP Hits ──────────────────────────────────────────────────
		mkCase(ctx, q,
			"EU Consolidated List Match — SEPA Payment Blocked",
			"Outbound SEPA credit transfer of €47,500 to beneficiary 'Dmitri Volkov Holding GmbH' "+
				"flagged at 87% fuzzy match against EU Consolidated Sanctions List (Council Regulation "+
				"(EU) 2022/259). Payment blocked automatically. Investigation opened. "+
				"FNTT notification required within 24 hours of confirmed match. "+
				"Correspondent bank in Germany notified per SWIFT gpi freeze protocol.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityCritical, u.analyst1.ID, a2,
			[]string{"sanctions", "eu-consolidated", "sepa", "fntt", "blocked"}, 1),

		mkCase(ctx, q,
			"PEP Account — Undisclosed Political Exposure (Lithuanian MEP Associate)",
			"Account holder Maria Kazlauskienė identified as close associate of Lithuanian MEP "+
				"Tomas Jankauskas (disclosed in EU Parliament transparency register). "+
				"Account was opened without PEP declaration. Enhanced due diligence was not applied. "+
				"AMLR 2024 Art. 33 breach — EDD mandatory for PEPs and their close associates. "+
				"Account frozen pending EDD completion and senior management approval.",
			dbgen.CaseStatusPendingReview, dbgen.CasePriorityHigh, u.analyst2.ID, mgr,
			[]string{"pep", "edd", "amlr", "fntt", "political-exposure"}, 9),

		// ── GDPR vs AMLR Conflict ─────────────────────────────────────────────────
		mkCase(ctx, q,
			"GDPR Art.17 Erasure Request — Active STR Investigation Conflict",
			"Data subject Eglė Kazlauskienė submitted GDPR Art. 17 right-to-erasure request. "+
				"Subject is the account holder in an active STR filed with FNTT. "+
				"GDPR Art. 17(3)(e) allows refusal for legal obligations — but AMLR 2024 Art. 77 "+
				"mandates 5-year retention. Resolution: pseudonymisation of PII fields. "+
				"Transaction records and AML investigation data retained. DPO sign-off required.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityMedium, u.analyst3.ID, mgr,
			[]string{"gdpr", "erasure", "pseudonymisation", "amlr", "dpo", "fntt"}, 7),

		// ── Crypto / CASP (MiCA) ──────────────────────────────────────────────────
		mkCase(ctx, q,
			"Crypto Asset Layering — CASP On-Ramp (MiCA Entity)",
			"MiCA-registered CASP flagged customer Andrius Stankevičius converting €47,500 cash "+
				"to BTC across 9 separate transactions in 5 days. Each transaction below €5,000 "+
				"to avoid CASP CDD thresholds. Funds subsequently split across 14 self-custody wallets. "+
				"FATF Travel Rule: CASP-to-CASP transfer records missing for 6 of 14 wallet transfers. "+
				"FNTT STR filed. Blockchain analytics report attached.",
			dbgen.CaseStatusInProgress, dbgen.CasePriorityHigh, u.analyst2.ID, a3,
			[]string{"crypto", "casp", "mica", "travel-rule", "fntt", "layering"}, 14),

		// ── Trade Finance / FX ────────────────────────────────────────────────────
		mkCase(ctx, q,
			"Trade-Based ML — FX Over-invoicing via Correspondent Account",
			"Cypriot FX broker client facilitated €250,000 in commodity trades at 310% above market "+
				"reference price. Correspondent account in Germany used for settlement. "+
				"Beneficial owner of importer entity matches partial name on EU Consolidated List "+
				"(68% fuzzy match — below auto-block threshold but above manual review threshold). "+
				"MOKAS STR filed. CySEC notified. ED referral under consideration.",
			dbgen.CaseStatusEscalated, dbgen.CasePriorityCritical, u.manager.ID, a2,
			[]string{"tblm", "fx-broker", "over-invoicing", "mokas", "cysec", "escalated"}, 20),

		// ── goAML Filing ──────────────────────────────────────────────────────────
		mkCase(ctx, q,
			"goAML Submission Rejected — FNTT Validation Error",
			"STR filing for case OWL-2026-00003 rejected by FNTT goAML portal. "+
				"Validation error: mandatory field 'reporting_entity.registration_number' missing. "+
				"EMI operating licence number required in EntityType.LicenceNumber field. "+
				"Re-submission deadline: 24 hours per FNTT operational guidance. "+
				"Compliance team coordinating with IT to update goAML entity configuration.",
			dbgen.CaseStatusOpen, dbgen.CasePriorityMedium, u.analyst1.ID, a1,
			[]string{"goaml", "fntt", "str", "validation", "resubmission"}, 5),

		// ── KYC / Identity ────────────────────────────────────────────────────────
		mkCase(ctx, q,
			"KYC Forgery — EU Residence Permit (Non-EU National)",
			"Third-country national submitted Lithuanian residence permit during EMI onboarding. "+
				"Biometric chip NFC verification failed — data on chip does not match printed document. "+
				"Document number cross-referenced with Migration Department database: not found. "+
				"AMLR 2024 Art. 21 EDD obligation triggered. Account not opened. "+
				"FNTT STR filed for attempted identity fraud. Passport number flagged in shared FIU database.",
			dbgen.CaseStatusClosed, dbgen.CasePriorityHigh, u.analyst3.ID, nil,
			[]string{"kyc", "forgery", "residence-permit", "nfc", "fntt", "edd"}, 22),

		// ── Dormant Account ───────────────────────────────────────────────────────
		mkCase(ctx, q,
			"Dormant EMI Account Reactivation — Coordinated EU Mule Pattern",
			"45 dormant EU e-money accounts reactivated via mobile app across 3 days. "+
				"Post-reactivation: each account received one SEPA Instant credit (€500–€2,000) "+
				"followed by immediate transfer to a single Maltese IBAN MT89X000000000000001230000789. "+
				"Pattern matches FNTT 2025 typology alert: cross-border mule farming using dormant EMI accounts. "+
				"All 45 accounts suspended. Bulk STR filed with FNTT.",
			dbgen.CaseStatusEscalated, dbgen.CasePriorityHigh, u.manager.ID, a3,
			[]string{"dormant", "mule", "sepa", "emi", "fntt", "account-farming"}, 3),
	}

	fmt.Printf("  📁 %d EU cases created (OWL-2026-00001 … OWL-2026-%05d)\n", len(cases), len(cases))
	return cases
}

// seedEUTasks creates EU-appropriate investigation tasks.
func seedEUTasks(ctx context.Context, q *dbgen.Queries, cases []dbgen.Case, u userSet) {
	actors := []uuid.UUID{u.analyst1.ID, u.analyst2.ID, u.analyst3.ID, u.manager.ID}

	euTaskTemplates := []struct {
		title, desc string
		prio        dbgen.TaskPriority
		dueHours    int
		isAI        bool
	}{
		{"Pull SEPA transaction data and flag velocity anomalies",
			"Export SEPA Instant and SEPA Credit Transfer records for the past 90 days. Apply velocity filter: >10 txns/hour or >€5,000/transaction below EDD threshold.",
			dbgen.TaskPriorityCritical, 24, false},
		{"Verify beneficiary IBANs against EU Consolidated Sanctions List",
			"Run all beneficiary IBANs and entity names against EU Consolidated List, UN Security Council, and OFAC SDN. Flag matches ≥75% similarity.",
			dbgen.TaskPriorityHigh, 12, true},
		{"File STR with FNTT via goAML XML export",
			"FNTT deadline: 3 working days from date of suspicion. Complete goAML Form, export XML, submit via goaml.fntt.lt. Retain submission receipt.",
			dbgen.TaskPriorityCritical, 72, true},
		{"Verify UBO to natural person level — AMLR 2024 Art. 42",
			"Identify all beneficial owners ≥25%. Cross-reference with Cyprus Registrar / Registrų Centras. Flag any nominee arrangements. Case cannot close until at least one UBO is Verified.",
			dbgen.TaskPriorityHigh, 48, false},
		{"Request EDD documentation from customer",
			"Source of funds statement, business profile, latest audited accounts. Deadline: 5 working days. Suspend business relationship if not received.",
			dbgen.TaskPriorityHigh, 120, false},
		{"Escalate to Senior Management — AMLR 2024 Art. 33 (PEP)",
			"PEP or PEP associate requires senior management approval to continue or establish business relationship. Prepare management briefing note.",
			dbgen.TaskPriorityCritical, 48, true},
		{"Preserve digital evidence — SEPA transaction logs and device fingerprints",
			"Export SEPA transaction logs, device fingerprint records, and IP geolocation data. Store in case documents folder. Chain of custody form required.",
			dbgen.TaskPriorityHigh, 24, false},
		{"Cross-reference with MOKAS / FNTT shared typology database",
			"Check subject names and IBANs against national FIU intelligence database. Flag any prior STR matches.",
			dbgen.TaskPriorityMedium, 96, true},
		{"Notify correspondent bank — SWIFT freeze notification",
			"Send SWIFT MT199 freeze notification to correspondent bank per FATF Recommendation 16. Retain acknowledgement.",
			dbgen.TaskPriorityHigh, 24, false},
		{"Process GDPR Art. 17 erasure request — pseudonymisation workflow",
			"Subject PII to be pseudonymised. Transaction and AML records retained per AMLR 2024 Art. 77 (5-year retention). DPO sign-off required before processing.",
			dbgen.TaskPriorityMedium, 720, false},
	}

	total := 0
	for i, c := range cases {
		numTasks := 3 + (i % 3)
		for j := 0; j < numTasks; j++ {
			tmpl := euTaskTemplates[(i*3+j)%len(euTaskTemplates)]
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
	fmt.Printf("  ✅ %d EU tasks created\n", total)
}

// seedEUNotes creates EU-context investigation notes.
func seedEUNotes(ctx context.Context, q *dbgen.Queries, cases []dbgen.Case, u userSet) {
	type noteSpec struct {
		caseIdx     int
		author      dbgen.CreateUserRow
		title       string
		content     string
		contentText string
		noteType    dbgen.NoteTypeEnum
		isPinned    bool
		aiGenerated bool
	}

	notes := []noteSpec{
		{0, u.analyst1, "Initial SEPA Cluster Analysis",
			`<p>Transaction analysis complete. Key findings:</p><ul><li>23 SEPA Instant transactions, €85,000 total, 48-hour window</li><li>Beneficiary IBANs: LV (8), EE (7), PL (8) — no prior relationship history</li><li>All transactions structured below €10,000 EDD threshold — classic placement structuring</li><li>Account velocity: 18x above 30-day SEPA baseline for this EMI segment</li></ul><p>FNTT STR deadline: Day 3. Draft STR initiated.</p>`,
			"SEPA cluster: €85,000, 23 txns, structured below €10,000 EDD threshold. FNTT STR Day 3.",
			dbgen.NoteTypeEnumInternal, true, false},

		{0, u.manager, "STR Escalation — FNTT Filing Approved",
			`<p>Reviewed analyst findings. Escalating to STR filing:</p><ol><li>Structuring below EDD threshold confirmed</li><li>Beneficiary IBANs cross-referenced — no matching EU sanctions hits</li><li>FATF typology: ML via EU e-money accounts (layering stage)</li></ol><p>goAML XML export to FNTT authorised. Filing must be submitted before EOD today — <strong>Day 3 of 3</strong>.</p>`,
			"STR escalation approved. goAML XML export to FNTT authorised. Day 3 deadline.",
			dbgen.NoteTypeEnumFormal, true, false},

		{1, u.analyst2, "UBO Structure Investigation — Offshore Chain",
			`<p>UBO verification findings:</p><ul><li>Declared UBO: nominee director (Seychelles registered agent)</li><li>Layer 1: Cyprus operating entity → BVI holding (100% same beneficial owner)</li><li>Layer 2: BVI holding → Seychelles trust → natural person</li><li>Natural person: Russian national, not on EU Consolidated List (as of screening date)</li></ul><p>AMLR 2024 Art. 42(3): nominee arrangements do not satisfy UBO identification. Enhanced due diligence required. MOKAS notification under review.</p>`,
			"UBO structure: 3-layer offshore chain. Nominee director does not satisfy AMLR Art. 42(3). MOKAS notification under review.",
			dbgen.NoteTypeEnumInternal, false, false},

		{2, u.analyst1, "EU Consolidated Sanctions Match — Preliminary Assessment",
			`<p>Sanctions screening result:</p><ul><li>Beneficiary name: Dmitri Volkov Holding GmbH</li><li>EU Consolidated List match: <strong>87% similarity</strong> — Dmitri Volkov (Program: Russian sectoral sanctions, Council Regulation 2022/259)</li><li>SEPA payment of €47,500 auto-blocked by sanctions screening engine</li></ul><p>Next steps: Confirm or dismiss hit. If confirmed — FNTT notification within 24 hours. Correspondent bank in Germany already notified via SWIFT MT199.</p>`,
			"EU Consolidated List 87% match on beneficiary. SEPA payment blocked. FNTT 24-hour notification clock started.",
			dbgen.NoteTypeEnumInternal, true, false},

		{3, u.manager, "GDPR Art.17 — Pseudonymisation Decision",
			`<h3>Legal Basis Assessment</h3><p>GDPR Art. 17(3)(e) permits refusal where processing is necessary for compliance with a legal obligation. AMLR 2024 Art. 77 mandates 5-year retention.</p><h3>Resolution</h3><p>Pseudonymisation applied: subject PII (name, IBAN, ID number, address, contact details) replaced with cryptographic pseudonym [ANON-E4F9B12A]. Transaction records, STR filing, and investigation history retained for full 5-year AMLR retention period.</p><p>DPO sign-off obtained. GDPR erasure request logged as Completed — Pseudonymised.</p>`,
			"GDPR Art.17 resolved via pseudonymisation. PII replaced. AML records retained per AMLR Art.77. DPO approved.",
			dbgen.NoteTypeEnumFormal, false, false},

		{4, u.analyst3, "AI Copilot — STR Narrative Draft",
			`<p><em>AI-generated draft. Review before submitting to FNTT.</em></p><p>The reporting entity identified a suspicious SEPA Instant cluster originating from EMI account LT-EMI-7741. Between [date] and [date+48h], 23 transactions totalling EUR 85,000 were transmitted to beneficiary accounts in three EU member states (Latvia, Estonia, Poland). All transactions were structured below the EUR 10,000 threshold for enhanced due diligence. The velocity of transactions exceeded the account's 30-day baseline by a factor of 18. The beneficiary accounts were opened within 60 days and have no prior transaction history with the reporting entity. The pattern is consistent with the FATF typology of layering via EU e-money accounts. The reporting entity has blocked the account and notified the relevant correspondent banks.</p>`,
			"AI Copilot STR narrative draft. Review before FNTT submission.",
			dbgen.NoteTypeEnumInternal, false, true},
	}

	created := make([]dbgen.CreateNoteRow, 0, len(notes))
	for _, ns := range notes {
		if ns.caseIdx >= len(cases) {
			continue
		}
		n, err := q.CreateNote(ctx, dbgen.CreateNoteParams{
			CaseID:      cases[ns.caseIdx].ID,
			AuthorID:    ns.author.ID,
			Title:       ns.title,
			Content:     ns.content,
			ContentText: ns.contentText,
			NoteType:    ns.noteType,
			IsPinned:    ns.isPinned,
			AiGenerated: ns.aiGenerated,
		})
		if err != nil {
			fmt.Println("  warn: EU note seed error:", err)
			continue
		}
		created = append(created, n)
	}

	commentSpecs := []struct {
		noteIdx int
		author  dbgen.CreateUserRow
		content string
	}{
		{0, u.manager, "Confirmed. STR draft is in Documents tab. Needs my sign-off before goAML XML export. Please ping me by 14:00 CET."},
		{0, u.analyst2, "I've cross-referenced all 23 beneficiary IBANs. No EU Consolidated List hits. Proceeding with STR."},
		{1, u.analyst1, "MOKAS notification template prepared. Awaiting senior management decision on whether to file simultaneously with STR."},
		{2, u.manager, "87% match is above our threshold for manual review. Hold the payment. Get legal on the call before we confirm or dismiss."},
		{3, u.analyst2, "DPO has been notified. Awaiting their written sign-off before processing the pseudonymisation."},
	}

	commentCount := 0
	for _, cs := range commentSpecs {
		if cs.noteIdx >= len(created) {
			continue
		}
		_, err := q.CreateComment(ctx, dbgen.CreateCommentParams{
			CaseID:   created[cs.noteIdx].CaseID,
			NoteID:   uuid.NullUUID{UUID: created[cs.noteIdx].ID, Valid: true},
			AuthorID: cs.author.ID,
			Content:  cs.content,
		})
		if err == nil {
			commentCount++
		}
	}

	fmt.Printf("  📝 %d EU notes · %d comments seeded\n", len(created), commentCount)
}

// seedEUAILogs creates EU-context AI log entries.
func seedEUAILogs(ctx context.Context, q *dbgen.Queries, cases []dbgen.Case, u userSet) {
	actors := []uuid.UUID{u.analyst1.ID, u.analyst2.ID, u.analyst3.ID}
	reviewers := []uuid.UUID{u.manager.ID, u.admin.ID}

	euAITemplates := []struct {
		actionType   string
		prompt       string
		response     string
		tokIn, tokOut int32
		latency      int32
		confidence   float64
		reviewStatus string
	}{
		{
			"ask",
			"What are the AML red flags in this case? What FATF typologies apply under EU AMLR 2024?",
			"Three primary AML red flags identified:\n\n" +
				"**1. Structuring:** All 23 transactions structured below the €10,000 EDD threshold — classic placement technique.\n" +
				"**2. Velocity:** Transaction frequency 18x above the account's 30-day SEPA Instant baseline.\n" +
				"**3. Beneficiary concentration:** Recipients are newly opened IBANs across three EU jurisdictions with no prior transaction history.\n\n" +
				"**Applicable FATF Typology:** Layering via EU e-money accounts (FATF Report: Money Laundering Through Virtual Assets, 2019). Also consistent with FNTT 2025 typology alert: cross-border mule farming via Lithuanian EMIs.\n\n" +
				"**Recommendation:** File STR with FNTT within 3 working days. Apply account freeze pending investigation.",
			1940, 380, 2450, 0.91, "approved",
		},
		{
			"summarize",
			"[SYSTEM: Generate investigation summary for FNTT STR narrative — EU AMLR 2024 context]",
			"**STR Narrative Summary**\n\nThe reporting entity identified a suspicious SEPA Instant cluster " +
				"exhibiting structuring behaviour below the EDD threshold. Transaction velocity exceeded account " +
				"baseline by 18x. Beneficiary accounts were recently established across three EU member states " +
				"with no prior commercial relationship. Pattern is consistent with layering via EU e-money " +
				"accounts. FATF Travel Rule records are present for all transactions. " +
				"EU Consolidated, UN, and OFAC sanctions lists have been screened — no confirmed matches.",
			2200, 310, 3200, 0.89, "edited",
		},
		{
			"extract_actions",
			"[SYSTEM: Extract required investigation steps — EU AMLR 2024 / FNTT regulatory context]",
			"Required investigation actions:\n\n" +
				"1. **File STR with FNTT via goAML** (Critical) — 3 working day deadline from date of suspicion\n" +
				"2. **Verify beneficiary IBANs** against EU Consolidated, UN, OFAC lists (High)\n" +
				"3. **Apply EDD** — source of funds, business profile, source of wealth (High)\n" +
				"4. **Verify UBO to natural person level** — AMLR 2024 Art. 42 (High)\n" +
				"5. **Preserve SEPA transaction logs** and device fingerprint records (High)\n" +
				"6. **Escalate to Senior Management** if PEP involvement confirmed — AMLR 2024 Art. 33 (Medium)",
			1750, 340, 2100, 0.85, "pending",
		},
		{
			"ask",
			"Does the GDPR Art.17 erasure request conflict with our AMLR 2024 retention obligations?",
			"**Legal Analysis — GDPR Art. 17 vs AMLR 2024 Art. 77**\n\n" +
				"GDPR Art. 17(3)(e) explicitly permits retention where processing is necessary for compliance with " +
				"a legal obligation. AMLR 2024 Art. 77 mandates minimum 5-year retention from the end of the " +
				"business relationship.\n\n" +
				"**Recommended resolution:** Pseudonymisation — not deletion.\n" +
				"Replace all PII fields (name, IBAN, ID number, contact details) with a cryptographic pseudonym. " +
				"Retain transaction records, STR filings, and investigation notes for the full 5-year period.\n\n" +
				"This approach satisfies both obligations. The data subject's right to erasure is fulfilled " +
				"(PII is effectively erased). The AML obligation is satisfied (transaction data is retained). " +
				"DPO sign-off is required before processing.\n\n" +
				"**Note:** This is AI-generated legal analysis. Your DPO must confirm before action.",
			1880, 420, 2800, 0.76, "approved",
		},
		{
			"ask",
			"Does the UBO nominee director arrangement satisfy AMLR 2024 Art. 42?",
			"**UBO Assessment — AMLR 2024 Art. 42(3)**\n\n" +
				"Article 42(3) requires obliged entities to identify the natural person(s) who ultimately " +
				"own or control the customer. Nominee directors and registered agents do not satisfy this " +
				"requirement — they must be looked through to the underlying beneficial owner.\n\n" +
				"**Finding:** The disclosed nominee director (Seychelles registered agent) does not satisfy " +
				"Art. 42(3). The institution must identify and verify the natural person behind the BVI " +
				"holding and Seychelles trust structure.\n\n" +
				"**Consequence of non-compliance:** AMLR 2024 Art. 42(6) — where the beneficial owner " +
				"cannot be identified, the institution must not establish or continue the business relationship " +
				"and must consider filing an STR.\n\n" +
				"**Recommendation:** Suspend account pending UBO identification. Notify MOKAS if UBO " +
				"cannot be confirmed within 5 working days.",
			1660, 390, 2350, 0.82, "pending",
		},
	}

	total := 0
	for i, c := range cases {
		numLogs := 1 + (i % 2)
		for j := 0; j < numLogs; j++ {
			tmpl := euAITemplates[(i+j)%len(euAITemplates)]
			actor := actors[(i+j)%len(actors)]
			reviewer := reviewers[i%len(reviewers)]

			confScore := sql.NullString{}
			if tmpl.confidence > 0 {
				confScore = sql.NullString{String: fmt.Sprintf("%.2f", tmpl.confidence), Valid: true}
			}

			citations := jsonb([]map[string]any{{
				"doc_id": "eu-seed-doc", "doc_name": fmt.Sprintf("sepa_export_%s.pdf", c.CaseNumber),
				"chunk_index": 2, "page_number": 3,
				"excerpt":    "Transaction velocity 18x above account 30-day SEPA baseline. Structuring pattern confirmed.",
				"similarity": 0.89 + float64(i%3)*0.02,
			}})

			row, err := q.CreateAILog(ctx, dbgen.CreateAILogParams{
				CaseID: c.ID, TriggeredBy: actor,
				ActionType: tmpl.actionType, PromptText: tmpl.prompt,
				ResponseText: tmpl.response, ModelUsed: "claude-sonnet-4-6",
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
						String: tmpl.response + "\n\n[Edited by reviewer: Added FNTT-specific filing reference and corrected FATF typology code to ML-01-EU per FNTT 2025 guidance]",
						Valid:  true,
					},
				})
				must(err)
			}
		}
	}

	// Today's AI logs for dashboard "AI Queries Today" metric
	todayCases := cases[:5]
	for _, c := range todayCases {
		_, err := q.CreateAILog(ctx, dbgen.CreateAILogParams{
			CaseID:      c.ID,
			TriggeredBy: u.analyst1.ID,
			ActionType:  "ask",
			PromptText:  "What is the current status of this investigation and what needs to be done before the FNTT deadline?",
			ResponseText: "Current status: In Progress — Day 2 of 3 FNTT STR deadline.\n\n" +
				"Remaining actions:\n" +
				"1. Manager sign-off on STR draft (pending — due today)\n" +
				"2. goAML XML export and submission to FNTT portal\n" +
				"3. Retain FNTT submission receipt in Documents tab\n\n" +
				"UBO verification: Completed. Account freeze: Applied. Correspondent bank: Notified.",
			ModelUsed:       "claude-sonnet-4-6",
			Provider:        "anthropic",
			TokensInput:     1100,
			TokensOutput:    160,
			LatencyMs:       1580,
			ConfidenceScore: sql.NullString{String: "0.88", Valid: true},
			Citations:       jsonb([]map[string]any{}),
		})
		must(err)
		total++
	}

	fmt.Printf("  🤖 %d EU AI log entries created (%d today)\n", total, len(todayCases))
}
