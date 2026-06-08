# OpsOwl AI — Demo Script
**Audience:** CCO / Head of Compliance / AML Head at NBFC or Bank
**Duration:** 30–45 minutes
**Persona:** You are the Chief Compliance Officer evaluating OpsOwl AI as your investigation platform.

---

## Opening Line
> "What I'm going to show you is the complete investigation lifecycle — from the moment an alert fires in your TMS, all the way to a formally closed, audit-defensible case ready for regulatory examination. Everything runs on your own server — no cloud, no data leaving your network."

---

## MODULE 1 — Administration Setup

### Feature 1 — Users & Roles
**URL:** `/users`

**What to show:**
- 5 roles: Super Admin, Admin, Manager, Analyst, Viewer
- Click role chips to filter by role (chips also show count per role)
- Search by name or email
- Click **+ Invite User** → fill Full Name, Email, Role, Temporary Password → click **Create User**
- Hover over a user's role badge → pencil icon appears → click to change role inline
- Click **Deactivate** next to any user → confirm

**Demo line:**
> "Every person in your compliance team gets a role. Analysts can only work on cases assigned to them. Managers see the full team. Viewers — like your internal auditor — can read everything but change nothing. When someone leaves, one click deactivates them instantly. All their past actions remain in the audit trail — nothing is deleted."

**Sample data:**
- Full Name: `Rahul Desai`
- Email: `rahul.desai@opsowl.dev`
- Role: `Analyst`
- Password: `Test@1234`

---

### Feature 2 — SLA Policies
**URL:** `/sla`

**What to show:**
- Create 3 SLA policies
- Show breach threshold, priority mapping
- Explain SLA countdown starts the moment a case is created

**Demo line:**
> "Regulators expect investigations to be completed within defined turnaround times. Every case gets an SLA clock from the moment it's created. When it breaches, the manager is alerted automatically."

**Sample data:**

| Name | Priority | Breach After |
|------|----------|-------------|
| Urgent — Regulatory Deadline | Critical | 24 hours |
| Standard AML Investigation | High | 72 hours |
| Routine Review | Medium | 7 days |

---

### Feature 3 — Case Templates
**URL:** `/settings/templates`

**What to show:**
- Create a template for AML investigation
- Pre-fill title, description, priority, case type, tags
- Explain it auto-populates fields when a case is created from this template

**Demo line:**
> "Every AML investigation starts the same way. Templates ensure your analysts don't start from a blank page — the right fields, priority, and case type are pre-filled from day one."

**Sample data:**
- Template Name: `AML — Suspicious Transaction`
- Case Type: `aml`
- Priority: `High`
- Tags: `structuring`, `threshold`

---

### Feature 4 — Investigation Playbooks
**URL:** `/settings/playbooks`

**What to show:**
- Create a playbook for AML case type
- Add ordered tasks: KYC review, Transaction analysis, Counterparty check, STR decision
- Explain tasks are auto-created when a case of this type is opened

**Demo line:**
> "Investigators, especially newer ones, miss steps under pressure. Playbooks enforce consistent investigation procedures automatically. When an AML case is created, these tasks are already waiting for the analyst."

**Sample data:**
- Playbook Name: `Standard AML Investigation`
- Case Type: `aml`
- Tasks (in order):
  1. Review customer KYC and CDD documents
  2. Analyse transaction history for last 6 months
  3. Check counterparty risk profile
  4. Assess structuring or threshold patterns
  5. Document findings with supporting evidence
  6. Make STR / no-STR decision with rationale

---

### Feature 5 — Assignment Rules
**URL:** `/settings/assignment-rules`

**What to show:**
- Create a rule: if case type = AML and priority = High → assign to Divya Reddy
- Explain round-robin option
- Explain rules run automatically when a case is created

**Demo line:**
> "No case should sit unassigned. Assignment rules route every alert to the right analyst automatically — by case type, priority, source, or round-robin across the team."

---

### Feature 6 — Alert Sources & Alert Rules
**URL:** `/settings/alert-sources` then `/settings/alert-rules`

**What to show:**
- Create an alert source: name it `Core Banking TMS`, type = webhook
- Copy the inbound URL
- Create an alert rule: if amount > 10,00,000 → priority High, case type AML, auto-assign

**Demo line:**
> "Your TMS fires an alert. That alert hits this URL, gets evaluated against your rules, and a case is created and assigned — automatically, in under a second. No manual triage."

---

### Feature 7 — Inbound Webhook Setup
**URL:** `/settings/inbound`

**What to show:**
- Show the inbound endpoint
- Show sample payload format
- Explain auto case creation flow end to end

---

### Feature 8 — Outbound Webhooks
**URL:** `/settings/webhooks`

**What to show:**
- Create a webhook: name = `Risk Platform Notify`, URL = any test URL
- Show events that can trigger it: case created, case closed, STR filed
- Show HMAC signing toggle
- Test the webhook

**Demo line:**
> "When a case closes or an STR is filed, your risk platform, core banking system, or any downstream tool gets notified automatically. HMAC signing ensures the payload cannot be tampered with in transit."

---

### Feature 9 — API Keys
**URL:** `/settings/api-keys`

**What to show:**
- Generate an API key
- Name it `LOS Integration`
- Show it only displays once
- Explain it allows external systems to create/read cases programmatically

---

### Feature 10 — Scheduled Reports
**URL:** `/settings/reports`

**What to show:**
- Create a report: weekly, every Monday 9am, PDF, email to `ravi.kumar@opsowl.dev`
- Report type: Case Summary

**Demo line:**
> "Every Monday morning your compliance head gets a PDF in their inbox — open cases, SLA breaches, STR filings last week. No one has to generate it manually."

---

### Feature 11 — Data Retention
**URL:** `/settings/retention`

**What to show:**
- Show retention policies per document type
- Explain PMLA requirement: 5 years minimum

---

### Feature 12 — Workspace Settings
**URL:** `/settings`

**What to show:**
- Change `Case` → `Investigation`
- Change `Analyst` → `Investigation Officer`
- Change `SLA` → `TAT`
- Show it reflects across the entire application instantly

**Demo line:**
> "Every bank and NBFC has their own terminology. OpsOwl adapts to your language — not the other way around."

---

## MODULE 2 — Alert Ingestion & Case Creation

### Feature 13 — Alert Queue
**URL:** `/alerts`

**What to show:**
- Show inbound alerts that arrived via webhook
- Show auto-created cases linked to each alert
- Show alert source, payload, timestamp

---

### Feature 14 — Create Case Manually
**URL:** `/cases/new`

**What to show:**
- Fill all fields with sample AML case data
- Assign to Divya Reddy
- Set priority High, SLA starts immediately
- Add tags

**Sample data:**
- Title: `Suspicious NEFT Transfers — Rahul Sharma`
- Description: `Customer made 9 NEFT transfers of ₹9.5L each over 3 days, just below ₹10L reporting threshold. Potential structuring pattern.`
- Case Type: `aml`
- Priority: `High`
- Assigned To: `Divya Reddy`
- Tags: `structuring`, `neft`, `threshold-evasion`

---

### Feature 15 — Bulk Import Cases
**URL:** `/cases/import`

**What to show:**
- Download CSV template
- Upload a filled CSV with 3-5 cases
- Show all cases created in one operation

**Demo line:**
> "If you're migrating from Excel or another system, bulk import lets you bring in thousands of cases at once. Each one gets its SLA clock, assignment, and audit trail from the moment it lands."

---

### Feature 16 — Cases List
**URL:** `/cases`

**What to show:**
- Filter by status, priority, assigned to
- Search by case number or title
- Show SLA countdown on each row
- Show priority badges
- Bulk select → bulk assign or bulk status change

---

## MODULE 3 — Investigation Workspace

### Feature 17 — Case Overview
**URL:** `/cases/[id]`

**What to show:**
- Case header: title, status, priority, SLA countdown
- Status transition buttons: move from Open → In Progress
- Assigned to, created by, due date
- AI Copilot button
- Export PDF button
- Sub-navigation: Notes, Tasks, Documents, Transactions, Findings, Narrative, Closure, Timeline

---

### Feature 18 — Notes
**URL:** `/cases/[id]/notes`

**What to show:**
- Add an internal note with rich text
- Add another note as a different user
- Show threaded notes with timestamps and author

**Sample note:**
> `Initial review complete. Customer has 9 NEFT transactions of ₹9.5L each over 3 days. Counterparty is a sole proprietorship registered 2 months ago. Escalating to detailed transaction analysis.`

---

### Feature 19 — Tasks
**URL:** `/cases/[id]/tasks`

**What to show:**
- Playbook tasks already created automatically
- Check off first task: KYC review done
- Add a new manual task: `Request 6-month bank statement from customer`
- Assign task to Arjun Mehta, due date tomorrow
- Show task status: todo → in progress → done

---

### Feature 20 — Documents
**URL:** `/cases/[id]/documents`

**What to show:**
- Upload a PDF (bank statement or KYC document)
- Show OCR processing
- Show document preview
- Show version history
- Explain AI will analyse this document

**Demo line:**
> "Every document uploaded to a case is OCR-processed and becomes searchable and AI-readable. Your analyst can ask the AI Copilot questions directly over these documents — in seconds."

---

### Feature 21 — Communications
**URL:** `/cases/[id]/comms`

**What to show:**
- Add a communication entry: email from customer
- Show entity extraction: parties, amounts, dates pulled out automatically
- Show red flags identified

---

### Feature 22 — Timeline
**URL:** `/cases/[id]/timeline`

**What to show:**
- Every action in the case listed chronologically
- Note added, document uploaded, status changed, task completed
- Real-time — new actions appear without page refresh
- Filter by event type

**Demo line:**
> "This is the complete audit trail of everything that happened on this case. Every action by every person, in order, timestamped. An examiner can sit here and trace the entire investigation without asking anyone a single question."

---

## MODULE 4 — Transaction & Entity Intelligence

### Feature 23 — Transactions & Flow of Funds
**URL:** `/cases/[id]/transactions`

**What to show:**
- Add a transaction manually with full details
- Show transaction timeline in chronological order
- Toggle to Flow of Funds graph — nodes and edges
- Flag a transaction as suspicious with reason
- Show summary bar: total flow, flagged count, channels

**Sample transaction:**
- Ref: `NEFT/2026/001`
- Date: today
- Amount: `950000`
- Currency: `INR`
- Type: `debit`
- Channel: `neft`
- From: `HDFC0000123456`
- To: `ICIC0000987654`
- Narration: `NEFT transfer to Rahul Enterprises`

**Demo line:**
> "Nine transactions, just under the reporting threshold, all going to the same new entity. The flow-of-funds graph makes the structuring pattern visible in 3 seconds. This would take an analyst 2 hours in Excel."

---

### Feature 24 — Subjects
**URL:** `/subjects`

**What to show:**
- Search for a subject
- Click into Subject 360 profile
- Show linked cases, linked accounts, entity relationships
- Show connected entity graph
- Add a relationship between two subjects

**Demo line:**
> "Who is this person really? Subject 360 pulls together every case they've appeared in, every linked account, every known relationship. This is what skip tracing looks like when it's built into the investigation platform."

---

## MODULE 5 — AI & Document Intelligence

### Feature 25 — AI Insights & Risk Score
**URL:** `/cases/[id]/insights`

**What to show:**
- Click **Run AI Analysis**
- Show risk score generated: High / Medium / Low
- Show risk factors identified
- Show AI summary of the case

**Demo line:**
> "The AI reads the entire case — documents, transactions, notes, entities — and produces a risk score with reasons. Your manager can triage 50 cases in 5 minutes instead of 5 hours."

---

### Feature 26 — AI Copilot
**URL:** `/cases/[id]` — click AI Copilot button

**What to show:**
- Ask: `What is the relationship between the customer and Rahul Enterprises?`
- Ask: `What transactions are above ₹5 lakhs?`
- Ask: `Summarise the key red flags in this case`
- Show PII masking note

**Demo line:**
> "Your analyst can interrogate the entire case in plain English. The AI answers from the case documents and data — and PII is masked before any external call. The data never leaves your server."

---

### Feature 27 — AI Logs
**URL:** `/cases/[id]/ai-logs`

**What to show:**
- Every AI call: model used, prompt tokens, response, latency
- Timestamp and user who triggered it

**Demo line:**
> "Every AI decision is logged. Model, prompt, response, latency — all auditable. Your compliance team can defend every AI-assisted decision to an examiner."

---

## MODULE 6 — Findings & Evidence

### Feature 28 — Findings & Evidence
**URL:** `/cases/[id]/findings`

**What to show:**
- Create a finding: title, description, severity High
- Attach evidence document to it
- Show severity gate: High requires evidence before submitting
- Submit finding
- Approve finding as manager

**Sample finding:**
- Title: `Structuring Pattern Detected`
- Description: `9 NEFT transactions of ₹9.5L each over 3 days, just below ₹10L CTR threshold. Consistent structuring pattern.`
- Severity: `High`
- Evidence: upload bank statement PDF

**Demo line:**
> "Findings are the structured conclusions of your investigation. High-severity findings cannot be submitted without attached evidence — the system enforces documentation discipline so nothing is defensible."

---

## MODULE 7 — Regulatory Reporting & Closure

### Feature 29 — STR / goAML
**URL:** `/cases/[id]/str`

**What to show:**
- Fill STR details: reporting entity, subject details, transaction details
- Generate goAML-compatible XML
- Download XML ready for FIU-IND submission

**Demo line:**
> "When you decide to file an STR, the platform generates a goAML-compatible XML file ready for direct submission to FIU-IND. No manual formatting, no XML editing."

---

### Feature 30 — AI Narrative
**URL:** `/cases/[id]/narrative`

**What to show:**
- Click **Generate Narrative**
- Show AI-drafted investigation summary
- Edit the body
- Manager approves narrative
- Show approve/reject workflow

**Demo line:**
> "The AI drafts the complete investigation narrative from everything in the case — transactions, findings, entities, timeline. Your analyst edits if needed, the manager approves. What used to take 3 hours takes 15 minutes."

---

### Feature 31 — Case Closure
**URL:** `/cases/[id]/closure`

**What to show:**
- Cannot close without disposition and rationale
- Fill: disposition = STR Filed, rationale, regulatory report reference
- Close the case
- Show re-open option

**Demo line:**
> "A case cannot be closed without a documented disposition and rationale. The platform enforces completeness. An examiner will never find a closed case without a reason."

---

## MODULE 8 — Reporting & Monitoring

### Feature 32 — Dashboard
**URL:** `/dashboard`

**What to show:**
- Open cases, SLA breached, pending review counts
- Attention Required panel
- Recent activity feed

---

### Feature 33 — Workload
**URL:** `/workload`

**What to show:**
- Cases per analyst
- Average TAT per analyst
- SLA breach rate
- Case ageing distribution

**Demo line:**
> "Your compliance manager can see at a glance who is overloaded, who has capacity, and which cases are about to breach SLA — before it happens."

---

### Feature 34 — Analytics
**URL:** `/analytics`

**What to show:**
- Case volume trends
- STR filing rate
- Risk level distribution
- Export to CSV for Power BI / Tableau

---

### Feature 35 — Global Tasks
**URL:** `/tasks`

**What to show:**
- All tasks across all cases in one view
- Filter by assigned to, due date, status

---

### Feature 36 — Global Documents
**URL:** `/documents`

**What to show:**
- All documents across all cases
- Search by filename or content

---

### Feature 37 — Search
**URL:** `/search`

**What to show:**
- Search across cases, documents, notes, subjects
- Show results with case context

---

## MODULE 9 — Audit & Compliance

### Feature 38 — Audit Logs
**URL:** `/audit`

**What to show:**
- Every system-wide action: login, case created, document uploaded, user deactivated
- SHA-256 hash chain — each entry linked to previous
- Filter by user, action type, date range
- Export audit log

**Demo line:**
> "This is the tamper-evident audit trail for the entire platform. Every action by every user, SHA-256 hash-chained. If an examiner asks 'who accessed this case on this date' — it's here. If anyone tries to delete or alter an entry, the hash chain breaks and it's immediately detectable."

---

### Feature 39 — Inbound Events Log
**URL:** `/alerts`

**What to show:**
- Every alert that arrived via inbound webhook
- Payload, source, timestamp, case created from it

---

## MODULE 10 — Deployment

### Feature 40 — On-Premise & Air-Gapped AI

**Demo line:**
> "Everything you've seen today runs on a single Linux server in your data centre. One command to install. No cloud dependency. The AI inference runs locally — Ollama runs llama3.1:8b on your own hardware. Not a single byte of investigation data leaves your network. You have full control."

---

## Closing Line
> "From the moment an alert fires in your TMS to a formally closed, goAML-filed, audit-defensible case — everything happens in one platform, on your infrastructure, with a complete hash-chained trail that survives any regulatory examination."

> "What questions do you have?"
