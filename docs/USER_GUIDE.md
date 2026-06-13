# OpsOwl AI — User Guide

**Platform:** AI-Native Enterprise Investigation Workspace  
**Version:** 1.0 (MVP + Phase 8)  
**Audience:** All users — Analyst, Manager, Admin, Viewer  
**Last updated:** 2026-06-13

---

## Quick Start

1. Open the OpsOwl web interface (your IT team will provide the URL)
2. Sign in with your email and password
3. You will land on the **Dashboard** — your personal view of active cases, SLA alerts, and your assigned queue

Your role determines what you can see and do. This guide is organized by role.

---

## Roles Overview

| Role | Who Uses It | What They Can Do |
|------|-------------|-----------------|
| **Viewer** | Auditors, regulators, read-only stakeholders | View cases, documents, notes, timelines. Cannot create or edit anything. |
| **Analyst** | Investigation staff, compliance officers | Create and work cases, write notes, complete tasks, upload documents, log comms |
| **Manager** | Team leads, compliance managers | Everything Analyst can do, plus assign cases, run STR filings, view team workload |
| **Admin** | IT admins, compliance heads | Everything Manager can do, plus manage users, configure settings, bulk operations |
| **Super Admin** | System administrators | Full access including destructive operations |

---

## Section 1: Viewer Guide

As a Viewer, you have read-only access to the system for oversight and audit purposes.

### What You Can See

- **Dashboard**: Summary statistics — open cases, SLA breaches, team activity
- **Cases**: Browse all cases, open any case to read details
- **Case tabs**: Notes, Tasks, Documents, Communications, AI Insights, Timeline
- **Search**: Full-text search across cases, documents, and notes

### What You Cannot Do

- Create, edit, or delete any content
- Access the admin sections (Users, Audit Logs, Settings)
- See the Workload / Analytics page
- See the STR / Suspicious Transaction Report tab

### Navigation Tips

- Use the sidebar **Search** to find any case, document, or note by keyword
- On the Cases list, use the Status filter dropdown to focus on specific stages (Open, In Progress, etc.)
- On a case, the **Timeline** tab shows a full chronological history of every action taken

---

## Section 2: Analyst Guide

Analysts are the primary users of OpsOwl. This section covers your full daily workflow.

### Daily Workflow

1. **Start from Dashboard** → check "My Queue" (top 5 assigned cases)
2. **Open your most urgent case** → the SLA countdown shows how much time remains
3. **Work the case**: add notes, complete tasks, upload documents
4. **Log communications**: every call/email related to the case should be logged in Comms
5. **Run AI analysis** to get risk scoring and entity extraction
6. **Update case status** as work progresses

### Cases

**Creating a Case**
- Sidebar → Cases → New Case (top-right button)
- Fill in: Title, Description, Priority, Due Date, Type
- Optionally assign to yourself or a colleague (if your role allows)

**Case Statuses**
| Status | Meaning |
|--------|---------|
| Draft | Not yet active |
| Open | Active, unstarted |
| In Progress | Being worked |
| Pending Review | Waiting for manager sign-off |
| Escalated | Needs urgent attention |
| Closed | Completed |

**Changing Status**
- Open a case → status buttons appear in the case header
- Routine transitions (Open → In Progress, In Progress → Pending Review) happen on click with no confirmation
- Consequential transitions (Escalate, Close, Reopen) show a reason modal

**Case Filters**
- Filter by multiple statuses simultaneously using the status checkbox dropdown
- Search by case number or title in the search bar
- Use "My Cases" toggle to see only your assigned cases

### Notes

- Go to a case → **Notes** tab
- Click "Add Note" → select type (Internal / Customer-facing / Escalation)
- Notes are plain text or rich text (bold, bullets, links)
- Notes appear on the Timeline automatically

### Tasks

- Go to a case → **Tasks** tab, or sidebar → **Tasks** for cross-case view
- Create a task: "Add Task" → fill Title, Due Date, Priority, Assignee
- Complete a task: click the checkbox — a 5-second undo toast appears if you clicked accidentally
- The **Done** tab shows completed tasks separately
- **"Assign to me"** shortcut appears on unassigned tasks

### Documents

- Go to a case → **Documents** tab
- Upload: drag a file or click the upload zone (PDF, DOCX, XLSX, images up to 50 MB)
- All uploads are virus-scanned automatically
- AI will extract text and analyse the document after upload (status shows: Queued → Processing → Done)
- Download or delete documents from the document card

### Communications

- Go to a case → **Comms** tab
- Log a communication: "Log Communication" → fill Channel (Phone/Email/In-Person), Direction (Inbound/Outbound), Subject, Notes
- Logged comms appear on the case Timeline

### AI Features

**AI Risk Scoring**
- Runs automatically when AI analysis is triggered
- Shows a score (0–100) and level (Low / Medium / High / Critical) in the case header
- Updates after each re-analysis

**AI Copilot** (🤖 button in case header, or Cmd+Shift+A / Ctrl+Shift+A)

Opens a slide-over panel with five tabs:

| Tab | What It Does |
|-----|-------------|
| **Ask** | Ask a free-text question — answered using RAG over your uploaded documents |
| **Summarize** | One-click structured summary of all documents in the case |
| **Draft Note** | AI drafts an investigation note; you review and approve before it is saved |
| **Extract Tasks** | Pulls suggested investigation tasks out of uploaded documents |
| **⚡ Agent** | Autonomous investigation — see below |

All Copilot responses are logged in **AI Logs** (case tab) and require analyst review before they are saved to the case record.

**⚡ AI Investigation Agent** (Copilot → Agent tab)

The Agent is an autonomous investigator. Unlike the other tabs (which answer a single question), the Agent plans and executes a multi-step investigation on its own.

*How to use it:*

1. Open the Copilot panel (🤖 button or Cmd+Shift+A)
2. Click the **⚡ Agent** tab — it has a distinct cyan colour
3. Type your investigation goal in plain language, for example:
   - "Analyse this case for suspicious transaction patterns and summarise the risk"
   - "Identify all entities involved and flag any red flags"
   - "Produce a structured investigation report suitable for manager review"
4. Click **⚡ Run Agent**
5. Watch the live feed as the agent works — each step is shown in real time:
   - *Italic grey text*: the agent's reasoning
   - **→ Case Context / Notes / Findings…**: tool being read (all read-only)
   - **✓ Notes: 2 analyst notes found**: result preview
6. When finished, the **Investigation Report** appears — a structured markdown summary ready to copy into a note or STR narrative
7. Click **Run again** to start a new investigation with a different goal, or **Cancel** to stop mid-run

*What the agent can read (all read-only, no data is modified):*

- Case metadata and context
- Analyst notes
- Tasks (open and completed)
- Findings and evidence links
- Transaction records (flow of funds)
- Case timeline (full event history)
- Uploaded documents
- Similar historical cases (via AI similarity search)

*Stats shown after completion:*  
Number of reasoning iterations · total tokens used · elapsed time

*Tips:*
- The more specific your goal, the more focused the report — "look for PEP exposure and cross-border transfers" gives better output than "analyse the case"
- The agent does not save anything automatically — copy the report text into a Note or the STR Narrative field
- If the AI service is unavailable, an error banner appears; the agent can be re-run once the service recovers

**Re-run AI Analysis** (⚡ button in case header)
- Re-extracts entities from new documents and updates risk score
- Takes 5–30 seconds depending on document volume

**AI Insights tab**
- Full extracted entity list (PAN, accounts, phones, companies)
- Risk score history over time
- Full AI narrative summary

### Search

- Sidebar → Search
- Type any keyword, case number, document name, or entity value
- Results are grouped by type: Cases, Documents, Notes

---

## Section 3: Manager Guide

Managers can do everything Analysts can do, plus additional oversight and filing capabilities.

### Team Workload

- Sidebar → **Workload** (under Intelligence)
- Shows all analysts and managers with: active case count, open task count, overdue tasks, avg case age
- Visual load bars for quick comparison
- **AI Risk Leaderboard**: top 20 cases by risk score — useful for triage

### Assigning Cases

- From any case detail → right column "Assigned To" → click the name or "Unassigned — click to assign" to open the assignee picker
- From the Cases list → select multiple cases via checkboxes → **Bulk Reassign** in the action bar

### Bulk Operations

- Select cases using checkboxes in the Cases list
- **Change Status**: update all selected cases to a new status in one action
- **Reassign**: assign all selected cases to a different analyst

### STR / Suspicious Transaction Reports

- Open a case → **STR** tab (visible to Manager+)
- Complete all required fields: subject details, transaction details, narrative
- Submit for dual authorization (maker-checker): another manager/admin must approve before filing
- Export as XML for FIU-IND submission or as PDF for internal records

### Case Templates

- Sidebar → Settings → Templates
- Create templates with pre-filled description, task list, and priority for common investigation types
- When creating a new case, select a template to auto-populate the form

### Approving STRs

- When an analyst creates an STR in "Submitted" state, you receive a notification
- Open the case → STR tab → Review → Approve or Reject with comments
- Approved STRs can be exported by admins

---

## Section 4: Admin Guide

Admins manage the platform for your organization.

### User Management

- Sidebar → **Users**
- Invite a new user: enter email + role → they receive an invitation email
- Change a user's role: click the role badge on any user row → select new role
- Deactivate a user: use the deactivate action (user loses access immediately, data is retained)

### SLA Policies

- Sidebar → **SLA Policies**
- Create policies: define breach time (hours) for each combination of case type + priority
- Example: `Fraud + Critical → 4 hours`, `AML + High → 24 hours`
- SLA countdown is visible on every case; breached cases appear in the Dashboard Attention panel

### Settings

**Auto-Assignment Rules** (Sidebar → Auto-Assignment)
- Create rules to automatically assign new cases to analysts based on case type or priority
- Rules are evaluated in order; first matching rule wins

**Inbound Events** (Sidebar → Inbound Events)
- View all events received via the inbound API (webhook events from CBS, LOS, or other systems)
- Each event shows raw payload, processed status, and the case it created (if any)

**Outbound Webhooks** (Sidebar → Webhooks)
- Register webhook URLs to receive real-time notifications when cases change status
- Supports case_created, case_status_changed, str_filed events
- Each delivery shows status code and response body for debugging

**Scheduled Reports** (Sidebar → Scheduled Reports)
- Create automated CSV reports emailed on a schedule (daily/weekly/monthly)
- Report types: Active Cases, SLA Summary, Team Workload, STR Filing Summary

**API Keys** (Sidebar → API Keys)
- Create API keys for system integrations (your CBS, LOS, or BI tool)
- Each key has a scope (read / write / admin)
- Keys can be revoked instantly

### Audit Logs

- Sidebar → **Audit Logs**
- Every action in the system is logged with: timestamp, user, action, affected resource
- The audit log is cryptographically hash-chained — entries cannot be modified or deleted without detection
- Filter by date range, user, or action type
- Export as CSV for regulatory submission

### Bulk Operations (Admin-Only)

- Select cases via checkboxes in Cases list
- **Delete**: permanently delete selected cases (shows confirmation modal)
- Also: Change Status + Reassign (same as Manager)

### Data Retention

- Sidebar → **Data Retention**
- Configure how long each data type is retained before archival/deletion
- PMLA-regulated resources (STRs, documents, audit logs) are hard-coded to 10 years minimum and cannot be reduced

---

## Section 5: Notifications

The notification bell (🔔) in the top-right of the sidebar shows your unread count.

**You are notified when:**
- A case is assigned to you
- A task is assigned to you
- A task you created is completed
- A case you're watching has a status change
- An SLA is about to breach (1 hour warning) or has breached

Click the bell to open the notification dropdown. Mark individual notifications as read, or dismiss all.

---

## Section 6: Common Workflows

### How to Investigate a Suspicious Transaction

1. **Create a case**: New Case → Type: "Suspicious Transaction" → Priority: High
2. **Upload documents**: Transaction records, account statements, KYC documents
3. **Wait for AI processing**: AI extracts entities (account numbers, PAN, amounts) and scores risk
4. **Add investigation notes**: Document your findings at each step
5. **Log all communications**: Calls with compliance, customer interviews, etc.
6. **Complete tasks**: Each required check becomes a task (e.g., "Verify KYC documents", "Check CIBIL")
7. **Change status to Pending Review**: When investigation is complete
8. **Manager reviews and files STR**: If suspicious activity is confirmed

### How to Respond to an RBI Audit

1. Auditor gets a **Viewer** account
2. They can search cases by date range, case type, status
3. Each case shows: full timeline, all documents, all notes, all communications
4. The audit log (Admin → Audit Logs) shows every action taken in the system
5. Export case as **PDF** (📄 button in case header) for offline submission
6. Export audit logs as **CSV** (Export button on Audit Logs page)

### How to Use the AI Agent for a Quick Investigation Report

1. Open any active case
2. Click **🤖 AI Copilot** in the case header (or press Cmd+Shift+A)
3. Switch to the **⚡ Agent** tab
4. Type your goal: `"Produce a structured AML investigation report covering all parties, transactions, and risk indicators"`
5. Click **⚡ Run Agent** — the agent typically completes in 10–30 seconds
6. When "Investigation Report" appears, review the output
7. Copy the report text → go to the **Notes** tab → "Add Note" → paste as an Internal note
8. Alternatively, paste the narrative directly into the **STR** tab → Narrative field

This replaces the manual step of reading every tab and writing a summary from scratch.

### How to Onboard a New Analyst

1. Admin → Users → Invite → enter email, role = Analyst
2. Analyst receives email invitation and sets their password
3. Admin creates an auto-assignment rule (optional) so new cases route to them
4. Analyst logs in and sees their My Queue on the Dashboard (empty on first day)

---

## Section 7: Keyboard & Navigation Tips

- **Back to Cases list**: Every case page has a `← Cases` link in the top-left of the sub-navigation bar
- **Tab navigation**: Case detail has tabs — Case (overview), Notes, Tasks, Documents, Comms, AI Insights, Timeline, AI Logs. Managers also see STR.
- **Status filter**: On the Cases list, the status dropdown allows multi-select — hold common filters for quick triage views
- **Infinite scroll**: Cases, Documents, Audit Logs all use infinite scroll — just scroll down to load more

---

## Section 8: Frequently Asked Questions

**Q: I uploaded a document but it shows "Processing" for a long time — is it stuck?**  
A: Processing can take 1–5 minutes for large PDFs or images (OCR is required). If it's been more than 10 minutes, ask your admin to check the document processor status.

**Q: I can't see the STR tab on a case.**  
A: STR is only visible to Manager, Admin, and Super Admin roles. Contact your admin if you need access.

**Q: Can I edit a note after I've saved it?**  
A: Notes can be edited by the original author or by admins. The edit is recorded in the Timeline.

**Q: The AI analysis says "no entities found" — what does that mean?**  
A: Either the uploaded documents were image-only (no OCR text), or the AI didn't detect structured entity patterns. Try uploading clearer source documents or add entity information manually in the case notes.

**Q: I accidentally closed a case — can I reopen it?**  
A: Yes. Open the case → use the "Reopen" status transition (requires a reason comment). Analysts can reopen cases they own; managers can reopen any case.

**Q: I can't find the bulk assign / bulk status buttons.**  
A: These require at least one case to be selected (checkbox on the left of the case row). Select one or more cases and the action bar will appear at the top of the list. Note: bulk assign/status is not available to the Analyst role.

---

*For technical support, contact your system administrator.*  
*For platform questions, contact: reddybasha524@gmail.com*
