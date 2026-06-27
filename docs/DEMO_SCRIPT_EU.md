# OpsOwl AI — EU Demo Script
## Audience: MLRO / Head of Compliance / CCO (EMI / FX Broker / Fintech)
## Jurisdiction: Lithuania (FNTT) or Cyprus (MOKAS)
## Duration: 45 minutes | Format: Screen share + narration

---

> **Before the call**
> - DB seeded with EU demo data: `SEED_MODE=eu go run ./cmd/seed`
> - Logged in as: `admin@opsowl.dev` / `OpsOwl@2024` (Aistė Bernotienė, Admin)
> - Browser: full screen, no other tabs
> - URL: `http://localhost:3001`
> - For Lithuania audience: FNTT context, 3-day STR deadline
> - For Cyprus audience: swap FNTT → MOKAS, 3-day → 5-day throughout

---

## PART 1 — THE HOOK (3 minutes)

### Opening line (say this before sharing screen)

**Lithuania / TransferGo-type EMI:**

"Here's a scenario. Your analyst flags a suspicious SEPA Instant cluster on Monday — 23 transactions, €85,000, structured just below the €10,000 EDD threshold. Under FNTT rules, your STR has to be filed within 3 working days. Today is Day 3. The analyst is in a governance training. Her notes are in a Word file. The CBS export is in a shared folder nobody else knows the path to.

FNTT has 20 AML inspections planned for 2026. They have published standardised templates. They will ask for the investigation trail.

Let me show you what Day 3 looks like when you have OpsOwl."

**Cyprus / IC Markets-type FX broker:**

"Here's a scenario. Your team identifies a client with a complex offshore structure — BVI holding, Seychelles nominee director, Malta intermediate. Under AMLR 2024 Art. 42, you need to verify the UBO before you can onboard or continue the business relationship. Your analyst has three documents, two of which contradict each other. MOKAS expects your STR within 5 working days of suspicion arising.

CySEC issued 14 enforcement actions in the first quarter of 2026. Three were for inadequate UBO verification.

Let me show you what that investigation looks like with OpsOwl."

**[Share screen now]**

---

## PART 2 — THE DASHBOARD (5 minutes)

**Navigate to:** `http://localhost:3001/dashboard`

"This is your compliance operations command center. Everything your team is working on right now — in one view."

**Stats row:**
- "22 open cases — active investigations, nothing forgotten."
- "6 critical cases — STR deadlines within 48 hours, sanctions hits, escalated UBO gaps."
- "48 AI Copilot queries today — your team is accelerating analysis."
- "3 cases pending your review as manager."

**STR Deadlines card:**
"This card shows every STR approaching its filing deadline. FNTT requires 3 working days. This counter is live. If an analyst misses a deadline, you know before it happens — not after."

**AI Oversight Due card:**
"EU AI Act Art. 13 requires human oversight of automated decisions. Every AI output in OpsOwl goes through a review queue. This number tells you how many are waiting for a qualified human to approve, edit, or reject them. That's your regulators' definition of an AI-governed workflow."

**Pause:**
"No Excel. No Slack message. No 'where is that case?' at 9 AM. It's all here."

---

## PART 3 — THE CASE QUEUE (4 minutes)

**Navigate to:** `http://localhost:3001/cases`

"Every open investigation — sortable, filterable, searchable."

**Filter Priority → Critical:**
"Six critical cases. Let me show you the SEPA Instant cluster."

**Point to case: "Suspicious SEPA Instant Cluster — EMI Account LT-EMI-7741":**
"€85,000. 23 transactions. Day 2 of 3 on the FNTT deadline. This is the case I described in the opening."

**Show search:**
"Type 'SEPA' — finds all related cases instantly. Type 'UBO' — surfaces every case with a pending UBO verification issue."

---

## PART 4 — INSIDE A CASE (12 minutes)

**Open:** "Suspicious SEPA Instant Cluster — EMI Account LT-EMI-7741"

"This is the case. €85,000, 23 SEPA Instant transactions to beneficiaries in Latvia, Estonia, and Poland. Day 2 of the FNTT 3-day STR window."

### 4a. Case Header

"Case number, status, priority, SLA deadline — all here. Assigned to Mantas Žukauskas. But even if Mantas is unreachable, any analyst opens this and sees exactly where the investigation stands."

### 4b. Tasks Tab

**Click Tasks:**

"Investigation checklist. Every required step, owner, deadline."

- "Pull SEPA transaction data from core banking — assigned, due in 6 hours."
- "Verify beneficiary IBANs against EU sanctions list — AI-suggested, due in 12 hours."
- "File STR to FNTT via goAML — due tomorrow. Deadline: 3 working days from detection."

"That last one says AI-suggested. The AI Copilot read the case description and added it automatically — with the correct FNTT deadline pre-calculated. Your analyst didn't need to remember the regulatory checklist."

### 4c. UBO Tab

**Click UBO tab:**

"This tab is mandatory for any case involving a legal entity. EU AMLR 2024 Art. 42 requires you to identify and verify every ultimate beneficial owner holding 25% or more."

**Show UBO record:**
"Owner: Dmitri Volkov. Ownership: 34%. Status: Pending verification. Nationality: Russian. That last field matters — we have an EU sanctions exposure flag on Russian nationals in certain programs."

**Show the pending warning banner:**
"'Case cannot be closed — UBO verification required.' This is a hard gate. Your manager cannot close or move this case to pending review until at least one UBO record is marked Verified. That's your AMLR 2024 Art. 42 compliance — enforced by the system, not by memory."

**Verify the UBO record:**
"Click Verify — gate lifts. Case can now progress. Every verification is logged with who did it and when. Permanent record."

### 4d. Documents Tab

**Click Documents:**

"Three documents: SEPA transaction export, account due diligence file, STR Form draft. Every document is AES-256 encrypted at rest, virus-scanned on upload, and tracked by who uploaded and downloaded it.

If your FNTT inspector asks 'who had access to the STR draft?' — the answer is here, to the second."

### 4e. Notes Tab

**Click Notes:**

"Pinned note from Day 1: 'SEPA Instant cluster flagged — 23 transactions, €85,000, beneficiaries in three EU member states. Structuring pattern below €10,000 EDD threshold. Recommend STR to FNTT.'

That note cannot be edited after saving. Timestamp, author, content — permanent. If this case goes to a FNTT inspection or a court proceeding, this record cannot be disputed."

### 4f. Timeline Tab

**Click Timeline:**

"The complete investigation history — automatically assembled. Case created, assigned, document uploaded, AI query run, status changed, UBO record added. Every action, every actor, every timestamp.

Your analyst doesn't create this. It creates itself."

---

## PART 5 — AI COPILOT (7 minutes)

**Stay in the case — click AI Copilot**

"Now the part that's genuinely different."

### 5a. Ask about red flags

**Type:** `What are the AML red flags in this case? What typologies apply under EU AMLR 2024?`

**While processing:**
"This reads the documents in the case — the SEPA export, the due diligence file — plus the case description and all prior AI findings. It knows this specific case."

**Read the response:**
"Three findings: structuring below the €10,000 EDD threshold, velocity 18x above the account's 30-day SEPA baseline, and beneficiary accounts opened within 60 days across three EU jurisdictions. Applicable FATF typology: Placement via electronic funds transfer, layering through EU EMI accounts.

That analysis would take your analyst 2 hours with a CBS export and a spreadsheet. With the AI Copilot — 90 seconds."

### 5b. EU AI Act compliance

"Every AI response goes through a mandatory human review workflow. Your manager reviews the output, approves, edits, or rejects it. The approved response becomes part of the permanent case record — with a log entry that says 'Manager approved this AI output at this time.'

That's the human-in-the-loop control that the EU AI Act requires. Article 13 — transparency. Article 14 — human oversight. It's built into every AI interaction in OpsOwl, not bolted on."

### 5c. AI Act Compliance page

**Navigate to:** `http://localhost:3001/settings/ai-act`

"This is the AI Act compliance dashboard. Three sections:

One — Conformity checklist. Every requirement from EU AI Act Art. 13 and 14: AI decisions logged with model version, human oversight active, explainability citations recorded, data minimisation enforced. All green.

Two — Pending review queue. Every AI output waiting for human sign-off, with links to the case.

Three — Export audit trail. Date range picker, CSV export. If your regulator or your DPO asks for a complete log of how AI was used in your investigations over the past 6 months — you click Export and hand them the file."

---

## PART 6 — goAML STR FILING (4 minutes)

**Navigate back to the case → STR tab**

"Let me show you the STR filing flow. Your analyst has completed the investigation. Time to file with FNTT."

**Show the STR form:**
"Subject details, transaction summary, EU FATF indicators — all pre-populated from the case. The FNTT entity ID, the goAML schema version, the submission country — all configured at deployment time."

**Click Export goAML XML:**
"This generates a valid goAML XML file. FNTT's goAML portal accepts this format directly. No manual re-entry. No copy-paste from Word into the portal. One click, valid XML, ready for submission.

For Lithuania — FNTT goAML portal: goaml.fntt.lt
For Cyprus — MOKAS goAML portal: goaml.mokas.gov.cy"

"The submission is logged in the audit trail. The STR deadline clock stops. Your compliance record shows the filing was made on Day 2 of 3."

---

## PART 7 — SANCTIONS SCREENING (3 minutes)

**Navigate to:** `http://localhost:3001/settings/sanctions`

"Sanctions screening. Three lists: EU Consolidated, UN Security Council, OFAC SDN. All three loaded and running."

**Show the lists:**
"EU Consolidated: 7,200 entries. UN: 814 entries. OFAC SDN: 12,400 entries. All updated daily at midnight UTC."

**Show an open hit:**
"This subject — Dmitri Volkov — has an 87% fuzzy match against the EU Consolidated List. Not an exact name match, which is why human review is essential. The system surfaced it; a qualified analyst reviews it.

Actions: Mark False Positive, Confirm Hit, Escalate. If confirmed — the regulator notification workflow triggers automatically."

"For your MLRO audit file: every screen, every hit, every decision — logged with who made it and when."

---

## PART 8 — GDPR MODULE (3 minutes)

**Navigate to:** `http://localhost:3001/settings/gdpr`

"Your GDPR Data Subject Access Request queue. This is where the GDPR Art. 17 right-to-erasure tension gets resolved.

An EU customer submits an erasure request. Your legal obligation under GDPR Art. 17 says erase them. Your legal obligation under AMLR 2024 Art. 77 says retain the AML records for 5 years.

Most compliance teams either refuse the erasure — and risk a GDPR fine — or erase the data — and risk a FNTT inspection finding.

OpsOwl resolves this with pseudonymisation. The subject's PII — name, ID number, address, contact details — is replaced with a cryptographic pseudonym. The transaction records and AML investigation data are retained for the full 5-year AMLR period. You've met both obligations. The resolution is logged and auditable."

**Show a DSAR request:**
"Subject name, request date, 30-day deadline countdown — right here. Actions: Start Processing → Pseudonymise. One click. Permanent log entry. GDPR compliance without destroying your AML audit trail."

---

## PART 9 — AUDIT TRAIL (3 minutes)

**Navigate to:** `http://localhost:3001/audit`

"The moment every compliance head dreads: your regulator arrives and asks for the complete record of a specific investigation.

With OpsOwl — this is what you show them."

**Show the audit log:**
"Every login, every document upload, every status change, every AI query, every sanctions hit decision — logged with timestamp, actor, IP address, and event type.

This is a hash-chain audit log. Each entry includes the hash of the previous entry. If anyone modifies or deletes a record, the hash chain breaks — immediately visible. Same principle as blockchain, purpose-built for compliance records.

FNTT asks: 'Can you prove this investigation was conducted properly and that nobody tampered with the record?' — yes, and here is the cryptographic proof."

---

## PART 10 — ON-PREMISE (2 minutes)

"One thing worth saying explicitly: every other modern compliance SaaS means your customer data — KYC records, IBAN details, STR drafts — sits in someone else's cloud. That's a GDPR Art. 32 risk. That's a DORA ICT risk. That's a question your regulator will ask.

OpsOwl runs on your own servers. One Docker Compose command. Your data centre, your network, your control. Zero runtime dependency on our infrastructure.

For your DPO and your IT security team: PostgreSQL for data, MinIO for documents, all open standards. We can configure the AI to run fully on-prem via Ollama — zero data leaves your network, even for AI inference."

---

## PART 11 — CLOSE (4 minutes)

**The recap:**

"We started with a SEPA cluster, Day 2 of a 3-day FNTT deadline, analyst unreachable. With OpsOwl: any team member opens the case, sees the full history from Day 1, sees the STR draft in Documents, sees the task says 'needs manager sign-off,' asks the AI Copilot 'is this draft complete?' — and files it today. The knowledge didn't disappear with the analyst."

**The numbers:**

"Three numbers that matter:

One: a late STR filing under AMLR 2024 can trigger FNTT enforcement action. One avoided penalty covers OpsOwl's annual fee.

Two: analyst time saved — 2 hours per investigation per day, 15 cases per month, 3-person team — that's 90 hours per month returned to actual investigation.

Three: when FNTT arrives, you don't spend 3 days assembling a document bundle. You export the case history in 30 seconds."

**Next step:**

"What I'd suggest is a 30-day proof of concept. We deploy on your own infrastructure, pre-configure it for FNTT or MOKAS, load your jurisdiction settings, and your team runs real cases through it. Your IT team doesn't need to do anything except open a port. We're live in one day.

No commitment required for the POC."

**Questions to invite:**

- "What does your current STR filing process look like end-to-end?"
- "Last time FNTT or MOKAS inspected — how long did it take to produce the investigation records they asked for?"
- "How are you currently handling the Art. 17 erasure vs AMLR retention conflict?"

---

## APPENDIX: HANDLING TOUGH QUESTIONS

| Question | Answer |
|----------|--------|
| "We already use a case management tool" | Does it have goAML XML export, a hash-chain audit log, UBO verification with AMLR Art. 42 gate, GDPR pseudonymisation, and EU AI Act human oversight logging? If yes, you don't need us. If any of those are missing, there's a gap your regulator will find. |
| "We're a small team — is this overkill?" | The platform scales down to 3 users. The compliance obligations don't scale down — FNTT applies the same STR deadline to a 3-person EMI as to a 300-person bank. |
| "What about DORA compliance?" | OpsOwl runs on-prem, so your ICT risk management is entirely within your control. We provide Docker Compose config, backup procedures, and incident response playbooks. Your IT team owns the infrastructure — no third-party ICT dependency at runtime. |
| "Is the AI GDPR compliant?" | Yes. All AI inference runs on your own servers via Ollama by default. No customer data is sent to any external API. If you upgrade to Anthropic Claude or Azure OpenAI, we implement PII masking before every external call — subject names, IBANs, and ID numbers are replaced with anonymised tokens. |
| "Can it integrate with our transaction monitoring system?" | Yes — inbound webhook receiver creates cases automatically from your TMS alerts. We also have a full REST API and outbound webhooks for pushing case status back to your core system. |
| "Who else in the EU is using this?" | We are currently selecting our first 3 EU design partners — institutions who go live first, help shape the product for the EU market, and get a significantly discounted rate in exchange for a reference. That's the conversation I'd like to have with you. |
| "What if the AI gets it wrong?" | Every AI output goes through mandatory human review before it becomes part of the case record. The AI recommends — a qualified analyst or manager decides. The EU AI Act calls this human-in-the-loop oversight. It's enforced by the platform, not by policy. |
| "What's the migration path if we want to leave?" | All data is in standard PostgreSQL. All documents are in MinIO (S3-compatible). Export is CSV or JSON. There is no proprietary data format. You are never locked in. |

---

## SCREEN FLOW CHEAT SHEET

```
Dashboard                → Stats, STR deadline card, AI Oversight Due counter
  ↓
Cases List               → Filter Critical → show queue
  ↓
Case: SEPA Cluster       → Header (SLA Day 2 of 3, FNTT deadline)
  ↓
  → Tasks tab            → AI-suggested STR task, FNTT deadline pre-calculated
  → UBO tab              → Pending UBO record, closure gate blocked
  → Documents tab        → Encrypted vault, chain of custody
  → Notes tab            → Immutable notes, timestamps
  → Timeline tab         → Auto-assembled investigation history
  ↓
AI Copilot               → AML red flags + FATF typology → EU AI Act review flow
  ↓
Settings → AI Act        → Conformity checklist, pending review queue, CSV export
  ↓
Case → STR tab           → goAML XML export → FNTT / MOKAS
  ↓
Settings → Sanctions     → EU Consolidated + UN + OFAC → open hit review
  ↓
Settings → GDPR          → DSAR queue → pseudonymisation workflow
  ↓
Audit Logs               → Hash-chain, tamper-evident, regulator-ready
  ↓
Close                    → 30-day PoC, 1-day deployment, your infrastructure
```

---

## DEMO ENVIRONMENT REFERENCE

| Item | Value |
|------|-------|
| Web URL | http://localhost:3001 |
| API URL | http://localhost:8080 |
| Admin login | admin@opsowl.dev / OpsOwl@2024 |
| Manager login | ravi.kumar@opsowl.dev / OpsOwl@2024 |
| Analyst login | divya.reddy@opsowl.dev / OpsOwl@2024 |
| Seed command | `SEED_MODE=eu go run ./cmd/seed` (from apps/api/) |
| Currency | EUR |
| Regulator (LT) | FNTT — 3-day STR deadline |
| Regulator (CY) | MOKAS — 5-day STR deadline |

---

*Document: OpsOwl Demo Script v1.0 — EU MLRO/CCO Audience — June 2026*
