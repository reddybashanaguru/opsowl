# OpsOwl AI — Demo Script
**Audience:** CCO / Head of Compliance / AML Head at NBFC or Bank
**Duration:** 30–45 minutes
**Persona:** You are the Chief Compliance Officer evaluating OpsOwl AI as your investigation platform.

---

## Application Usage Log
*This section records what was actually observed and done during live walkthroughs of the application. Updated as each feature is demoed.*

### Feature 1 — Users & Roles ✅
- **Actual URL:** `http://localhost:3001/users`
- **Observed UI:** Role filter chips (Viewer·1, Analyst·3, etc.) show per-role counts and act as filter. Search input. Table with Name, Email, Role badge, Status, Actions columns.
- **Fix applied during demo:** Removed redundant `<select>` role dropdown — chips already handled filtering. One mechanism is cleaner.
- **Button label confirmed:** Top right = **+ Invite User**. Modal button = **Create User** (not "Save").
- **Inline role change:** Hover over a role badge → pencil icon appears → click to change role without opening a form.
- **Deactivate:** Button on each row, requires confirmation. Past actions remain in audit trail.
- **Sample data used:** Rahul Desai / rahul.desai@opsowl.dev / Analyst / Test@1234

### Feature 2 — SLA Policies ✅
- **Actual URL:** `http://localhost:3001/sla`
- **Observed UI:** 4 fixed policies (one per priority level). No "Create" — policies are fixed, only editable. Edit button per row.
- **Three fields explained:**
  - **Warning Hours** = private notification to analyst at X hours elapsed (e.g. 2h for Critical)
  - **Warning % of Breach** = when case row turns AMBER across the platform (e.g. 80% of 4h = 3h)
  - **Breach Hours** = hard regulatory deadline; case turns RED, manager alerted
- **Progress bar label decoded:** `3h → 4h` = amber starts at 3h (80% of 4h), breach at 4h. Calculated as `breach_hours × warn_pct / 100 → breach_hours`
- **Three-level timeline for Critical:** 2h = analyst notification → 3h = row turns amber → 4h = breach + manager alert
- **Default values:** Critical (4h breach), High (1d), Medium (3d), Low (7d)

### Feature 3 — Case Templates ✅
- **Actual URL:** `http://localhost:3001/settings/templates`
- **Observed UI:** Template cards showing name, ACTIVE/INACTIVE badge, priority badge, description, tags, task count, updated date. **+ New Template** button opens a modal.
- **Form fields in modal:** Template Name, Description Template, Default Priority (chips: CRITICAL/HIGH/MEDIUM/LOW), Tags (comma-separated), Active toggle, Task Checklist (with + Add task), Create Template button.
- **No Case Type field** — templates do not have a case type. They are manually picked at case creation.
- **Active toggle** — when ON: label is cyan "Active — visible in case picker". When OFF: grey "Inactive — hidden from case picker". Toggle uses `p-0` on the button and explicit `left-0.5` anchor on the thumb to prevent overflow.
- **Task Checklist in template** — optional starter tasks created on the case when the template is applied. Different from Playbooks (which are auto-applied by case type). Templates = manual pick + starter tasks. Playbooks = auto-applied mandatory procedure.
- **Fix applied during demo:** Toggle thumb was overflowing outside the track. Root cause: no explicit `left` anchor on absolute thumb + browser default button padding. Fixed with `p-0` + `left-0.5`.
- **Sample template created:** AML — Suspicious Transaction / HIGH / tags: structuring, threshold, aml

### Feature 4 — Investigation Playbooks ✅
- **Actual URL:** `http://localhost:3001/settings/playbooks`
- **Observed UI:** List of playbook cards showing name, case type tag (cyan), active/inactive badge, description, step count, expand chevron. **New Playbook** button (was violet — fixed to cyan for consistency).
- **Form fields (inline panel):** Playbook Name, Case Type (now a dropdown — see improvement below), Description, Steps (each step: title, priority dropdown, due hours input, dependency dropdown, optional description). + Add step link. Create Playbook button.
- **Step dependency dropdown:** "No dependency" = step available immediately. "After step N" = step is BLOCKED until step N is marked complete. Enforces investigation sequence, not just a checklist.
- **Case Type field — improvement applied:** Originally free-text input causing typo risk (e.g. "AML" vs "aml" silently breaks playbook trigger). **Replaced with managed Case Types registry:**
  - Migration 00043: `case_types` table seeded with 7 default types: `aml`, `fraud`, `kyc`, `sanctions`, `cyber`, `insider`, `pep`
  - API: `GET /api/v1/case-types` (all users), `POST` / `DELETE` (admin only)
  - `CaseTypeSelect` component: dropdown from registry + inline "Create new type…" option
  - Applied to: Playbooks form, Case creation form
  - DELETE is blocked if any cases use the type
- **Sort order fix:** Playbooks list now sorted by `created_at DESC` — newest first. Was previously sorted by case_type, name.
- **E2E test data cleaned:** 7 "E2E AML Playbook" entries deleted via API before demo.
- **Real-world example playbook — Sanctions Hit Review:**

| # | Title | Priority | Due Hours | Dependency |
|---|-------|----------|-----------|------------|
| 1 | Freeze or hold the flagged transaction immediately | critical | 1 | No dependency |
| 2 | Verify identity match — confirm same person, not false positive | critical | 3 | After step 1 |
| 3 | Collect full KYC, beneficial ownership, source of funds documents | high | 8 | After step 2 |
| 4 | Escalate to Compliance Head with match evidence and documents | critical | 10 | After step 3 |
| 5 | If confirmed match — file report with FIU-IND, freeze account permanently | critical | 24 | After step 4 |
| 6 | Document final disposition — cleared / confirmed / escalated to regulator | high | 48 | After step 5 |

### Feature 5 — Assignment Rules ✅ (5/40)
- **Actual URL:** `http://localhost:3001/settings/assignment-rules`
- **Observed UI:** Rule cards showing rule name, priority badge, case type badge, match summary, assign-to. Numbered list (1, 2, 3…) reflects evaluation order. **+ Add Rule** button top right opens modal.
- **Modal fields:** Rule Name, Priority Order (lower = higher priority), Description, Match Conditions section (Priority dropdown + Case Type dropdown), Assign To section (Role round-robin toggle vs Specific User toggle).
- **Match Conditions — two fields:**
  - **Priority** — dropdown: Any priority / Critical / High / Medium / Low
  - **Case Type** — managed registry dropdown (same `CaseTypeSelect` component used in Playbooks/Case creation)
  - Both can be left blank = matches any case. Combined = both must match (AND logic).
- **Assign To — two modes:**
  - **Role (round-robin):** picks the analyst/manager/admin with fewest open cases at assignment time. Load-balances automatically.
  - **Specific User:** dropdown of real user names + role label (e.g. "Divya Reddy (analyst)"). Previously was a raw UUID text input — fixed to proper user picker.
- **Rule card display:** Shows priority badge (colour-coded) + case type chip (cyan monospace) side by side. "Assign:" shows user full name (not UUID) or "least-loaded {role}".
- **Evaluation order:** Rules run top-to-bottom by `priority_order` when a case is created. First match wins. Rule 1 is evaluated before Rule 2.
- **Improvements applied during demo:**
  1. Added `match_case_type` field — migration 00044, backend Rule struct + Evaluate engine + CRUD SQL, frontend form + card display
  2. Replaced raw UUID text input for "Specific User" with a proper `<select>` populated from `/api/v1/users`
  3. Rule card now shows full user name instead of truncated UUID
  4. List container scrollable (`max-h-[calc(100vh-240px)]`) — consistent with Playbooks page improvement
- **9 production rules defined (create in this order):**

| Priority Order | Name | Match Priority | Match Case Type | Assign To |
|---|---|---|---|---|
| 10 | Sanctions Critical → Divya Reddy | Critical | sanctions | Specific User: Divya Reddy |
| 20 | Critical (any type) → Manager | Critical | (any) | Role: Manager |
| 30 | AML High → Sneha Patel | High | aml | Specific User: Sneha Patel |
| 40 | PEP → Manager (EDD Required) | (any) | pep | Role: Manager |
| 50 | AML (routine) → Analyst Pool | (any) | aml | Role: Analyst |
| 51 | Fraud → Analyst Pool | (any) | fraud | Role: Analyst |
| 52 | KYC → Analyst Pool | (any) | kyc | Role: Analyst |
| 60 | Insider / Cyber → Manager | (any) | insider / cyber | Role: Manager |
| 100 | Fallback → Analyst Pool | (any) | (any) | Role: Analyst |

- **System fallback (no rules configured):** Engine falls back to least-loaded manager → admin → super_admin. Nothing ever stays unassigned.
- **Key design principle:** Most specific rules (lowest priority_order) first. Catch-all (100) last. Rules 1 and 3 use Specific User; all others use Role (round-robin).

---

## Opening Line
> "What I'm going to show you is the complete investigation lifecycle — from the moment an alert fires in your TMS, all the way to a formally closed, audit-defensible case ready for regulatory examination. Everything runs on your own server — no cloud, no data leaving your network."

---

## MODULE 1 — Administration Setup

### Feature 1 — Users & Roles
**URL:** `/users`

**Why we built this:**
Compliance teams have multiple personas — analysts who investigate, managers who review and approve, auditors who should only read, and admins who configure the system. Without role separation, a junior analyst can accidentally close a case, an auditor can tamper with records, and there is no way to enforce accountability. When someone leaves the organisation, their access needs to be revoked immediately — not after IT raises a ticket three days later.

**How it helps:**
- Role-based access ensures every person can only do what their role permits — no accidental or unauthorised changes
- Managers get full visibility; analysts work only on their assigned cases; viewers (internal auditors, external examiners) can read everything but change nothing
- One-click deactivation instantly removes access without deleting any history — the audit trail of all their past actions remains intact
- Role chips show count per role so managers can see team composition at a glance

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

**Why we built this:**
Regulators — RBI, FIU-IND, SEBI — define turnaround time requirements for investigations. Without an enforced SLA system, cases age silently in queues. There is no automated alert when a deadline is approaching, no visual signal for managers, and no audit evidence that cases were resolved within regulatory timeframes. In examination findings, "case aged beyond regulatory TAT" is one of the most common deficiencies cited.

**How it helps:**
- Every case gets a SLA clock the moment it is created — no manual tracking required
- Three-layer system gives advance warning before a breach, not just a notification after it happens
- Amber visual on dashboard and case list makes the entire team aware — manager sees it without being told
- Breach events are logged in the audit trail, giving examiners a complete TAT compliance record

**What to show:**
- Four fixed policies — one per priority level: Critical, High, Medium, Low
- Click **Edit** on Critical to open the three-field form and walk through each field in detail (see below)
- Save and return to show the updated bar
- Explain SLA countdown starts the moment a case is created and is visible on every case row

---

#### SLA Field-by-Field Explanation (say this while Edit is open on Critical)

There are **three independent controls** per policy. Each controls a different layer of the SLA system:

**1. Warning Hours (e.g. `2` for Critical)**
- This is the **early notification trigger** — a private alert sent to the assigned analyst.
- When a case has been open for this many hours without resolution, the system proactively notifies the analyst: "your Critical case is approaching its deadline."
- For Critical with `Warning Hours = 2`: notification fires at the **2-hour mark** from case creation. There are still 2 hours left — this is a heads-up, not a panic alert.
- This fires **before** anything turns amber on the dashboard. Only the analyst receives it.
- Think of it as: *"Wake up the analyst early so they are not scrambling at the last minute."*

**2. Breach Hours (e.g. `4` for Critical)**
- This is the **hard regulatory deadline**.
- When a case has been open for this many hours, it officially **breaches SLA**.
- At breach: the case row turns **red** everywhere, the dashboard "Attention Required" count increments, and the **manager is alerted automatically**.
- This is the number examiners ask about — "was the investigation completed within X hours?"
- For AML: set Critical to `24` to match the RBI/FIU-IND 24-hour STR preliminary action requirement.

**3. Warning % of Breach (e.g. `80` for Critical)**
- This controls the **visual amber indicator** across the entire platform — dashboard, cases list, case detail header.
- When a case ages past `warn_pct%` of its breach deadline, every case row and panel turns amber.
- For Critical: `80% × 4h = 3.2h ≈ 3h` → case row turns amber at the **3-hour mark**.
- This is visible to everyone — managers, analysts, admins — not just the assigned user.
- The `3h → 4h` label on the progress bar is calculated as `(breach_hours × warn_pct / 100) → breach_hours`.

**Why Warning Hours and warn_pct are intentionally decoupled:**
The notification (Warning Hours) fires earlier as a private signal to the analyst. The visual amber (warn_pct) fires later as a public signal to the whole team. Two different audiences, two different trigger points. You can tune them independently.

---

#### How the three levels work together (Critical: 4h breach)

```
Case created
    |
    ├─── 2h elapsed  →  NOTIFICATION sent to analyst only (Warning Hours = 2)
    │                   "Your Critical case needs attention — 2 hours remaining"
    │
    ├─── 3h elapsed  →  CASE ROW TURNS AMBER on dashboard + cases list (warn_pct = 80%)
    │                   Visible to entire team — manager sees it without being told
    │
    └─── 4h elapsed  →  BREACH — case turns RED, manager alerted (Breach Hours = 4)
                         SLA missed — shows in Attention Required panel + analytics report
```

#### Progress bar label decoded (`3h → 4h` on the right)
- **Left number** = `breach_hours × warn_pct / 100` rounded = when the amber visual starts
- **Right number** = `breach_hours` = when breach occurs
- **Yellow fill** = the safe investigation window (0% to warn_pct%)
- **Grey unfilled** = the amber danger window (warn_pct% to 100%)

**Default thresholds (pre-configured):**

| Priority | Notification at (Warning Hours) | Row turns amber at (warn_pct) | Hard breach (Breach Hours) | Bar label |
|----------|--------------------------------|------------------------------|---------------------------|-----------|
| Critical | 2h | 3h (80% of 4h) | 4h | `3h → 4h` |
| High | 20h | 19h (80% of 24h) | 1 day (24h) | `19h → 1d` |
| Medium | — | 2d 10h (80% of 3d) | 3 days | `2d 10h → 3d` |
| Low | — | 5d 14h (80% of 7d) | 7 days | `5d 14h → 7d` |

**Demo line:**
> "Three independent controls per priority level. Warning Hours sends a private notification to the analyst early — before anything turns amber — so they have time to act. Warning Percent turns the case row amber on the dashboard, visible to the whole team. Breach Hours is the hard regulatory deadline — that is what examiners check. For Critical AML: notified at 2 hours, amber at 3 hours, breach at 4. No case silently ages past its deadline."

---

### Feature 3 — Case Templates
**URL:** `/settings/templates`

**Why we built this:**
Every AML investigation starts the same way — same description structure, same priority, same standard tags, same initial task checklist. Without templates, analysts fill the same fields from scratch every time, introduce inconsistencies in description quality, and sometimes set the wrong priority. Inconsistent case setup leads to inconsistent SLA assignment and inconsistent reporting. New investigators especially waste time figuring out what to write in the description and what tasks to create first.

**How it helps:**
- Analysts pick a template from the case picker and every field is pre-filled — description, priority, tags, and an initial task checklist
- Correct priority is set from the start so the SLA clock begins accurately
- Tags are pre-applied so cases are categorised consistently from creation
- Task checklist auto-creates the first set of tasks the moment the template is applied — no waiting for a playbook match
- Templates can be toggled Active/Inactive without deleting — deactivate a template during process reviews without losing it
- Reduces training overhead — the template guides new investigators on what a well-formed case looks like

**Form fields (modal — "+ New Template"):**
1. **Template Name** *(required)* — e.g. `AML — Suspicious Transaction`
2. **Description Template** — pre-fills the case description when applied. Write it like a brief for the analyst.
3. **Default Priority** — clickable chips: CRITICAL / HIGH / MEDIUM / LOW
4. **Tags** *(comma-separated)* — pre-applied to the case on creation
5. **Active toggle** — "Active — visible in case picker". Inactive templates are hidden from analysts but not deleted.
6. **Task Checklist** — tasks added here are automatically created on the case when this template is applied. Each task has a title, optional description, and priority (critical/high/medium/low). Click **+ Add task** to add more. Counter shows "0 task(s)" updating live.
7. **Create Template** button

**What to show:**
- Click **+ New Template**
- Fill in the form with AML sample data below
- Add 3 tasks to the checklist using **+ Add task**
- Toggle Active is on by default — point it out: "inactive templates are hidden from analysts without being deleted"
- Click **Create Template**
- Show the new card appears in the list with ACTIVE badge, HIGH priority, tags, and task count

**Key talking point — Templates vs Playbooks:**
- **Templates** are manually selected from the case picker when creating a case. They pre-fill description, priority, tags, and create a starter checklist. Analyst chooses which template to use.
- **Playbooks** (Feature 4) are automatically applied based on case type — no analyst choice required. Best for mandatory procedures.
- Use both: template for fast case setup, playbook for enforced investigation procedure.

**Demo line:**
> "When an analyst creates a new AML case, they pick this template from the picker. The description is pre-filled, priority is set to High — SLA clock starts correctly — tags are applied, and the first three tasks are already waiting. They do not start from a blank page. And if we update the template, every new case from that point gets the updated version — no retraining needed."

**Sample data:**
- Template Name: `AML — Suspicious Transaction`
- Description: `Customer flagged for suspicious transaction activity. Review transaction history, KYC documents, and counterparty profiles to determine whether an STR is required.`
- Default Priority: `HIGH` (click the HIGH chip)
- Tags: `structuring, threshold, aml`
- Active: on (default)
- Task Checklist:
  1. Task: `Review customer KYC and CDD documents` — Priority: High
  2. Task: `Analyse transaction history for last 6 months` — Priority: High
  3. Task: `Check counterparty risk profile` — Priority: Medium

---

### Feature 4 — Investigation Playbooks
**URL:** `/settings/playbooks`

**Why we built this:**
Under pressure — high case volumes, tight deadlines — investigators, especially junior ones, skip steps. A KYC check is missed. A counterparty is not screened. The STR decision is made without documenting the rationale. In regulatory examinations, missed steps become findings. There is no way to enforce investigation procedure in Excel or email — it relies entirely on individual discipline.

**How it helps:**
- Standardises investigation procedure across every analyst, every case, every time
- Tasks are auto-created when a case of a matching type is opened — the analyst does not have to remember what to do next
- Each task must be checked off before the case can progress — no step can be silently skipped
- New investigators follow the same procedure as veterans from day one
- Examiners see a complete, consistent task completion record for every case

**Form fields (inline panel — "New Playbook" button):**
1. **Playbook Name** *(required)* — e.g. `Standard AML Investigation`
2. **Case Type** *(required)* — must exactly match the `case_type` value used when creating a case (e.g. `aml`). This is the trigger — when a case is created with this case type, this playbook's steps are auto-applied.
3. **Description** — brief note on when this playbook applies
4. **Steps** — each step has:
   - **Title** — the task name (e.g. `Review customer KYC and CDD documents`)
   - **Priority** — critical / high / medium / low
   - **Due Hours** — how many hours after case creation this step is due (optional)
   - **Dependency** — "After step N" — this step is BLOCKED until step N is marked complete. Used to enforce investigation sequence (e.g. step 3 can only start after step 2 is done)
   - Click **+ Add step** to add more rows
5. **Create Playbook** button — disabled until Name and Case Type are filled

**Key concept — Case Type matching:**
The Case Type field in the playbook must exactly match the case_type entered when creating a case. If the playbook has `case_type = aml` and a new case is created with type `aml`, all steps auto-create on that case instantly. If the types don't match, the playbook doesn't fire. This means you can have multiple playbooks for different investigation types — `aml`, `fraud`, `kyc`, `cyber` — each with their own step sequence.

**Key concept — Step dependencies:**
Steps can be marked "After step N" to enforce sequence. Example: step 3 (Counterparty Check) depends on step 2 (Transaction Analysis) — the analyst cannot start step 3 until step 2 is marked complete. This enforces the investigation procedure order, not just a checklist.

**What to show:**
- Click **New Playbook**
- Fill in the form with AML sample data below
- Add all 6 steps using **+ Add step**
- On step 3, set dependency to "After step 2" to show the dependency feature
- Click **Create Playbook**
- Show the new playbook card in the list — expand it to see all 6 steps

**Demo line:**
> "The Case Type is the trigger. When an analyst opens a case of type AML, every step in this playbook is automatically created on that case — in order, with priorities and due dates. Step 3 is blocked until step 2 is done. The investigation procedure is enforced by the system, not by memory or discipline."

**Sample data:**
- Playbook Name: `Standard AML Investigation`
- Case Type: `aml`
- Description: `Standard procedure for AML suspicious transaction investigations`
- Steps:

| # | Title | Priority | Due Hours | Dependency |
|---|-------|----------|-----------|------------|
| 1 | Review customer KYC and CDD documents | high | 4 | No dependency |
| 2 | Analyse transaction history for last 6 months | high | 8 | After step 1 |
| 3 | Check counterparty risk profile | high | 12 | After step 2 |
| 4 | Assess structuring or threshold patterns | medium | 16 | After step 3 |
| 5 | Document findings with supporting evidence | high | 20 | After step 4 |
| 6 | Make STR / no-STR decision with rationale | critical | 24 | After step 5 |

---

### Feature 5 — Assignment Rules
**URL:** `/settings/assignment-rules`

**Why we built this:**
When alerts arrive — especially during high-volume periods — cases sit unassigned. A manager has to manually read each alert, decide which analyst to route it to, and make the assignment. This creates a triage bottleneck. If the manager is unavailable, cases queue up. Senior analysts end up overloaded while junior analysts have capacity. There is no audit evidence of why a case was routed to whom.

**How it helps:**
- Every case is assigned automatically the moment it is created — zero manual triage for routine cases
- Rules match on case type + priority (AND logic) — as specific or as broad as needed
- Round-robin by role picks the analyst with fewest open cases — automatic load balancing
- Specific User assignment routes high-stakes cases to your best person every time
- Assignment decision is logged — examiners can see who was assigned, when, and why
- If no rule matches, the system falls back to the least-loaded manager → admin — nothing ever stays unassigned

**End-to-end flow:**
```
Case is created (manual or via alert automation)
         ↓
Assignment engine fires immediately (synchronous, same request)
         ↓
Rules evaluated in priority_order: Rule 1 → Rule 2 → Rule 3 ...
         ↓
First rule where ALL conditions match → wins
         ├── assign_to_user  → direct assignment to that person
         └── assign_to_role  → picks user with fewest open cases in that role
         ↓
If no rule matches → system fallback: least-loaded manager → admin → super_admin
```

**Realistic volume:** 8–15 rules for a 10–30 analyst NBFC team. Bounded by the number of distinct routing decisions the team needs — one rule per case type + priority overrides + catch-all.

**What to show:**
- Open the page — explain evaluation order (numbered list, lower number fires first)
- Open "+ Add Rule" modal — show Match Conditions (Priority + Case Type, both optional = AND logic), Assign To (Role round-robin vs Specific User dropdown)
- Create the 9 rules below one by one
- After creating, show the ordered list — demonstrate how Rule 1 intercepts critical sanctions before Rule 3 even gets evaluated

**Demo line:**
> "No case should sit unassigned. These rules run the moment a case is created — critical sanctions go directly to your sanctions specialist, PEP cases go to a manager for enhanced due diligence sign-off, and routine AML cases distribute themselves across the analyst team. Your manager never triages manually again."

---

**The 9 rules to create (in this order):**

**Rule 1 — Sanctions critical: direct to specialist**
```
Name:            Sanctions Critical → Divya Reddy
Priority Order:  10
Description:     Sanctions hits require FIU-IND notification within 24 hours — always to the specialist
Match Priority:  Critical
Match Case Type: sanctions
Assign To:       Specific User → Divya Reddy (analyst)
```
*Regulatory basis: PMLA Section 12A requires reporting a sanctions match to FIU-IND within 24 hours. Cannot afford round-robin.*

---

**Rule 2 — Any critical case not caught above → manager**
```
Name:            Critical (any type) → Manager
Priority Order:  20
Description:     Critical cases not matched by a specialist rule escalate to the manager immediately
Match Priority:  Critical
Match Case Type: (blank — matches any)
Assign To:       Role → Manager
```
*Why: A critical AML, fraud, or cyber case the specialist rules didn't catch needs senior ownership immediately, not an available-junior analyst.*

---

**Rule 3 — High AML → Sneha Patel (STR specialist)**
```
Name:            AML High → Sneha Patel
Priority Order:  30
Description:     High-value AML cases are potential STR candidates — Sneha handles STR filing end-to-end
Match Priority:  High
Match Case Type: aml
Assign To:       Specific User → Sneha Patel (analyst)
```
*Why: STR filing under PMLA requires continuity — same analyst owns the case from alert to FIU submission. Avoids handoff errors in the report.*

---

**Rule 4 — PEP cases → manager (Enhanced Due Diligence)**
```
Name:            PEP → Manager (EDD Required)
Priority Order:  40
Description:     PEP cases require Enhanced Due Diligence and senior management sign-off per RBI KYC norms
Match Priority:  (blank)
Match Case Type: pep
Assign To:       Role → Manager
```
*Regulatory basis: RBI Master Direction on KYC (2016, updated 2023) — PEP onboarding and review requires approval at senior management level.*

---

**Rule 5 — Standard AML → analyst pool**
```
Name:            AML (routine) → Analyst Pool
Priority Order:  50
Description:     Medium/Low AML cases not caught by higher rules — distributed across the analyst team
Match Priority:  (blank)
Match Case Type: aml
Assign To:       Role → Analyst
```
*Note: Rule 3 already took High AML. This catches Medium and Low AML that fell through.*

---

**Rule 6 — Fraud cases → analyst pool**
```
Name:            Fraud → Analyst Pool
Priority Order:  51
Description:     Fraud investigation cases distributed across the analyst team
Match Priority:  (blank)
Match Case Type: fraud
Assign To:       Role → Analyst
```

---

**Rule 7 — KYC cases → analyst pool**
```
Name:            KYC → Analyst Pool
Priority Order:  52
Description:     KYC review cases distributed across the analyst team
Match Priority:  (blank)
Match Case Type: kyc
Assign To:       Role → Analyst
```

---

**Rule 8 — Insider threat / Cyber → manager (sensitive, need-to-know)**
```
Name:            Insider / Cyber → Manager
Priority Order:  60
Description:     Internal fraud and cyber cases must not be routed to general analyst pool — information is sensitive
Match Priority:  (blank)
Match Case Type: insider
Assign To:       Role → Manager
```
*Repeat with case_type=cyber at priority_order 61.*
*Why: Insider cases risk evidence tampering and tipping-off if visible to the wrong analyst. Manager controls the information perimeter.*

---

**Rule 9 — Catch-all fallback**
```
Name:            Fallback → Analyst Pool
Priority Order:  100
Description:     Catches anything not matched by rules above — no case stays unassigned
Match Priority:  (blank — catch-all)
Match Case Type: (blank — catch-all)
Assign To:       Role → Analyst
```
*The amber "this rule matches every case" warning is expected and correct here. This is intentional.*

---

**Runtime routing table (what happens when a case is created):**

| Case Priority | Case Type | Rule fires | Assigned to |
|---|---|---|---|
| Critical | sanctions | Rule 1 | Divya Reddy directly |
| Critical | aml | Rule 2 | Least-loaded manager |
| Critical | fraud | Rule 2 | Least-loaded manager |
| High | aml | Rule 3 | Sneha Patel directly |
| Medium | aml | Rule 5 | Least-loaded analyst |
| Any | pep | Rule 4 | Least-loaded manager |
| Any | fraud | Rule 6 | Least-loaded analyst |
| Any | kyc | Rule 7 | Least-loaded analyst |
| Any | insider | Rule 8 | Least-loaded manager |
| Any | cyber | Rule 8 | Least-loaded manager |
| Any | anything else | Rule 9 | Least-loaded analyst |
| *(no rules at all)* | *(any)* | System fallback | Least-loaded manager → admin |

---

### Feature 6 — Alert Sources & Alert Rules
**URL:** `/settings/alert-sources` then `/settings/alert-rules`

**Why we built this:**
Every bank and NBFC has a Transaction Monitoring System (TMS) or core banking system that fires alerts. Today, those alerts arrive in email, Excel exports, or a separate queue — and a human manually reads each one to decide if it needs a case. This is slow, error-prone, and creates a gap between when the alert fires and when the investigation starts. That gap is where regulatory breaches happen.

**How it helps:**
- TMS alerts are ingested in real time via webhook — no manual monitoring of email or dashboards
- Rules evaluate each alert automatically: if amount > threshold, set priority and case type
- Case is created and assigned in under a second — SLA clock starts immediately
- No alert falls through the cracks — every inbound event is logged even if no rule matches
- Audit trail shows exactly when the alert arrived, what rule matched, and what case was created

**What to show:**
- Create an alert source: name it `Core Banking TMS`, type = webhook
- Copy the inbound URL
- Create an alert rule: if amount > 10,00,000 → priority High, case type AML, auto-assign

**Demo line:**
> "Your TMS fires an alert. That alert hits this URL, gets evaluated against your rules, and a case is created and assigned — automatically, in under a second. No manual triage. The SLA clock starts at the moment the alert arrives."

---

### Feature 7 — Inbound Webhook Setup
**URL:** `/settings/inbound`

**Why we built this:**
Alert sources need a documented, secure, authenticated endpoint to push events to. Without this, integration is ad-hoc — different teams configure different endpoints, payloads are inconsistent, and there is no centralised log of what came in and when. Integration also needs to survive TMS upgrades and schema changes.

**How it helps:**
- Single documented endpoint for all inbound alert traffic
- Sample payload shown in the UI — TMS team knows exactly what to send
- Every inbound event is logged with timestamp, source, and full payload — full traceability
- End-to-end flow is visible: alert received → rule evaluated → case created → assigned

**What to show:**
- Show the inbound endpoint URL
- Show sample payload format
- Explain the auto case creation flow end to end

**Demo line:**
> "This is the single URL your TMS team configures. The payload format is documented right here — send us a JSON event with transaction details and we handle the rest. Every event that arrives is logged regardless of whether a rule matched."

---

### Feature 8 — Outbound Webhooks
**URL:** `/settings/webhooks`

**Why we built this:**
Investigation outcomes need to flow back to downstream systems — the risk platform, core banking, the loan origination system, the regulatory reporting tool. Today, compliance teams copy-paste case closure details into emails or update systems manually. This is slow, creates data inconsistency, and means downstream systems are always stale. There is also no way to verify that the notification was delivered and not tampered with.

**How it helps:**
- Case events (created, closed, STR filed, escalated) are pushed automatically to any downstream system
- HMAC signing ensures the payload cannot be tampered with in transit — downstream systems can verify authenticity
- Test button lets you verify the integration before going live
- Multiple webhooks can be configured — one for the risk platform, one for core banking, one for reporting

**What to show:**
- Create a webhook: name = `Risk Platform Notify`, URL = any test URL
- Show events that can trigger it: case created, case closed, STR filed
- Show HMAC signing toggle
- Test the webhook

**Demo line:**
> "When a case closes or an STR is filed, your risk platform, core banking system, or any downstream tool is notified automatically. HMAC signing ensures the payload cannot be tampered with in transit. No more manual copy-paste between systems."

---

### Feature 9 — API Keys
**URL:** `/settings/api-keys`

**Why we built this:**
Some systems — loan origination platforms, onboarding systems, fraud engines — need to create cases or read case data programmatically, without a human logging in. Without API keys, every integration requires a shared user account, which cannot be audited per-system, cannot be revoked per-integration, and creates a security risk if one system is compromised.

**How it helps:**
- Each external system gets its own API key — revoke one key without affecting others
- Keys are shown only once at creation — they cannot be retrieved later, preventing exposure
- Every API call made with a key is logged with the key name — audit trail shows which system did what
- Scope is limited: external systems can create/read cases but cannot change roles or access admin functions

**What to show:**
- Generate an API key
- Name it `LOS Integration`
- Show it only displays once
- Explain it allows external systems to create/read cases programmatically

**Demo line:**
> "Your loan origination system needs to automatically open an investigation case when a suspicious application is flagged. This API key is how it authenticates. It shows only once — copy it now. Each integration gets its own key so you can revoke one without affecting the others."

---

### Feature 10 — Scheduled Reports
**URL:** `/settings/reports`

**Why we built this:**
Compliance heads and CCOs need regular visibility into investigation health — how many cases are open, how many SLAs were breached last week, how many STRs were filed. Today this requires someone to manually query the system, compile numbers, and email a summary. If that person is on leave, the report doesn't go out. There is no consistent cadence.

**How it helps:**
- Reports are generated and emailed automatically on a defined schedule — weekly, monthly, or custom
- PDF format means no system access is needed to read the report — suitable for board-level circulation
- Report goes to defined recipients regardless of who is in the office
- Creates a consistent compliance monitoring cadence with zero manual effort

**What to show:**
- Create a report: weekly, every Monday 9am, PDF, email to `ravi.kumar@opsowl.dev`
- Report type: Case Summary

**Demo line:**
> "Every Monday morning your compliance head gets a PDF in their inbox — open cases, SLA breaches, STR filings from last week. No one has to generate it manually. The cadence never slips because someone forgot or was on leave."

---

### Feature 11 — Data Retention
**URL:** `/settings/retention`

**Why we built this:**
PMLA (Prevention of Money Laundering Act) requires financial institutions to retain records of transactions and investigations for a minimum of 5 years. Without a managed retention policy, data may be deleted too early (regulatory breach) or stored indefinitely (storage and privacy risk). There is also no audit evidence that retention rules are being applied consistently.

**How it helps:**
- Retention policies are configured per document type and enforced automatically
- Data is not deleted before the regulatory minimum — no accidental early purge
- After the retention period, data can be purged systematically — no indefinite accumulation
- Retention configuration is logged — examiners can verify the policy was in place

**What to show:**
- Show retention policies per document type
- Explain PMLA requirement: 5 years minimum

**Demo line:**
> "PMLA mandates 5-year retention. This page is where you configure it — per document type, per data category. The system enforces it. No data is deleted before its time, and nothing accumulates indefinitely."

---

### Feature 12 — Workspace Settings
**URL:** `/settings`

**Why we built this:**
Every bank and NBFC uses different terminology. Some call it an "Investigation," others call it a "Case." Some use "TAT" instead of "SLA," "Investigation Officer" instead of "Analyst," "Inquiry" instead of "Note." A platform that forces its own terminology creates friction with existing processes, training materials, and regulatory reports. Staff spend time translating instead of investigating.

**How it helps:**
- Every label in the application is configurable — the platform speaks your language
- Changes reflect instantly across the entire application — no partial updates
- New staff see terminology consistent with internal SOPs from day one
- Regulatory reports generated by the platform use your terminology, not generic labels

**What to show:**
- Change `Case` → `Investigation`
- Change `Analyst` → `Investigation Officer`
- Change `SLA` → `TAT`
- Show it reflects across the entire application instantly

**Demo line:**
> "Every bank and NBFC has their own terminology. OpsOwl adapts to your language — not the other way around. Change it here, it reflects everywhere immediately — dashboard, reports, audit logs, everything."

---

## MODULE 2 — Alert Ingestion & Case Creation

### Feature 13 — Alert Queue
**URL:** `/alerts`

**Why we built this:**
When alerts arrive via webhook, the compliance team needs a centralised view of every inbound event — what arrived, from which source, when, what payload it carried, and which case was created from it. Without this, if an alert-to-case pipeline fails or a rule doesn't match, the event is invisible — no one knows the alert arrived and no investigation is started.

**How it helps:**
- Every inbound alert is visible in one queue regardless of source
- Full payload is stored — if a rule was misconfigured, the original alert data is available for reprocessing
- Linked case is shown next to each alert — one click to jump to the investigation
- Timestamp and source allow precise audit trail: "this alert arrived at 14:32 from Core Banking TMS and created case C-0042"

**What to show:**
- Show inbound alerts that arrived via webhook
- Show auto-created cases linked to each alert
- Show alert source, payload, timestamp

**Demo line:**
> "Every alert that arrived from your TMS is here — full payload, timestamp, source, and the case it created. If an alert came in but no case was created, you will see it here and can investigate why."

---

### Feature 14 — Create Case Manually
**URL:** `/cases/new`

**What to show:**
- Fill all fields with sample AML case data
- Assign to Divya Reddy
- Set priority High, SLA starts immediately
- Add tags

**Why we built this:**
Not every investigation originates from an automated alert. Referrals from relationship managers, internal tips, STR reviews, and manual review of transactions all require a case to be opened manually. Without a structured creation form, cases are opened inconsistently — wrong priority, missing tags, no assignment — which breaks downstream automation (SLA, playbook, reporting).

**How it helps:**
- Structured form ensures all required fields are filled before the case is created
- SLA clock starts the moment the case is created — no delay from manual setup
- Tags and case type trigger playbook and assignment rules automatically
- Case number is assigned immediately — traceable from creation

**Demo line:**
> "Not every case comes from an automated alert. A relationship manager flags something suspicious, a colleague spots a pattern — you open a case manually here. The same SLA, playbook, and assignment rules apply from the moment you hit create."

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

**Why we built this:**
Banks migrating from Excel, legacy case management systems, or another compliance tool need to bring existing cases into the platform without re-entering them one by one. Even in ongoing operations, periodic imports from external screening systems or watchlist hits require batch case creation. Without bulk import, migration is either manual (weeks of effort) or not done (history is lost).

**How it helps:**
- Thousands of cases can be imported in a single operation from a standard CSV
- Each imported case gets its SLA clock, assignment, playbook tasks, and audit trail from the moment it lands
- Template CSV is downloadable — the TMS or source system can generate it directly
- Import errors are flagged row by row — no silent failures

**What to show:**
- Download CSV template
- Upload a filled CSV with 3-5 cases
- Show all cases created in one operation

**Demo line:**
> "Migrating from Excel or another system? Bulk import brings in thousands of cases at once. Each one gets its SLA clock, assignment, and audit trail from the moment it lands. The template CSV is downloadable — your source system generates it directly."

---

### Feature 16 — Cases List
**URL:** `/cases`

**Why we built this:**
Analysts and managers need to work their queue efficiently. Without filtering, searching, and SLA visibility on the list view, the analyst has to open each case to understand its urgency. In a queue of 200 cases, finding breached or near-breach cases is impossible without tooling. Bulk operations — reassigning when an analyst is on leave, changing status in batch — need to be done in seconds, not one case at a time.

**How it helps:**
- SLA countdown on every row means urgency is visible without opening a case
- Filter by status, priority, assignee to immediately surface the right subset
- Search by case number or title to find any specific case in seconds
- Bulk select → bulk assign or bulk status change handles workload redistribution in one action
- Priority badges provide at-a-glance severity across the full list

**What to show:**
- Filter by status, priority, assigned to
- Search by case number or title
- Show SLA countdown on each row
- Show priority badges
- Bulk select → bulk assign or bulk status change

**Demo line:**
> "This is the analyst's queue. SLA countdown on every row — no case needs to be opened to know its urgency. Filter to your name, sort by SLA, work the most urgent first. If an analyst goes on leave, bulk-select their cases and reassign the entire queue in one action."

---

## MODULE 3 — Investigation Workspace

### Feature 17 — Case Overview
**URL:** `/cases/[id]`

**Why we built this:**
Investigation is not a single activity — it involves reviewing documents, recording notes, completing tasks, analysing transactions, and producing a narrative. Without a unified workspace, investigators bounce between email, shared drives, spreadsheets, and multiple systems. Context is lost between sessions. There is no single source of truth for what has been done on a case.

**How it helps:**
- Everything about a case is accessible from one screen — status, priority, SLA, assignee, all sub-sections
- Status transitions are one-click — move from Open → In Progress → Pending Review without leaving the page
- AI Copilot is available from the case header — analyst can ask questions over the entire case in plain English
- Export PDF produces a complete, formatted case report for sharing with examiners or senior management
- Sub-navigation (Notes, Tasks, Documents, Transactions, Findings, Narrative, Closure, Timeline) organises all investigation activity in one place

**What to show:**
- Case header: title, status, priority, SLA countdown
- Status transition buttons: move from Open → In Progress
- Assigned to, created by, due date
- AI Copilot button
- Export PDF button
- Sub-navigation: Notes, Tasks, Documents, Transactions, Findings, Narrative, Closure, Timeline

**Demo line:**
> "This is the investigation workspace. Everything about this case — status, SLA, assignee, all activity — is on one screen. The analyst never needs to leave this page to do their work."

---

### Feature 18 — Notes
**URL:** `/cases/[id]/notes`

**Why we built this:**
Investigation involves continuous judgment calls — "I reviewed the KYC, counterparty looks suspicious, escalating." These observations need to be recorded in real time, attributed to the right person, timestamped, and preserved as part of the case record. Email threads and chat messages are not part of the case record, are not auditable, and are lost when an analyst changes roles.

**How it helps:**
- Every observation is recorded inside the case — no external tools needed
- Notes are timestamped, attributed to the author, and immutable — they cannot be edited or deleted
- Multiple investigators can add notes independently — the case builds a collaborative record
- In regulatory examination, examiners can read every note in sequence to understand exactly how the investigation progressed

**What to show:**
- Add an internal note with rich text
- Add another note as a different user
- Show threaded notes with timestamps and author

**Demo line:**
> "Every observation, every judgment call, every communication — recorded inside the case, timestamped, attributed to the author. This is the investigator's running diary. An examiner reads it and understands exactly what was thought and when, without asking anyone."

**Sample note:**
> `Initial review complete. Customer has 9 NEFT transactions of ₹9.5L each over 3 days. Counterparty is a sole proprietorship registered 2 months ago. Escalating to detailed transaction analysis.`

---

### Feature 19 — Tasks
**URL:** `/cases/[id]/tasks`

**Why we built this:**
Investigation has multiple parallel workstreams — document review, transaction analysis, counterparty check, customer contact. Without task management inside the case, work is tracked in email, chat, or Excel. There is no visibility into what has been completed and what is pending. Managers cannot see investigation progress without asking the analyst directly. Steps are missed.

**How it helps:**
- Playbook tasks are auto-created when the case is opened — the analyst's checklist is ready from day one
- Each task has an assignee, due date, and status — managers see progress without asking
- Manual tasks can be added for case-specific requirements
- Completed tasks show who completed them and when — audit evidence of investigation thoroughness
- Open task count is visible on the case header — one number tells you how much work remains

**What to show:**
- Playbook tasks already created automatically
- Check off first task: KYC review done
- Add a new manual task: `Request 6-month bank statement from customer`
- Assign task to Arjun Mehta, due date tomorrow
- Show task status: todo → in progress → done

**Demo line:**
> "The playbook tasks were already here when the case was created — the analyst did not set them up. Each task has an owner and a due date. As tasks complete, the manager sees progress in real time. Every completed task is timestamped — examination evidence that the procedure was followed."

---

### Feature 20 — Documents
**URL:** `/cases/[id]/documents`

**Why we built this:**
Investigations are built on documents — KYC packets, bank statements, transaction records, correspondence. Today these live in email attachments, shared drives, and personal folders. Documents are not versioned, not searchable, and not tied to the specific case. When an examiner asks "show me the KYC document used in this investigation," the analyst has to hunt through email to find it.

**How it helps:**
- Every document uploaded to a case is stored against that case — permanently, with version history
- OCR processing makes every document full-text searchable — find any phrase across all case documents
- AI Copilot can read and answer questions over uploaded documents — "summarise the bank statement" in seconds
- Version history means every update to a document is tracked — examiners see what was available when
- ClamAV antivirus scans every upload — no malicious files enter the system

**What to show:**
- Upload a PDF (bank statement or KYC document)
- Show OCR processing
- Show document preview
- Show version history
- Explain AI will analyse this document

**Demo line:**
> "Every document uploaded to a case is OCR-processed and stored against that case permanently. It becomes searchable and AI-readable. Your analyst can ask the Copilot questions over this document in seconds. When an examiner asks for the KYC document used in this investigation, it is right here — not in someone's email."

---

### Feature 21 — Communications
**URL:** `/cases/[id]/comms`

**Why we built this:**
Investigations involve communication — emails from customers, letters from counterparties, notices from regulators. These communications contain entities, amounts, dates, and red flags that are relevant to the case. Without structured capture, communications sit in email inboxes — they are not part of the case record, not analysed for red flags, and not available to the examiner.

**How it helps:**
- Communication entries are stored inside the case — email, letter, call summary, all types
- AI automatically extracts entities (parties, amounts, dates) from the communication text
- Red flags are identified and surfaced automatically — the analyst does not have to read carefully for every indicator
- Full communication record is part of the case audit trail — examiners see all relevant correspondence

**What to show:**
- Add a communication entry: email from customer
- Show entity extraction: parties, amounts, dates pulled out automatically
- Show red flags identified

**Demo line:**
> "Customer sends an email explaining the transfers. It goes into the case here. The AI reads it and extracts the parties, amounts, and dates — and flags anything suspicious. The analyst gets a structured summary instead of reading a wall of text."

---

### Feature 22 — Timeline
**URL:** `/cases/[id]/timeline`

**Why we built this:**
In regulatory examination, the question is always "walk me through what happened on this case." Without an automated timeline, the analyst has to reconstruct the investigation from memory, notes, and email threads — a process that takes hours and is never complete. There is also no way to verify that actions were taken in the right order or that nothing was altered after the fact.

**How it helps:**
- Every action on the case is recorded automatically — notes, document uploads, status changes, task completions, AI queries
- Timeline is chronological, immutable, and real-time — new actions appear without page refresh
- No human has to maintain the audit trail — it builds itself
- Examiners can trace the entire investigation without asking the analyst a single question
- Filter by event type to find specific actions instantly

**What to show:**
- Every action in the case listed chronologically
- Note added, document uploaded, status changed, task completed
- Real-time — new actions appear without page refresh
- Filter by event type

**Demo line:**
> "This is the complete, automated audit trail of everything that happened on this case. Every action by every person, in order, timestamped, without any human maintaining it. An examiner can sit here and trace the entire investigation from open to close without asking anyone a single question."

---

## MODULE 4 — Transaction & Entity Intelligence

### Feature 23 — Transactions & Flow of Funds
**URL:** `/cases/[id]/transactions`

**Why we built this:**
AML investigations are fundamentally about transactions — amounts, channels, counterparties, timing. Without structured transaction data inside the case, analysts work in Excel, manually building flow-of-funds tables. Structuring patterns — multiple transactions just below reporting thresholds — are invisible until someone plots them manually. This takes hours. The visual insight that matters arrives too late.

**How it helps:**
- Transactions are structured inside the case — ref number, amount, channel, counterparty, date, narration
- Transaction timeline shows chronological flow — patterns emerge immediately
- Flow of Funds graph renders transactions as nodes and edges — visualises money movement at a glance
- Suspicious transactions can be flagged with a reason — feeding directly into findings and the STR
- Summary bar shows total flow, flagged count, and channels — the key numbers without scrolling

**What to show:**
- Add a transaction manually with full details
- Show transaction timeline in chronological order
- Toggle to Flow of Funds graph — nodes and edges
- Flag a transaction as suspicious with reason
- Show summary bar: total flow, flagged count, channels

**Demo line:**
> "Nine transactions, just under the reporting threshold, all going to the same new entity. The flow-of-funds graph makes the structuring pattern visible in 3 seconds. This would take an analyst 2 hours in Excel."

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

---

### Feature 24 — Subjects
**URL:** `/subjects`

**Why we built this:**
A subject — customer, counterparty, or entity — often appears across multiple cases. Without a centralised subject profile, investigators have no visibility into the subject's full history. "Has this person been investigated before? What accounts are they linked to? Who are their known associates?" — these questions require manual cross-referencing across multiple cases, which is slow and incomplete.

**How it helps:**
- Subject 360 profile aggregates all cases, accounts, and relationships for a single entity
- Connected entity graph shows the network — who is linked to whom, through what accounts
- Relationships between subjects can be added manually — building the investigative picture
- Cross-case history is visible — if this person has been investigated 3 times before, the analyst knows immediately
- Skip tracing and beneficial ownership analysis is built into the investigation workflow

**What to show:**
- Search for a subject
- Click into Subject 360 profile
- Show linked cases, linked accounts, entity relationships
- Show connected entity graph
- Add a relationship between two subjects

**Demo line:**
> "Who is this person really? Subject 360 pulls together every case they have appeared in, every linked account, every known relationship. This is what skip tracing looks like when it is built into the investigation platform — not a separate tool, not a manual cross-reference."

---

## MODULE 5 — AI & Document Intelligence

### Feature 25 — AI Insights & Risk Score
**URL:** `/cases/[id]/insights`

**Why we built this:**
Managers reviewing large case queues cannot read every case in detail to prioritise. They need a fast signal: which cases are high risk and need immediate attention, which are low risk and can wait. Without AI triage, managers spend hours reading to do what should take minutes. Risk scoring based on gut feel is inconsistent and indefensible in examination.

**How it helps:**
- AI reads the entire case — documents, transactions, notes, entities — and produces a structured risk score
- Risk factors are listed with explanations — not a black box, each factor is shown and justified
- Managers can triage 50 cases in 5 minutes instead of 5 hours
- Risk score is stored and auditable — examiners can see what the AI assessed and why
- Score updates when new information is added to the case — live risk picture, not a one-time snapshot

**What to show:**
- Click **Run AI Analysis**
- Show risk score generated: High / Medium / Low
- Show risk factors identified
- Show AI summary of the case

**Demo line:**
> "The AI reads the entire case — documents, transactions, notes, entities — and produces a risk score with reasons. Your manager can triage 50 cases in 5 minutes instead of 5 hours. Every AI assessment is stored and auditable — the examiner can see exactly what the AI found and why."

---

### Feature 26 — AI Copilot
**URL:** `/cases/[id]` — click AI Copilot button

**Why we built this:**
Investigators spend significant time searching through documents and transaction records to answer specific questions — "what transactions are above ₹5 lakhs?", "what does the counterparty's KYC say?", "are there any red flags in this bank statement?" Without AI, each question requires manual review. Complex cases with hundreds of pages of documents are simply not read thoroughly under time pressure.

**How it helps:**
- Analyst can ask any question over the entire case in plain English — documents, transactions, notes, all included
- Answers are grounded in the case data — no hallucination, every response cites what it found
- PII is masked before any AI call — sensitive customer data is protected even in the query
- Reduces time-per-case for document-heavy investigations from hours to minutes
- All AI queries and responses are logged — examiners can review what was asked and what the AI answered

**What to show:**
- Ask: `What is the relationship between the customer and Rahul Enterprises?`
- Ask: `What transactions are above ₹5 lakhs?`
- Ask: `Summarise the key red flags in this case`
- Show PII masking note

**Demo line:**
> "Your analyst can interrogate the entire case in plain English. The AI answers from the case documents and data — no hallucination, grounded in what is actually in the case. PII is masked before any call. The data never leaves your server. Every query and response is logged for examination."

---

### Feature 27 — AI Logs
**URL:** `/cases/[id]/ai-logs`

**Why we built this:**
AI-assisted decisions in compliance are only defensible if the AI's reasoning is auditable. "The AI said it was high risk" is not an acceptable answer in a regulatory examination without showing exactly what the AI was given, what it produced, and when. Without AI logs, every AI-assisted decision is a black box — and black boxes are a regulatory liability.

**How it helps:**
- Every AI call is logged: model used, prompt sent, response received, latency, timestamp, user who triggered it
- Examiners can inspect the exact AI reasoning for any decision in any case
- Compliance team can defend AI-assisted decisions with a complete paper trail
- Latency tracking enables performance monitoring — slow AI calls are visible and actionable
- Provides evidence that AI was used appropriately and within policy

**What to show:**
- Every AI call: model used, prompt tokens, response, latency
- Timestamp and user who triggered it

**Demo line:**
> "Every AI decision is logged. Model, prompt, response, latency — all auditable. When an examiner asks 'how did you assess this case as high risk?', you open AI Logs and show them exactly what the AI read and what it concluded. No black box."

---

## MODULE 6 — Findings & Evidence

### Feature 28 — Findings & Evidence
**URL:** `/cases/[id]/findings`

**Why we built this:**
The conclusion of an investigation — "this is suspicious, here is why, here is the evidence" — needs to be formally documented. Without a structured findings module, conclusions are buried in notes, emails, or narrative text. High-severity findings without attached evidence are conclusions without proof — indefensible in examination. Without an approval workflow, findings are not reviewed before they drive STR decisions.

**How it helps:**
- Findings are structured — title, description, severity, evidence attachment
- High-severity findings cannot be submitted without attached evidence — the system enforces documentation discipline
- Approval workflow ensures a manager reviews every finding before it is finalised
- Findings feed directly into the AI narrative and STR — the chain from evidence to report is traceable
- Examiner can see every finding, its severity, the evidence attached, who submitted, and who approved

**What to show:**
- Create a finding: title, description, severity High
- Attach evidence document to it
- Show severity gate: High requires evidence before submitting
- Submit finding
- Approve finding as manager

**Demo line:**
> "Findings are the structured conclusions of your investigation. High-severity findings cannot be submitted without attached evidence — the system enforces documentation discipline. A manager approves every finding before it is finalised. The chain from evidence to conclusion to STR is fully traceable."

**Sample finding:**
- Title: `Structuring Pattern Detected`
- Description: `9 NEFT transactions of ₹9.5L each over 3 days, just below ₹10L CTR threshold. Consistent structuring pattern.`
- Severity: `High`
- Evidence: upload bank statement PDF

---

## MODULE 7 — Regulatory Reporting & Closure

### Feature 29 — STR / goAML
**URL:** `/cases/[id]/str`

**Why we built this:**
Suspicious Transaction Reports must be filed with FIU-IND in goAML XML format. Without automation, analysts fill a separate template, manually format the XML, and submit it — a process that takes 1-2 hours per STR and introduces data entry errors. The XML format is complex and version-sensitive. Errors in the submission mean re-filing, delays, and regulatory scrutiny.

**How it helps:**
- Case data (subject, transactions, reporting entity) is pre-filled into the STR form — no re-entry
- goAML-compatible XML is generated automatically — no manual XML formatting
- Download and submit directly to FIU-IND — no intermediary tool
- STR filing is recorded on the case timeline — audit evidence of when the report was filed
- Reduces per-STR effort from 1-2 hours to 15 minutes

**What to show:**
- Fill STR details: reporting entity, subject details, transaction details
- Generate goAML-compatible XML
- Download XML ready for FIU-IND submission

**Demo line:**
> "When you decide to file an STR, the platform generates a goAML-compatible XML file ready for direct submission to FIU-IND. No manual formatting, no XML editing, no separate template. The case data is already here — fill the reporting-specific fields and download."

---

### Feature 30 — AI Narrative
**URL:** `/cases/[id]/narrative`

**Why we built this:**
Every closed investigation needs a written narrative — a structured summary of what was found, what was analysed, and what decision was made. Writing this from scratch takes an analyst 2-3 hours per case. Under high caseload, narrative quality degrades, important details are missed, and the document looks different every time. Without a consistent narrative structure, examiners get inconsistent documentation across cases.

**How it helps:**
- AI drafts the complete investigation narrative from all case data — transactions, findings, entities, timeline, documents
- Draft is produced in seconds — the analyst reviews and edits rather than writing from scratch
- Manager approval workflow ensures every narrative is reviewed before the case is closed
- Consistent structure across all cases — every narrative follows the same format
- Time per narrative reduced from 2-3 hours to 15 minutes

**What to show:**
- Click **Generate Narrative**
- Show AI-drafted investigation summary
- Edit the body
- Manager approves narrative
- Show approve/reject workflow

**Demo line:**
> "The AI drafts the complete investigation narrative from everything in the case — transactions, findings, entities, timeline. Your analyst edits if needed, the manager approves. What used to take 3 hours takes 15 minutes. Every narrative follows the same structure — consistent documentation across every case."

---

### Feature 31 — Case Closure
**URL:** `/cases/[id]/closure`

**Why we built this:**
A case cannot be considered complete if it has no documented reason for closure. "Closed — no action" without a disposition and rationale is one of the most common examination findings. Without enforcement, analysts close cases to clear their queue without proper documentation. Re-opening a closed case also needs to be possible — and audited — when new information emerges.

**How it helps:**
- Case cannot be closed without selecting a disposition and recording a rationale — enforced by the system
- Regulatory report reference can be attached at closure — links the case to the STR or CTR filing
- Closure is recorded on the timeline — examiner sees when the case was closed and by whom
- Re-open option is available — with the re-open reason recorded — so subsequent investigation is traceable
- An examiner will never find a closed case without a documented reason

**What to show:**
- Cannot close without disposition and rationale
- Fill: disposition = STR Filed, rationale, regulatory report reference
- Close the case
- Show re-open option

**Demo line:**
> "A case cannot be closed without a disposition and rationale. The system enforces completeness — there is no way around it. An examiner will never find a closed case without a documented reason. If new information emerges, re-open it — the re-open reason is recorded in the audit trail."

---

## MODULE 8 — Reporting & Monitoring

### Feature 32 — Dashboard
**URL:** `/dashboard`

**Why we built this:**
A compliance manager starting their day needs to immediately answer three questions: What is breaching SLA right now? What needs my attention? What happened yesterday? Without a dashboard, the answer requires opening the cases list, filtering, sorting — taking 15 minutes before the manager can start their day. Critical cases breach while the manager is still getting oriented.

**How it helps:**
- Open cases, SLA breached, and pending review counts are visible at a glance — no filtering required
- Attention Required panel surfaces the cases that need the manager's action right now
- Recent activity feed shows what happened across the team since last login — full situational awareness in seconds
- Counts refresh automatically — breaches appear without page reload

**What to show:**
- Open cases, SLA breached, pending review counts
- Attention Required panel
- Recent activity feed

**Demo line:**
> "The manager opens this page at 9am and immediately knows: how many cases are breached, what needs their attention, and what the team worked on yesterday. Situational awareness in under 10 seconds."

---

### Feature 33 — Workload
**URL:** `/workload`

**Why we built this:**
Without visibility into individual analyst workloads, managers distribute cases by intuition. Some analysts are overloaded while others have capacity. TAT per analyst varies significantly but no one knows why. SLA breach rates are not tracked by analyst — when an examiner asks "why did these cases breach?" the answer is not available.

**How it helps:**
- Cases per analyst view makes overload and underutilisation immediately visible
- Average TAT per analyst enables performance management with data, not guesswork
- SLA breach rate by analyst identifies who needs support — before their queue breaches further
- Case ageing distribution shows how long cases have been open — no case ages invisibly

**What to show:**
- Cases per analyst
- Average TAT per analyst
- SLA breach rate
- Case ageing distribution

**Demo line:**
> "Your compliance manager can see at a glance who is overloaded, who has capacity, and which cases are about to breach SLA — before it happens. Performance management with data, not guesswork."

---

### Feature 34 — Analytics
**URL:** `/analytics`

**Why we built this:**
Compliance heads need trend data to present to boards and regulators — are case volumes increasing? Is the STR filing rate appropriate relative to case volume? Are high-risk cases resolved faster than low-risk? Without analytics, these questions require manual data extraction, Excel analysis, and hours of preparation before every board meeting.

**How it helps:**
- Case volume trends show whether investigation activity is growing — early signal of systemic risk
- STR filing rate shows the proportion of cases resulting in regulatory reports — key compliance KPI
- Risk level distribution shows the risk profile of the investigation portfolio
- CSV export feeds directly into Power BI or Tableau for board-level reporting

**What to show:**
- Case volume trends
- STR filing rate
- Risk level distribution
- Export to CSV for Power BI / Tableau

**Demo line:**
> "Every metric your board presentation needs is here — case volumes, STR rates, risk distribution. Export to CSV for your Power BI dashboard. No manual extraction, no Excel pivot tables."

---

### Feature 35 — Global Tasks
**URL:** `/tasks`

**Why we built this:**
Analysts work across multiple cases simultaneously. Without a cross-case task view, an analyst with 12 open cases has to visit each case individually to see their pending tasks. Overdue tasks — critical investigation steps — are invisible until someone explicitly checks each case. Managers cannot see task completion rates across the team without opening every case.

**How it helps:**
- All tasks across all cases in one view — the analyst sees their complete workload at a glance
- Filter by assigned to, due date, status — find overdue tasks instantly
- Managers see incomplete tasks across the team without opening individual cases
- Tasks due today are surfaced immediately — no task is missed because it was buried in a case

**What to show:**
- All tasks across all cases in one view
- Filter by assigned to, due date, status

**Demo line:**
> "Every task across every case, in one view. The analyst sees their complete workload without opening each case individually. Filter to today's due dates — these are the tasks that need action right now."

---

### Feature 36 — Global Documents
**URL:** `/documents`

**Why we built this:**
Investigations often involve the same document type appearing in multiple cases — the same counterparty's KYC, the same entity's bank statement. Without a global document view, finding all documents related to a specific entity requires opening each case individually. During examination, when an examiner asks for "all KYC documents for this customer," the answer requires a manual search across cases.

**How it helps:**
- Every document across every case is searchable from one place
- Search by filename or content (full-text, OCR-powered) finds documents across the entire platform
- Documents are linked to their source case — one click to open the case context
- Examiners can search for all documents related to a specific entity or account without case-by-case navigation

**What to show:**
- All documents across all cases
- Search by filename or content

**Demo line:**
> "Every document uploaded to any case is searchable here. Search by filename or by content — OCR means the full text of every PDF is indexed. Find every document mentioning a specific account number or entity name across all investigations."

---

### Feature 37 — Search
**URL:** `/search`

**Why we built this:**
Compliance operations generate large volumes of data — hundreds of cases, thousands of notes, thousands of documents. Finding a specific piece of information — a customer name, an account number, a transaction reference — requires knowing which case it is in. Without global search, investigators waste time browsing. Examiners asking "show me everything related to this entity" cannot be answered quickly.

**How it helps:**
- Single search bar covers cases, notes, documents, and subjects simultaneously
- Results show case context — the case number and title are displayed with each result
- Cross-entity search enables relationship discovery — all mentions of an entity across the platform
- Response time is seconds, not minutes of manual browsing

**What to show:**
- Search across cases, documents, notes, subjects
- Show results with case context

**Demo line:**
> "One search box. Cases, notes, documents, subjects — everything. Type an account number, a customer name, a transaction reference — every mention across the platform surfaces instantly with case context."

---

## MODULE 9 — Audit & Compliance

### Feature 38 — Audit Logs
**URL:** `/audit`

**Why we built this:**
In regulatory examination, the question is not just "what happened?" but "can you prove what happened and that no one tampered with the record?" Without a tamper-evident audit trail, any log can be altered after the fact. An examiner cannot trust a log that could have been modified. SHA-256 hash-chaining is the technical standard for tamper-evidence — each entry contains the hash of the previous one, making any alteration mathematically detectable.

**How it helps:**
- Every action by every user across the entire platform is logged — login, case created, document uploaded, status changed, user deactivated
- SHA-256 hash chain links every entry to the previous one — any tampering breaks the chain and is immediately detectable
- Filter by user, action type, and date range — find any action in seconds
- Export audit log for examination submission — provide the complete record without granting system access
- No action can be hidden — even failed login attempts are recorded

**What to show:**
- Every system-wide action: login, case created, document uploaded, user deactivated
- SHA-256 hash chain — each entry linked to previous
- Filter by user, action type, date range
- Export audit log

**Demo line:**
> "This is the tamper-evident audit trail for the entire platform. Every action by every user, SHA-256 hash-chained. If an examiner asks 'who accessed this case on this date' — it is here. If anyone tries to delete or alter an entry, the hash chain breaks and it is immediately detectable. This is examination-grade evidence."

---

### Feature 39 — Inbound Events Log
**URL:** `/alerts`

**Why we built this:**
When an alert-to-case automation fails — a rule doesn't match, a webhook payload is malformed, or a case isn't created — there needs to be a record that the alert arrived. Without this, alert failures are invisible. An examiner asking "did your system receive an alert for this transaction?" cannot be answered without a log.

**How it helps:**
- Every inbound event is stored regardless of whether a rule matched or a case was created
- Payload, source, timestamp, and outcome (case created / no match / error) are all recorded
- Failed ingestion events surface immediately — compliance team can identify and fix gaps in real time
- Provides the evidence chain: "alert arrived at T+0, case created at T+0.8 seconds, assigned at T+1 second"

**What to show:**
- Every alert that arrived via inbound webhook
- Payload, source, timestamp, case created from it

**Demo line:**
> "Every alert that arrived is here — even the ones that didn't create a case. If a rule didn't match or a payload was malformed, it is still logged. The examiner can verify that the system received every alert that was sent."

---

## MODULE 10 — Deployment

### Feature 40 — On-Premise & Air-Gapped AI

**Why we built this:**
Banks and NBFCs cannot send investigation data to a cloud platform. Customer PII, transaction records, STR details — all of this is regulated data that cannot leave the institution's network under RBI data localisation requirements. SaaS compliance tools require sending data to external servers. On-premise deployment with local AI inference is the only architecture that satisfies data sovereignty requirements for Indian regulated entities.

**How it helps:**
- Single Linux server in your data centre — no cloud dependency
- One-command installation: `bash install.sh`
- Ollama runs llama3.1:8b locally — all AI inference on your hardware, not external APIs
- Not a single byte of investigation data leaves your network — full data sovereignty
- Air-gapped operation is supported — works without internet access after initial setup
- You control upgrades, backups, and retention — no vendor lock-in on data

**Demo line:**
> "Everything you have seen today runs on a single Linux server in your data centre. One command to install. No cloud dependency. The AI inference runs locally — Ollama runs llama3.1:8b on your own hardware. Not a single byte of investigation data leaves your network. You have full control. This is the only investigation platform built from the ground up for on-premise deployment at Indian regulated entities."

---

## Closing Line
> "From the moment an alert fires in your TMS to a formally closed, goAML-filed, audit-defensible case — everything happens in one platform, on your infrastructure, with a complete hash-chained trail that survives any regulatory examination."

> "What questions do you have?"
