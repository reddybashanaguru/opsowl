# OpsOwl AI — Product Demo Script
## Audience: Chief Compliance Officer / Chief Risk Officer (NBFC/Fintech)
## Duration: 45 minutes | Format: Screen share + narration

---

> **Before the call**
> - DB seeded: 29 live cases, 6 critical, 115 tasks, 48 AI queries today
> - Logged in as: `admin@opsowl.dev` / `OpsOwl@2024` (Priya Sharma, Admin)
> - Browser: full screen, no other tabs visible
> - URL: `http://localhost:3001`
> - Keep `http://localhost:3001/cases` and `http://localhost:3001/dashboard` pinned

---

## PART 1 — THE HOOK (3 minutes)

### Opening line (say this before you share your screen)

> "Before I show you anything — let me describe a scenario. Your analyst identified a suspicious UPI cluster last Tuesday. Forty-seven high-value transactions, ₹42 lakh, structured just below the CTR threshold. She flagged it. But the STR has to be filed within 7 working days of detection — that's RBI's mandate. Today is Day 6. She's on leave. Her notes are in a Word file on her laptop. The documents are in a folder on the shared drive labelled 'UPI — October.' Nobody else knows where the investigation stands.
>
> The penalty for a late STR can be ₹10,000 to ₹1 lakh per day. More importantly — if FIU-IND asks for the investigation trail, what do you show them?
>
> That's the problem OpsOwl solves. Let me show you what Day 6 looks like when you have this platform."

**[Share your screen now]**

---

## PART 2 — THE DASHBOARD (5 minutes)

**Navigate to:** `http://localhost:3001/dashboard`

### What to say:

> "This is the command center for your compliance operations. Everything your team is working on — right now — is visible here."

**Point to the stats row:**
- "22 open cases — these are active investigations. Nothing is forgotten."
- "115 pending tasks across all cases — every required investigation step, assigned to a specific analyst with a deadline."
- "48 AI queries today — your team has been using the AI Copilot to accelerate analysis."
- "6 critical priority cases — the ones that need your attention today."

**Point to the status breakdown:**
> "10 in progress, 3 pending your review as manager, 4 escalated — these four have been escalated to the Chief Risk Officer level and need leadership attention."

**Point to the SLA breach counter (should show 0):**
> "Zero SLA breaches right now. That's what a well-run compliance operation looks like. This number updates in real time. If it goes up — you'll know before your analyst does."

**Point to Priority breakdown:**
> "6 critical, 13 high priority. This tells your team exactly where to focus tomorrow morning. No more 'which case should I work on first?' conversations."

**Pause for effect:**
> "Notice what's missing here: no Excel file. No Slack message. No 'where is that case?' conversation at 9 AM. It's all here."

---

## PART 3 — THE INVESTIGATION QUEUE (4 minutes)

**Navigate to:** `http://localhost:3001/cases`

### What to say:

> "Here's every open investigation your team has — sortable, filterable, and searchable."

**Show the case list:**
> "You can see case number, title, priority badge, status, who it's assigned to, and the SLA deadline. Let me filter to show only critical cases."

**Click Priority filter → Critical:**
> "Six critical cases. Let me show you what one of these looks like — this first one, the UPI mule network case."

**Scroll to show case: "NACH Mandate Fraud — 340 Accounts Affected"**
> "This one is escalated. 340 accounts, ₹1 crore in unauthorised debits. Your compliance head needs to see this every morning. With OpsOwl, they do."

**Show search:**
> "Search works instantly across all cases. Let me type 'ATM' — see how it filters immediately."

> "Let me type 'KYC' — it finds the KYC forgery case instantly. No more 'I can't find that case' conversations."

---

## PART 4 — INSIDE A CASE: THE FLAGSHIP DEMO (12 minutes)

**Navigate to:** First case in the list — "Suspicious UPI Cluster — Account 9876543210"

> "This is the case I described in my opening. ₹42 lakh, 47 transactions, Day 6 of 7. Let me walk you through exactly what your analyst sees."

### 4a. Case Header (1 min)

**Point to the header:**
> "Case number OWL-2026-00001. Status: In Progress. Priority: Critical. SLA deadline is right here — the system calculated this automatically based on the priority level you set when the case was created.
>
> Assigned to Arjun Mehta. But even if Arjun is on leave, any analyst with access to this case sees everything."

### 4b. Tasks Tab (3 min)

**Click on Tasks tab:**

> "This is the investigation checklist. Every required step — who owns it, what priority, when it's due."

**Point to tasks:**
- "Pull CBS transaction logs — assigned, due in 24 hours."
- "Prepare STR Form A — marked high priority, AI-suggested."
- "Escalate to Compliance HOD — due in 48 hours."

> "That last word — 'AI-suggested.' The AI Copilot read the case description and automatically suggested these investigation steps. The analyst didn't have to remember the STR checklist. The system told them."

**Click on "Prepare Suspicious Transaction Report":**
> "Full description: 'Deadline: 7 working days from detection. Attach CBS export and account profile.' This is your institutional knowledge — captured, repeatable, and assigned. When a new analyst joins, they don't need to be trained on 'what do I do for a UPI fraud case?' The system tells them."

### 4c. Documents Tab (2 min)

**Click on Documents tab:**

> "Three documents uploaded so far. CBS transaction export — 1.2 MB. Account statement for 90 days. STR Form A draft."

**Point to the upload status:**
> "Every document is automatically virus-scanned using ClamAV on upload. AES-256 encrypted at rest. Even if someone physically takes the server, they cannot read these documents without the encryption key.
>
> The CISO at your institution — whoever reviews security — will ask you these exact questions. Here are your answers."

**Point to document list details:**
> "Uploaded by Divya Reddy, 3 days ago. Downloaded by Arjun Mehta, yesterday. This is chain of custody. If a regulator asks 'who had access to this document?' — you have a complete answer."

### 4d. Notes Tab (2 min)

**Click on Notes tab:**

**Point to the pinned note "Initial Investigation Findings":**
> "This note was written by Divya on Day 1. She documented the transaction window, the device fingerprint mismatch, her recommendation to block the card. Notice the timestamp — to the second. And notice this: **she cannot edit it after saving.** That's intentional.
>
> Why? Because a tamper-proof investigation record means nobody can change the story after the fact. If this goes to a regulator audit or — in a serious case — to court, these notes cannot be disputed. The timestamp, the author, the content — all permanent."

**Click on "Escalation Justification":**
> "This is a formal note from the manager. 'Escalating to Cyber Fraud desk. SIM swap event confirmed. STR mandatory.' Three data points. One immutable record. That's the narrative a regulator wants to see — not a WhatsApp group screenshot."

### 4e. Timeline Tab (2 min)

**Click on Timeline tab:**

> "This is the living history of the investigation. Every single event — automatically captured, in real time."

**Scroll through the timeline:**
> "Case created. Case assigned to Arjun. Document uploaded — evidence PDF. AI query run — the AI Copilot was asked about suspicious patterns. Status changed to in_progress. Task created — STR filing. Document uploaded — STR draft.
>
> This is a complete, verified chain of custody. If FIU-IND asks for the investigation chronology, this is it — exported as PDF in 30 seconds. No reconstruction, no 'let me check my email' — it's already assembled."

**Key line:**
> "Your analyst doesn't create this timeline. It creates itself. Every action, every decision, every document — automatically recorded."

---

## PART 5 — AI COPILOT (7 minutes)

**Stay on the case. Navigate to AI Copilot (if button visible in header) or navigate to a fresh case.**

> "Now let me show you the part that's genuinely different from anything else in this space — the AI Copilot."

### 5a. The Ask (3 min)

**Click AI Copilot button.**

> "Your analyst is looking at this case. She's dealing with 8 other cases today. She needs to quickly understand: what are the AML red flags here? Instead of spending 45 minutes reviewing the CBS export — she asks."

**Type in the Copilot:** `What are the suspicious patterns in this case? What are the AML red flags I should be concerned about?`

**While it's processing:**
> "This is using the documents uploaded to the case — the CBS export, the account statement — plus the case description and all previous AI findings. It's not a generic chatbot. It knows this specific case."

**Read the response aloud or point to it:**
> "Three primary findings: structuring below reporting thresholds, velocity 12x above the account's 90-day baseline, and beneficiary accounts opened recently with minimum KYC. And a clear recommendation: file the STR within 7 working days.
>
> That analysis took your analyst 45 minutes in Excel. With the AI Copilot — 90 seconds."

### 5b. AI Task Extraction (2 min)

**Click "Extract Action Items" or similar button (or type):** `What are the specific steps I need to take to complete this investigation?`

**Read the response:**
> "The AI gives you a numbered action list: file STR, obtain bank statements via LOC, verify KYC via UIDAI, preserve digital evidence, escalate to compliance HOD. These are investigation best practices — encoded into the system. The AI doesn't guess. It applies your own compliance checklist to every case."

**Key point:**
> "And here's what makes this different from a generic AI tool: every AI response goes through a human review workflow. Your manager reviews the AI output, approves it, edits it, or rejects it. The approved response becomes part of the permanent case record — with a note that 'Manager approved this AI output.' That's the human-in-the-loop control that regulators actually want to see."

### 5c. AI Logs (2 min)

**Navigate to:** AI Logs section (sidebar)

> "Every single AI query — who asked it, what they asked, what the AI said, what tokens were used, how long it took, and what the reviewer decided — is logged here permanently. Your auditors can see exactly how AI was used in every investigation.
>
> This is not just a compliance feature — it's your defence. When someone asks 'did you just let the AI make this decision?' — you can show them: no, the AI suggested, a qualified human reviewed and approved, and that's in the record."

---

## PART 6 — SLA MANAGEMENT (4 minutes)

**Navigate to:** `http://localhost:3001/sla` (SLA Policies page)

> "SLA — Service Level Agreement — in a compliance context means: how long do you have to complete this investigation? The RBI mandates that STRs be filed within 7 working days. If you miss that window — you're exposed.
>
> OpsOwl has an SLA engine built in."

**Show the SLA policies:**
> "Four policies: critical cases have a 3-day deadline, high priority 7 days, medium 14 days, low 30 days. These are configurable. If your specific regulatory obligation is different — you can set it here."

**Key point:**
> "When a case is created with 'High' priority, the system automatically calculates the SLA deadline and starts counting down. Your analyst sees it. Your manager sees it. Nobody forgets. And when the clock runs out — the system flags it as a breach immediately."

**Navigate back to cases list, filter by priority:**
> "Look at the SLA column in your case list. Every case shows exactly how much time is remaining — or how overdue it is. This replaces a calendar reminder that only the analyst has, on a device that might be out of service."

---

## PART 7 — THE AUDIT MOMENT (4 minutes)

> "Let me show you the moment that every compliance head dreads — and what it looks like when you have OpsOwl."

**Navigate to:** `http://localhost:3001/audit` (Audit Logs section)

> "The scenario: RBI sends an inspection team. They want to see the complete record for a specific investigation. Without OpsOwl — you'd spend 3 days gathering documents, emails, WhatsApp messages, and reconstructing what happened. With OpsOwl:"

**Show the audit log:**
> "Every API call, every login, every document upload, every status change — logged here with timestamp, actor, IP address, and what happened. This is a hash-chained audit log. Each entry includes a hash of the previous entry. If anyone tries to modify or delete a record, the hash chain breaks — and the system immediately shows tampering.
>
> This is the same principle used in blockchain — but purpose-built for compliance investigation records."

**Point to a specific entry:**
> "Login from this IP. Case created by this analyst. Document uploaded. Status changed to escalated. Manager approved the STR. Filed with FIU-IND. Every step, every actor, every timestamp — irrefutable."

**Closing line for this section:**
> "Your auditor asks: 'Can you prove that this investigation was conducted properly and that nobody tampered with the record after the fact?' With OpsOwl, your answer is: yes, and here's the cryptographic proof."

---

## PART 8 — RBAC & TEAM MANAGEMENT (2 minutes)

**Navigate to:** `http://localhost:3001/users`

> "Quick look at access control. Five roles: Super Admin, Admin, Manager, Analyst, Viewer. Each role has strictly defined permissions.
>
> An analyst can work cases assigned to them. They cannot delete documents. They cannot close a case without manager review. They cannot access another analyst's cases without assignment.
>
> Your CISO will ask: 'What's the principle of least privilege?' — this is your answer."

**Point to the team:**
> "Priya Sharma — Admin. Ravi Kumar — Manager. Three analysts. Each person sees exactly what they need to see, and nothing else."

---

## PART 9 — SEARCH & ANALYTICS (2 minutes)

**Navigate to:** Global search bar

**Type:** `STR`
> "Global search — finds cases, documents, notes, and tasks that mention STR. Instant. No waiting."

**Type:** `NACH mandate`
> "Found the NACH mandate fraud case immediately. One search, all context."

**Navigate to:** Analytics page
> "Summary analytics: case resolution time, cases by priority, analyst workload distribution, AI utilization. For your management committee report — this replaces a manual Excel exercise that takes half a day."

---

## PART 10 — ON-PREMISE & DEPLOYMENT (2 minutes)

> "One thing I haven't mentioned yet — and it may be the most important thing I'll say: this entire platform runs on **your own servers.**"

> "Every other modern SaaS compliance tool means your customer data — KYC records, transaction details, STR drafts — leaves your infrastructure and sits in someone else's cloud. That's a data localization problem. That's a regulatory risk. That's an RBI concern."

> "OpsOwl is designed from day one for on-premise deployment. One `docker-compose up` command. Your servers. Your data. Your network. Zero data goes to any cloud — not AWS, not Azure, not our servers. Not even the AI — we can configure the LLM to run on your own infrastructure if that's a requirement."

> "For your IT security team — this is a single Docker Compose stack. PostgreSQL for data, MinIO for document storage, Prometheus and Grafana for monitoring. All open standards. No proprietary lock-in."

---

## PART 11 — CLOSE (4 minutes)

### The "magic moment" recap

> "Let me revisit where we started. Day 6, analyst on leave, STR deadline tomorrow. In the world without OpsOwl — that's a panic, a missed deadline, and possibly a penalty."

> "In the world with OpsOwl: any analyst logs in, opens the case, sees the complete history from Day 1, sees that the STR draft is in the Documents tab, sees that the task says 'needs manager sign-off before submission,' clicks AI Copilot and asks 'is this STR draft complete?' — and files it today."

> "The knowledge didn't walk out the door with the analyst. It stayed in the system."

### The ROI pitch

> "Three numbers your finance team will care about:
>
> One: the penalty for a late STR filing is ₹10,000–₹1 lakh per day. One avoided penalty pays for OpsOwl's annual license.
>
> Two: the analyst time saved — 2 hours per investigation day, 20 cases per month, 5-person team — that's 200 hours per month that goes back to actual investigation rather than Excel management.
>
> Three: when your RBI auditor arrives, you're not spending 3 days preparing a document bundle. You export the case history in 30 seconds."

### Next steps

> "What I'd suggest as a next step is a 30-day proof of concept. We deploy this on a test server at your institution — or a VM we set up for you — and you run 5 real cases through it. Your compliance team uses it for real work. At the end of 30 days, you'll know whether this fits your operation.
>
> We handle the entire setup. Your IT team doesn't need to do anything except open a port. It takes less than a day to go live."

### Questions to invite

- "What's the biggest process pain you have today in your compliance operations?"
- "When your RBI auditors visited last — how long did it take you to produce the investigation records they asked for?"
- "How many STR deadlines did your team come close to missing in the last 6 months?"

---

## APPENDIX: HANDLING TOUGH QUESTIONS

| Question | Answer |
|----------|--------|
| "We already have Jira for case tracking" | Jira has no audit chain, no SLA engine, no STR filing, no document encryption, no AI Copilot, no compliance-specific workflow. It's a project management tool. You cannot show a Jira board to an RBI auditor and call it a compliance investigation record. |
| "What if we need to integrate with our CBS?" | OpsOwl has a full REST API and webhook receiver. Your CBS can POST an alert directly to OpsOwl to auto-create a case. We've also built SIEM integration for SIEM-based alert ingestion. |
| "What about data backup?" | WAL archiving to MinIO, automated daily backups, point-in-time recovery. RBI's data backup requirements are covered out of the box. |
| "How does the AI know about our policies?" | The AI Copilot is contextual — it reads the case content (documents, notes, timeline) to generate responses. You can also provide a policy document as a reference document on any case. Prompt templates can be configured per case type. |
| "What's the pricing?" | Annual license model, no per-user charges. Starter (₹8–12 Lakhs for up to 25 users), Growth (₹18–28 Lakhs for up to 100 users), Enterprise (₹40–60 Lakhs, unlimited). One-time implementation: ₹3–8 Lakhs. Year 1 support is included. |
| "Who else is using this?" | [Answer honestly — this is currently in its production-ready MVP phase, looking for design partners. Position: "We are looking for 3 design partner institutions to go live together, co-shape the product, and get a significantly discounted rate in exchange for a reference case study."] |
| "What if the AI gives wrong advice?" | Every AI output goes through mandatory human review before it becomes part of the case record. The AI recommends — a qualified human decides. That human approval is logged permanently. The AI is a co-pilot, not the pilot. |
| "Is this DPDP compliant?" | Yes. We have a built-in data erasure workflow under India's Digital Personal Data Protection Act. Subject access requests can be fulfilled from the admin panel. Data retention policies are configurable. |

---

## SCREEN FLOW CHEAT SHEET

```
Dashboard              → Show stats, SLA breaches, critical cases
  ↓
Cases List             → Filter by critical → show queue
  ↓
Case Detail (UPI case) → Show header (SLA, status, priority)
  ↓
  → Tasks tab          → Show AI-suggested tasks, ownership, deadlines
  → Documents tab      → Show encrypted vault, chain of custody
  → Notes tab          → Show immutable notes, timestamps
  → Timeline tab       → Show live history, automated events
  ↓
AI Copilot (in case)   → Ask AML question → read response
                       → Extract actions → show task list
  ↓
AI Logs (sidebar)      → Show human review workflow, every query logged
  ↓
SLA Policies           → Show 3/7/14/30 day tiers, configurable
  ↓
Audit Logs             → Show hash-chain, show an entry, explain tamper-proof
  ↓
Users / RBAC           → Show 5 roles, principle of least privilege
  ↓
Search                 → Type "STR", type "NACH mandate"
  ↓
Close                  → 30-day PoC offer
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
| Total cases seeded | 29 |
| Critical cases | 6 |
| Total tasks | 115 |
| AI queries today | 48 |
| Pending AI reviews | 23 |

---

*Document: OpsOwl Demo Script v1.0 — CCO/CRO Audience — May 2026*
