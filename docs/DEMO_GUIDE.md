# OpsOwl AI — Complete Demo Guide
### For: CTO / Decision-Maker Audience
### Prepared by: Product Team
### Version: 1.0 · June 2026

---

> **How to use this guide**
> Read this top-to-bottom before the demo. Each section has:
> - The **talking point** (what to say)
> - The **click path** (exactly where to go in the app)
> - The **"why it matters"** hook (the business outcome)
> - The **objection handler** (what the CTO will push back on)

---

## OPENING — The Problem You Are Solving (2 minutes)

**Say this before touching the app:**

> "Before I show you anything, let me describe a Friday afternoon at most NBFCs and mid-sized banks.
>
> Your transaction monitoring system fires 200 alerts. Your compliance team has 4 analysts. Each analyst opens an alert, opens Excel, copies transaction data, emails the KYC team, waits for a reply, writes a Word document summarising their findings, emails it to the manager, the manager prints it, reads it, signs it, scans it, and files it in a folder.
>
> Three days later, a different analyst gets a new alert on the same customer. They have no idea the first analyst already investigated this person last month. They start from scratch.
>
> Meanwhile, FIU-IND's deadline for your STR filing is in 4 days. Nobody knows which cases are close to breaching it. The MLRO calls the compliance head every Monday asking for a status update. The compliance head opens six different Excel sheets to answer.
>
> This is the problem OpsOwl solves. Let me show you how."

**Now open the browser to `localhost:3001` and log in:**
```
Email:    admin@opsowl.dev
Password: OpsOwl@2024
```

---

## PART 1 — THE DASHBOARD (3 minutes)

**Click path:** You land on the Dashboard automatically.

### What to show:

Point to the **four stat cards** at the top:

> "The moment your MLRO or compliance head opens their laptop in the morning, they see this. Total cases in the system, open cases being worked right now, cases that have already breached their SLA deadline, and cases that need urgent attention today. No phone calls, no Excel updates, no Monday morning emails. The number is live."

Point to the **Attention Required panel** (red/amber cases):

> "These are the cases your team needs to act on today. Every case here has either breached its SLA or been escalated. This panel refreshes every 10 seconds. If a case breaches SLA while your analyst is eating lunch, it appears here within seconds."

Point to the **My Queue panel** (bottom):

> "This is the analyst's personal view. The five cases assigned to them, sorted by SLA urgency. They don't need to filter anything or ask their manager what to work on next. The system tells them."

**Why it matters:**
> "The average compliance team wastes 45 minutes every morning figuring out what to prioritise. This dashboard eliminates that. Your team starts working in 30 seconds, not 45 minutes."

**Objection:** *"We already have a dashboard in our core banking system."*
> "Your CBS dashboard shows transactions. This shows the investigation status of those transactions — who is working on them, whether they'll breach regulatory deadlines, and whether your team has capacity. These are completely different questions."

---

## PART 2 — CASES: THE CORE OF EVERY INVESTIGATION (10 minutes)

**Click path:** Sidebar → **Cases**

### 2a. The Cases List

> "Every investigation in your organisation lives here as a Case. Every alert from your transaction monitoring system, every internal referral, every customer complaint that needs investigation — it becomes a case. Look at what each row tells you without opening anything."

Point to the columns:
- **Case Number** — INV-0001, INV-0002 — unique, auto-assigned, permanent
- **Title** — what the investigation is about
- **Status** — where it is in the workflow right now
- **Priority** — Critical / High / Medium / Low
- **Assignee** — exactly who owns this investigation
- **SLA** — how much time is left before a regulatory deadline is breached
- **Risk** — AI-calculated risk score (we'll come to this)

> "Your current system has some of these. I want to show you what you don't have."

**Demo: Filters**

Click the **Status dropdown** → select "In Progress":
> "Instantly filtered to only the cases actively being worked. If I'm a manager doing a morning review, I can see every case in-flight in one view."

Click the **My Cases** toggle (person icon):
> "This analyst just logged in. One click and they see only their cases. No distractions, no noise."

Type "Patel" in the **Search bar**:
> "Full-text search across all case titles and numbers. Results appear as you type. This is server-side — it searches cases your analyst hasn't loaded yet, not just what's visible on screen."

**Demo: Bulk Operations**

Select 3 cases using the checkboxes on the left → show the action bar:
> "Select any number of cases and bulk-reassign them to a different analyst, or bulk-change their status. If your senior analyst goes on leave, their 15 active cases reassign to two junior analysts in 10 seconds. In Excel, that's an hour of emails."

**Why it matters:**
> "The average compliance manager spends 2 hours a day just tracking case status. This page eliminates that entirely."

---

### 2b. Opening a Case — The Investigation Workspace

**Click path:** Click any case in the list.

> "Let me open this case and walk you through the full investigation workspace. This is where your analysts spend 90% of their time."

Point to the **sticky header**:
- Case number, title
- Status badge with transition buttons
- Priority badge
- SLA countdown (live, ticking)
- AI Risk Score badge (e.g., "HIGH 74")
- 🤖 AI Copilot button
- ⚡ Analyse button
- 📄 PDF Export button

> "Everything your analyst needs to know about this case is visible without scrolling. The SLA is counting down in real time. The AI risk score is updated every time new documents are uploaded or analysis is re-run. The status buttons — see how there are two green ones and one red? Green buttons are routine transitions that happen on one click. Red is escalate — that opens a confirmation with a mandatory reason field."

**Demo: Status Transition**

Click "In Progress" (a green button):
> "One click, status updated, timeline entry created, assignee notified. No form, no confirmation. Compare that to a workflow system where a status change requires filling a 5-field form."

---

### 2c. The Case Overview — Two-Column Workspace

Point to the **left column** (Investigation Feed):

> "The left side is where the investigation story unfolds. The description tells you what triggered this case. Below it is the Live Feed — a real-time stream of every action taken on this case. Watch this."

Open a second browser tab, go to the same case, and add a note. Come back to the first tab:
> "The note appeared here without refreshing. This is Server-Sent Events — a persistent connection between the browser and your server. When an analyst in Mumbai adds a note, the manager in Delhi sees it in under a second. No refresh button, no polling."

Point to the **right column** (Context Panel):

> "The right side gives you everything you need without switching tabs. Who owns this case, when it was created, when it's due, the SLA deadline. The top 3 open tasks. The last 2 uploaded documents. You can see the entire case status at a glance."

**Demo: Inline Reassignment**

Click the assignee name in the right column:
> "Click the assignee and a dropdown appears inline. Reassign without opening a form, without leaving the page. This sounds small but when you're managing 50 cases, this saves 30 seconds per reassignment — multiply that by 20 reassignments a day."

---

## PART 3 — THE INVESTIGATION WORKFLOW (20 minutes)

> "Now I'm going to walk through a complete, realistic investigation. Kiran Patel is a customer who has been making structured cash deposits followed by immediate transfers to shell companies. Your transaction monitoring system flagged it. Let me show you what your analyst does."

---

### STEP 1: Create the Case

**Click path:** Cases → **New Case** (top right)

Fill in:
- **Title:** Suspicious Cash Deposits — Kiran Patel
- **Case Type:** AML
- **Priority:** High
- **Template:** Select "AML — Suspicious Cash Transaction"

> "Watch what happens when I select the template."

All 8 tasks auto-populate. Description is pre-filled.

> "Case templates encode your organisation's investigation SOPs. You defined these 8 steps once — verify KYC, pull 12-month history, identify beneficiaries, check registrations, adverse media, CIBIL, findings summary, draft STR. Every analyst follows this checklist automatically. A new analyst joining next month follows the same procedure as your 10-year veteran."

**Objection:** *"Our analysts know what to do. They don't need a checklist."*
> "Your senior analysts do. But what about new hires? What about 2am when a junior analyst is the only one on duty? And more importantly — your RBI auditor doesn't care whether your analyst knew what to do. They care whether you can prove it was done. Every task completion is timestamped and logged."

---

### STEP 2: Upload Evidence Documents

**Click path:** Case → **Documents** tab → drag and drop any PDF

> "The analyst uploads the bank statement, KYC documents, and company registration papers. Watch the status badge."

Show the status cycling: `Queued → Processing → Done`

> "In the background, our AI service is doing OCR on any scanned images, extracting text from PDFs, chunking the content, creating vector embeddings, and storing them in a searchable index. By the time the analyst finishes their coffee, the AI can answer questions about this 50-page document."

> "Three documents. Normally, an analyst reads all three manually — 2 hours minimum. With OpsOwl — 30 seconds."

---

### STEP 3: Log Communications

**Click path:** Case → **Comms** tab → **Log Communication**

Fill in:
- Channel: Phone
- Direction: Outbound
- Contact: Kiran Patel
- Notes: Customer stated funds are from agricultural income. Unable to provide documentary evidence. Evasive on beneficiary entity details.

> "Every phone call, every email, every in-person meeting related to this investigation is logged here with timestamp, channel, and your notes. If this case ever goes to court or regulatory review, you have a complete communication log. Try doing that with email threads."

---

### STEP 4: Add Investigation Notes

**Click path:** Case → **Notes** tab → **Add Note**

Type a detailed investigation note. Select type: Internal.

> "Notes have types — Internal (only your team sees), Customer-facing (goes in correspondence), Escalation (flags to management). The analyst documents their analysis here in real time, not in a Word document on their desktop that nobody else can see."

Point to the note appearing in the Timeline:
> "The moment the note is saved, it appears in the Timeline — permanently, with timestamp and author. This is the case's living history."

---

### STEP 5: Complete Tasks

**Click path:** Case → **Tasks** tab

Check off the first three tasks:

> "The analyst works through the checklist. Click a checkbox, task is done, timestamp recorded, SLA progress tracked. That 5-second toast is an undo button — in case they checked the wrong box."

Point to the task with "Assign to me" button on an unassigned task:
> "Unassigned tasks get this button. An analyst can self-assign without asking their manager. Managers can see which tasks are unowned and reassign in bulk."

---

### STEP 6: Add Findings

**Click path:** Case → **Findings** tab → **New Finding**

Create Finding 1:
- **Title:** Structured deposits to avoid CTR threshold
- **Severity:** High
- **Description:** 11 cash deposits totalling ₹58.5L, all between ₹4.8L–₹6.2L — consistently below the ₹10L CTR threshold. Classic structuring pattern.
- **Evidence:** Link the bank statement document
- Click **Confirm**

Create Finding 2:
- **Title:** 100% of funds transferred to shell companies within 48 hours
- **Severity:** Critical
- **Description:** All deposited funds transferred to 3 entities incorporated <6 months ago with no GST registration or verifiable business activity. Consistent with layering.
- **Evidence:** Link the company registration document
- Click **Confirm**

> "Findings are the formal conclusions of your investigation. Not just notes — formal, evidence-linked, severity-rated conclusions. These feed directly into the STR narrative. The AI uses them to draft your regulatory filing. Your auditor can see exactly which document supports each finding."

**Objection:** *"Our analysts write all this in the narrative directly."*
> "And when the auditor asks 'what evidence supports this claim in your STR?', what do you show them? A paragraph of text? Findings give you a traceable evidence chain: Finding → Evidence Document → AI Log → STR Narrative. Every claim is backed by a linked source."

---

### STEP 7: Log Transactions

**Click path:** Case → **Transactions** tab → **Add Transaction**

Add:
1. Cash deposit ₹52,00,000 — 15 Jan 2024 — Cash → A/C 9876543210 — **Flag it**
2. NEFT ₹19,50,000 — 16 Jan 2024 — A/C 9876543210 → Sunrise Trading — **Flag it**
3. NEFT ₹18,00,000 — 16 Jan 2024 — A/C 9876543210 → GreenLeaf Commodities — **Flag it**

> "The Transactions tab is your flow-of-funds record. ₹52 lakhs came in. Where did it go? This visual trail is exactly what FIU-IND requires in an STR — account numbers, amounts, dates, counterparties. The AI narrative generator reads this data when writing the filing."

---

### STEP 8: Add a Subject (Subject 360)

**Click path:** Sidebar → **Subjects** → **New Subject**

Fill in:
- Type: Individual
- Full Name: Kiran Patel
- PAN: ABCDE1234F
- Account: 9876543210
- Risk Level: High

Go back to the case → right column → **Link Subject** → link Kiran Patel.

> "Now watch what this unlocks."

Go to Subjects → click Kiran Patel's record:

> "Every case this person has ever appeared in, across your entire system, is listed here. Risk score history. Number of STRs filed. All linked cases. This is Subject 360 — a cross-case profile of every entity in your system.
>
> Why does this matter? Money mule networks reuse the same people and accounts across dozens of transactions. Without cross-case linking, each analyst sees one transaction. With Subject 360, you see the network. The same person who appeared in a ₹2 crore suspicious transfer last month is now showing up in this ₹58 lakh case. That pattern is invisible without this feature."

---

## PART 4 — AI FEATURES (15 minutes)

> "Now I want to show you the AI layer. This is what makes OpsOwl different from every other case management system you've seen."

---

### 4a. AI Risk Analysis

**Click path:** Case header → **⚡ Analyse** button

Wait 15–20 seconds. Show the risk score updating in the header.

**Click path:** Case → **AI Insights** tab

> "The AI has now read all three uploaded documents. It extracted:
> - Every account number mentioned
> - Every PAN number
> - Every company name
> - Every transaction amount
> - Every date
>
> And it calculated a risk score of 74 out of 100 — HIGH risk. This score is based on the number of confirmed findings, the severity of those findings, and the transaction patterns.
>
> An analyst who would have spent 2 hours manually extracting these entities just got this in 20 seconds."

Point to the **Risk Score History** chart:
> "Every time analysis runs, the score is recorded. If risk increases as new evidence is added — you see it trending. Your MLRO can see which cases are getting riskier over time."

---

### 4b. AI Copilot — Ask

**Click path:** Case header → **🤖 AI Copilot** button → **Ask tab**

Type: *"What are the key red flags in the uploaded documents?"*

Wait for response.

> "The AI just read all three documents and answered a specific question about them. This is Retrieval-Augmented Generation — the AI doesn't hallucinate. It cites only what's in your documents. Watch — it references specific pages and specific amounts from the bank statement."

Type: *"What is the total amount transferred to Sunrise Trading?"*

> "Your analyst doesn't need to manually search through 50 pages of statements to answer a specific question. They ask in plain English and get an exact answer in 5 seconds."

**Objection:** *"Can we trust AI outputs for regulatory filings?"*
> "Everything the AI produces goes to AI Logs — permanently stored with the model version, prompt version, and response. Nothing is auto-saved. Every AI response requires human review and approval before it affects any case record. The AI is an assistant, not a decision-maker. Your analyst remains fully accountable."

---

### 4c. AI Copilot — Summarize

**Click path:** Copilot panel → **Summarize** tab → **Generate Summary**

> "One click generates a structured summary of all documents — what they contain, key entities, key amounts, key dates. A new analyst assigned to take over this case from a colleague can get up to speed in 2 minutes instead of 2 hours."

---

### 4d. AI Copilot — Draft Note

**Click path:** Copilot panel → **Draft Note** tab → **Draft Note**

> "The AI drafts an investigation note based on the case context. The analyst reviews it, edits it if needed, and approves it. The approved version is saved to the case. The draft is logged separately in AI Logs so you can always see what the AI originally wrote versus what the analyst approved."

---

### 4e. AI Copilot — Extract Tasks

**Click path:** Copilot panel → **Extract Tasks** tab → **Extract Actions**

> "The AI reads the documents and suggests investigation tasks — what else should be done based on what's in these documents. It might find a reference to a third account that the analyst hasn't investigated yet and suggest it as a task. It's like having a senior analyst looking over your junior analyst's shoulder."

---

### 4f. AI Investigation Agent (The Most Powerful Feature)

**Click path:** Copilot panel → **⚡ Agent** tab

> "This is the most powerful feature in the platform. The previous tabs answer one question at a time. The Agent runs an autonomous multi-step investigation."

Type in the goal textarea:
```
Analyse this case for money laundering indicators. Identify all parties,
map the flow of funds, assess the risk level, and produce a structured
investigation report suitable for manager review and STR filing.
```

Click **⚡ Run Agent**.

Watch the live event feed appear:
- `→ Case Context  fetching…`
- `✓ Case Context: AML case INV-0052, High priority…`
- `→ Notes  fetching…`
- `✓ Notes: 2 analyst notes found…`
- `→ Transactions  fetching…`
- `✓ Transactions: 3 flagged transactions, ₹58.5L total…`
- `→ Findings  fetching…`
- `✓ Findings: 2 confirmed findings, Critical + High…`
- `→ Similar Cases  fetching…`
- `✓ Similar Cases: 3 prior cases with structuring patterns…`

When the Investigation Report appears, read it aloud:

> "The agent read the case notes, tasks, findings, transactions, timeline, documents, and searched for similar historical cases — all in one run. It produced a complete investigation report. An analyst who would have spent 3–4 hours compiling this report just got a first draft in 45 seconds.
>
> **This is not replacing your analyst.** The analyst reviews this, makes corrections, approves it, and pastes it into the narrative. But the cognitive work of synthesising information from 6 different sources into a structured report — that 3 hours just became 10 minutes."

**Objection:** *"What if the AI gets something wrong?"*
> "The agent only has read access. It cannot modify any data. Every action it takes is visible in this feed — full transparency into exactly what it read and what it concluded. If the report contains an error, the analyst corrects it before approving. The AI log preserves the original output permanently."

---

### 4g. Generate AI Narrative

**Click path:** Case → **Narrative** tab → **Generate Narrative**

Select type: **STR Suspicion Basis**

Add investigator notes: *"Focus on structuring pattern and shell company transfers. Customer claimed agricultural income."*

Click **Generate Narrative**. Wait 20–30 seconds.

> "The AI just wrote the narrative for your STR filing. It used:
> - Your 2 confirmed findings (with severity and evidence links)
> - Your 3 flagged transactions (amounts, dates, counterparties)
> - The case metadata (risk score, subjects, description)
>
> This is not a generic template. It's a specific, evidence-grounded narrative for this exact case."

Click **Edit** → show editing capability → click **Approve**:

> "The analyst reviews, edits if needed, and approves. Only approved narratives feed into the STR filing. The entire history — AI draft, analyst edits, approval — is preserved. Your regulator can trace every word of your STR narrative back to the evidence it came from."

---

## PART 5 — STR FILING (5 minutes)

**Click path:** Case → **STR** tab *(log in as Manager: ravi.kumar@opsowl.dev)*

> "The STR tab is visible only to Manager and above — your analysts can build the investigation, but filing is a management decision."

Walk through the form:
- Subject details (auto-populated from linked Subject)
- Suspicious activity period
- Total amount involved
- Narrative (auto-populated from the approved narrative from the previous step)
- Suspicion basis

Click **Submit for Approval**:

> "OpsOwl enforces the maker-checker principle required by RBI. The analyst who submitted this STR cannot also approve it. A second manager must review and approve before it can be filed."

Log in as Admin → go to the same case → STR tab → **Approve**:

> "Approved. Now export."

Click **Export XML**:

> "This generates a goAML-compatible XML file — the exact format FIU-IND requires. Your compliance officer uploads this file directly to the goAML portal. No manual data re-entry, no transcription errors."

Click **Export PDF**:

> "And a PDF for your internal records — the complete STR with all fields formatted for archival."

**Why it matters:**
> "Your current STR process: analyst writes Word document → manager reviews → someone manually enters it into goAML → someone else checks the goAML entry → file. 2–3 days, 4 people involved, high error risk.
>
> OpsOwl process: AI drafts narrative from evidence → analyst reviews → manager approves → export XML → upload. 45 minutes, 2 people, zero manual re-entry."

---

## PART 6 — CASE CLOSURE (3 minutes)

**Click path:** Case → **Closure** tab

> "When an investigation is complete, it needs a formal disposition — not just a status change to 'Closed'. The Closure tab captures the outcome."

Fill in:
- Outcome: STR Filed
- Disposition: str_filed
- Summary: Investigation confirmed structuring and layering. STR filed with FIU-IND on 13 Jun 2024. Account flagged for enhanced monitoring. Case closed.

Click **Submit Closure** → Approve as Manager:

> "The case is now formally closed with a documented decision. Your MLRO dashboard will reflect this. Your disposition breakdown chart will update. And if an auditor asks 'why was this case closed?', the answer is right here — with timestamp, approver, and summary."

---

## PART 7 — THE INTELLIGENCE LAYER (8 minutes)

---

### 7a. Workload Dashboard

**Click path:** Sidebar → **Workload**

> "This is the management view. Every analyst and manager in your team, with their current numbers."

Point to the table:
- **Active Cases** — how many cases each person owns right now
- **Open Tasks** — total tasks assigned to them
- **Overdue Tasks** — red flag — tasks past their due date
- **Avg Case Age** — how long cases are sitting before being closed
- **Load Bar** — visual relative workload

> "As a manager, I can see in 3 seconds that Divya has 18 active cases and Arjun has 7. I click Divya's cases, select 5, and bulk-reassign to Arjun. Done. Before OpsOwl, you'd need to email HR for the caseload report, wait a day, get an Excel, sort it manually, email the reassignment to both analysts, confirm receipt. 2 days → 2 minutes."

Scroll down to **AI Risk Leaderboard**:

> "Top 20 cases by AI risk score across the entire organisation. If you want to know which open cases are most dangerous right now, this is your answer. Your MLRO should be looking at this every morning."

---

### 7b. MLRO Dashboard

**Click path:** Sidebar → **MLRO Dashboard**

> "This page is for your MLRO, your compliance head, or your board-level risk committee. It answers the question: how is our AML program performing this month?"

Point to the KPI cards:
- **Cases Opened MTD** — volume of alerts your team processed
- **Cases Closed MTD** — investigation throughput
- **STRs Filed MTD** — regulatory output — directly auditable
- **STRs Pending Review** — backlog — if this is high, you have a process bottleneck
- **SLA Breached (Open)** — the most important number — active regulatory risk
- **Average Case Age** — efficiency metric

Point to the **Disposition Breakdown** chart:
> "Of all cases closed this month, how many resulted in STR filings, how many were no-action, how many led to account closures? This is your program effectiveness chart. If 95% of your cases are 'no action', either your alert thresholds are wrong or your investigators are closing too quickly."

Point to the **Investigator Performance table**:
> "Per-analyst breakdown: active cases, closed this month, STRs filed, SLA breaches. If one analyst has 5 SLA breaches and everyone else has 0, that's a performance issue — or a training issue — that you can act on now."

**Objection:** *"I can get this data from our MIS team."*
> "How long does that take? 2 days for the Monday morning report? This is live. Right now. And it's based on the actual case workflow data — not manually entered numbers. When a case is closed in OpsOwl, the dashboard updates in 30 seconds. There's no data entry step that can be wrong."

---

## PART 8 — ADMINISTRATION & COMPLIANCE (10 minutes)

---

### 8a. User Management

**Click path:** Sidebar → **Users**

> "Every person who touches your investigations system must be formally provisioned. Look at what you have per user: name, email, role, status, last login date, and MFA status."

Click **+ Invite User**:

> "Creating a new analyst: enter email, select role — Analyst — click invite. They receive an email, set their password, and they're in. Role-based access is enforced automatically. An analyst cannot see the STR tab. A viewer cannot create anything. A manager can reassign cases. No custom permissions to configure."

**Demo: Role Levels**

| Role | What they can do |
|------|-----------------|
| Viewer | Read-only — for auditors, regulators, board members |
| Analyst | Full investigation work — create, edit, upload, complete tasks |
| Manager | Everything + STR filing, team oversight, bulk operations |
| Admin | Everything + user management, system configuration |
| Super Admin | Everything + destructive operations (delete cases, purge users) |

> "The Viewer role is specifically designed for your RBI auditor. You give them a Viewer account, they log in, they can see every case, every document, every note, every timeline entry. They never need to call your team asking for reports. Your team never needs to prepare anything special for the audit."

**Demo: MFA**

Click on any user → show MFA status badge:

> "Multi-factor authentication is available and enforced per user. For your compliance staff who have access to STR filings, you can require MFA. If a device is lost or compromised, you invalidate the session from here."

---

### 8b. SLA Policies

**Click path:** Sidebar → **SLA Policies**

> "SLA Policies define what 'on time' means for your organisation. Every combination of Case Type + Priority gets a breach window."

Show a policy table:
- AML + Critical → 24 hours
- AML + High → 48 hours
- Fraud + High → 36 hours
- AML + Medium → 72 hours

> "When an AML Critical case is created, the clock starts. 24 hours later — if it's not closed — the case turns red everywhere in the system. Dashboard, case list, case header. The analyst gets a notification. Their manager gets a notification. Nobody can say 'I didn't know'.
>
> Why does this matter for compliance? RBI requires STR filing within 7 days of suspicion. Your internal SLA of 24 hours for Critical cases gives you a 6-day buffer. If you miss your internal SLA, you have time to catch up before you breach the regulatory deadline."

---

### 8c. Case Templates

**Click path:** Sidebar → **Templates**

> "Templates encode your investigation SOPs. You've been running AML investigations for 10 years. Your senior investigators have developed best practices. Templates capture those practices so every analyst, from day one, follows the same process."

Show the template detail (the AML one you created earlier):
- Pre-filled description
- Pre-set priority
- 8 pre-created tasks

> "When a new analyst joins and creates their first AML case, they select this template. 8 tasks are created automatically. The description is pre-filled. They know exactly what steps to follow. No tribal knowledge lost when a senior analyst leaves."

---

### 8d. Investigation Playbooks

**Click path:** Sidebar → **Playbooks**

> "Playbooks go one step further than templates. While templates pre-fill the case creation form, playbooks automatically apply a task list every time a case of a specific type is created — even if no template is selected."

Show a playbook:
- Trigger: Case Type = AML
- Tasks: 8 ordered investigation steps

> "You can't forget to apply a playbook. The moment any AML case is created — by any analyst, from any source — the playbook tasks are created automatically. Your SOPs are enforced at the system level, not dependent on the analyst remembering to apply a template."

---

### 8e. Auto-Assignment Rules

**Click path:** Sidebar → **Assignment Rules**

> "Rules that route cases to the right analyst the moment they're created."

Show a rule: AML + Critical → Divya Reddy

> "Your senior AML investigator handles all Critical AML cases. The moment a Critical AML case is created — whether by an analyst, by an API call from your CBS, or by the alert queue — it's assigned to Divya automatically. No manager involvement, no delay.
>
> Rules are evaluated in priority order. First matching rule wins. You can build sophisticated routing: Critical cases to senior analysts, Medium cases round-robin to available analysts, Fraud cases to your fraud specialist."

---

### 8f. Audit Logs

**Click path:** Sidebar → **Audit Logs**

> "This is the most important page for regulatory compliance. Every single action taken in OpsOwl is logged here — permanently, immutably."

Scroll through the log. Show columns:
- Timestamp
- User (full name + email)
- Action (case_created, note_added, document_uploaded, str_approved, etc.)
- Resource (case ID, document ID)
- IP Address

> "Your RBI auditor walks in and asks: 'Who approved this STR?' — you filter by action = str_approved, find the exact entry, see the timestamp and the approver's email. 10 seconds.
>
> They ask: 'Was this document available when the STR was filed?' — you filter by document ID, see the upload timestamp. Confirm it was uploaded 2 days before the STR was approved. 10 seconds."

Click **Export CSV**:

> "Export the full audit log for any date range as CSV. This is what you submit to the regulator. It's signed, it's timestamped, it covers every user action."

Point to the critical detail:

> "This audit log is **cryptographically hash-chained**. Every entry contains a hash of the previous entry — like a blockchain. If anyone — including your own database administrator — modifies or deletes a single row, the chain breaks and the tampering is immediately detectable. This is not just logging. It's tamper-evident logging. That's the difference between a log and a regulatory-grade audit trail."

**Objection:** *"Our current system has audit logs."*
> "Does your current system's audit log break if someone modifies it? Can your DBA delete a row without detection? OpsOwl's hash-chained audit trail cannot be silently tampered with. That's a significant compliance differentiator."

---

### 8g. Data Retention

**Click path:** Sidebar → **Data Retention**

> "DPDP Act 2023 requires you to not hold personal data longer than necessary. PMLA requires AML records for 10 years. These are directly conflicting requirements.
>
> OpsOwl handles this automatically. For STR filings, documents, and audit logs — the minimum retention is hard-coded at 10 years. You cannot reduce it even as Super Admin. For internal notes and case files, you configure the retention period yourself.
>
> When your legal team says 'we need to comply with DPDP for customer data', you point to this page. The system enforces it."

---

## PART 9 — INTEGRATIONS (8 minutes)

> "OpsOwl doesn't exist in isolation. Your team uses other systems. Let me show you how OpsOwl integrates with the rest of your technology stack."

---

### 9a. Inbound Events (Your CBS → OpsOwl)

**Click path:** Sidebar → **Inbound Events**

> "Your core banking system fires alerts. Your transaction monitoring system fires alerts. Your fraud detection engine fires alerts. All of these can be sent to OpsOwl via a simple webhook — a single API call.
>
> This page shows every event that arrived. The timestamp, the source system, the raw payload, whether it was successfully processed, and the case it created."

Show an event row. Click it to see the raw payload:

> "Nothing is lost. Every event that arrived is stored here permanently. If a case failed to create because of a processing error, the original event is here — your team can manually trigger it."

---

### 9b. Alert Queue (Triage)

**Click path:** Sidebar → **Alert Queue**

> "Events from your CBS don't automatically become cases. They land in the Alert Queue first. An analyst reviews them — some are false positives, some need investigation.
>
> For each alert, the analyst clicks 'Escalate to Case' and it becomes a full investigation. Or 'Dismiss' with a reason. Every decision is logged."

Show an alert → click **Escalate to Case** (or describe it):

> "The time from 'alert fired in your CBS' to 'investigation case created in OpsOwl' — under 2 minutes, with one click. Currently, how long does that take at your organisation?"

---

### 9c. Outbound Webhooks (OpsOwl → Your Systems)

**Click path:** Sidebar → **Webhooks** → **New Webhook**

Show the webhook configuration:
- URL: your system's endpoint
- Events: case_created, case_status_changed, str_filed

> "When a case status changes to 'Escalated', OpsOwl immediately calls your risk reporting system's webhook. When an STR is filed, your management dashboard gets notified in real time. When a new case is created, your BI tool is updated.
>
> Your operations team doesn't need to manually export data and import it into other systems. OpsOwl pushes it automatically."

---

### 9d. API Keys

**Click path:** Sidebar → **API Keys** → **Create API Key**

> "For systems that need to call OpsOwl programmatically — create a case from your CBS, upload a document from your document management system, query case status from your risk dashboard — API keys provide secure, scoped access.
>
> Each key has a scope: read-only, read-write, or admin. If a key is compromised, revoke it in 2 seconds from this page. Zero downtime, immediate effect."

---

### 9e. Scheduled Reports

**Click path:** Sidebar → **Scheduled Reports** → **New Report**

Configure:
- Report type: SLA Summary
- Frequency: Weekly, every Monday 8am
- Email to: MLRO, Compliance Head

> "Every Monday at 8am, your MLRO receives a CSV with the full SLA summary for the previous week — cases opened, closed, breached, average age, per-analyst breakdown. Without anyone sending it. Without anyone compiling it. It runs automatically from live data."

---

## PART 10 — SEARCH (3 minutes)

**Click path:** Sidebar → **Search**

Type: "Kiran Patel"

> "Full-text search across everything — cases, documents, notes, subjects. Type a name, an account number, a PAN, a company name, a transaction amount. It searches inside documents — not just titles."

Type: "9876543210"

> "That account number appears in 3 cases and 2 documents. Your analyst can find every mention of this account in the entire system in under 5 seconds.
>
> When your fraud investigation team gets a new alert, the first thing they do is search for the account number. If it's appeared in a previous AML investigation, they'll see it here. Cross-case intelligence, instant."

---

## PART 11 — TIMELINE (3 minutes)

**Click path:** Open any case → **Timeline** tab

> "The Timeline is the case's complete, chronological history. Every action by every user — in order."

Scroll through:
- Case created
- Document uploaded
- AI analysis run
- Note added
- Finding confirmed
- Task completed
- STR submitted
- STR approved

> "This is your case's story. A new analyst taking over the case reads the timeline and knows exactly what happened, in what order, by whom. No handover meetings, no lengthy emails, no risk of miscommunication.
>
> For your auditor, this is the evidence that the investigation was conducted properly — every step, timestamped."

---

## PART 12 — PDF EXPORT (2 minutes)

**Click path:** Case header → **📄 PDF Export** button

> "One button generates a complete case summary PDF — case details, timeline, notes, findings, documents list, AI insights, communications log. Everything a regulator or auditor needs to review this case, in a single formatted document.
>
> Your analyst emails this PDF to the auditor. Done. No PowerPoint, no custom report, no manual compilation."

---

## CLOSING — THE ROI ARGUMENT (5 minutes)

> "Let me put numbers to what you just saw."

---

### Time Savings Per Investigation

| Task | Current Process | With OpsOwl | Saving |
|------|----------------|-------------|--------|
| Document review and entity extraction | 2 hours | 20 seconds (AI) | 1h 59m |
| Investigation note writing | 45 mins | 5 mins (AI draft + edit) | 40m |
| STR narrative writing | 2–3 hours | 30 mins (AI draft + edit) | 2h 30m |
| goAML data entry | 1 hour | 2 minutes (XML export) | 58m |
| Status reporting to management | 30 mins/day | 0 (live dashboard) | 30m |
| Case handover to new analyst | 1 hour | 0 (Timeline) | 1h |
| **Total per complex investigation** | **~8 hours** | **~1.5 hours** | **~6.5 hours** |

> "If your team handles 100 complex investigations per month, that's 650 hours saved. At ₹800/hour fully loaded cost for a compliance analyst, that's ₹5.2 lakhs per month in productivity savings. OpsOwl pays for itself."

---

### Risk Reduction

> "The second ROI is risk reduction — harder to quantify, but far more important.
>
> - One missed STR deadline can cost ₹1 lakh to ₹10 lakh in RBI penalties per violation.
> - One RBI show-cause notice costs your team 3 months of preparation time.
> - One regulatory censure costs your institution its ability to launch new products for 12–18 months.
>
> OpsOwl's SLA tracking, maker-checker workflows, and hash-chained audit logs aren't features. They're your compliance insurance policy."

---

### Compliance Posture

> "The third ROI is audit readiness. Today, when RBI calls for an inspection, your team spends 2 weeks preparing — pulling data, compiling reports, formatting documents. With OpsOwl, you hand the auditor a login and a date range. They see everything themselves. Your team spends 0 days on audit preparation."

---

## TECHNICAL QUESTIONS A CTO WILL ASK

**Q: Where does the data live?**
> OpsOwl is built for on-premise deployment. Your data never leaves your data center. We provide Docker containers, Kubernetes manifests, and a complete deployment guide. Your infosec team has full control.

**Q: What AI model is used?**
> By default, we use Qwen 2.5 (7B) running on your own infrastructure — no data sent to OpenAI, Anthropic, or any external API. For clients who want better AI quality and are comfortable with cloud, we support plugging in any OpenAI-compatible endpoint.

**Q: How does it integrate with our CBS?**
> Two ways: (1) Inbound webhook — your CBS calls our API when it generates an alert. (2) We provide an API key — your CBS queries OpsOwl for case status. Both are documented, RESTful, and can be implemented by your integration team in a day.

**Q: What's the uptime SLA?**
> Since it runs on-premise, uptime is determined by your infrastructure. We provide health check endpoints, Docker health probes, and a monitoring runbook. Most clients achieve 99.9% uptime running it on a 2-node setup.

**Q: How many users can it support?**
> The current architecture handles 50 concurrent users comfortably on a single 4-core, 8GB server. For teams above 100 users, we recommend a horizontal scale-out configuration — documented and supported.

**Q: What about DPDP Act compliance?**
> Built-in. Data retention policies, Right-to-Erasure endpoint (with PMLA hold protection), audit logs of all data access. Your DPO can review the compliance documentation.

---

## SUMMARY — WHAT YOU JUST SAW

> "Let me recap the complete investigation lifecycle we just walked through:
>
> 1. **Alert arrived** from CBS → landed in Alert Queue
> 2. **Analyst triaged** → escalated to Case
> 3. **Case auto-assigned** by rule → Divya Reddy
> 4. **Template applied** → 8 tasks created automatically
> 5. **Documents uploaded** → AI extracted entities in 20 seconds
> 6. **Communications logged** → permanent record
> 7. **Notes added** → visible to entire team in real time
> 8. **Tasks completed** → SLA tracked, progress visible
> 9. **Findings documented** → evidence-linked, severity-rated
> 10. **Transactions recorded** → flow of funds mapped
> 11. **Subject linked** → cross-case profile updated
> 12. **AI analysed** → risk score: HIGH 74
> 13. **AI Agent ran** → complete investigation report in 45 seconds
> 14. **AI Narrative generated** → STR draft in 30 seconds
> 15. **STR filed** → maker-checker enforced, goAML XML exported
> 16. **Case closed** → formal disposition recorded
> 17. **MLRO Dashboard updated** → live, no data entry
> 18. **Audit log** → every step, every user, hash-chained
>
> Total time for a complex AML investigation with OpsOwl: **under 3 hours**.
> Current industry average: **3–5 days**.
>
> That is the OpsOwl difference."

---

*For pricing, deployment timelines, and a trial environment for your team — contact: reddybasha524@gmail.com*

*OpsOwl AI · Investigation Workspace · v1.0*
