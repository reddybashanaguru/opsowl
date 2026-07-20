# OpsOwl AI — Complete Application Guide

> Version: July 2026 · Complete reference for all 41 screens across Administration, Workspace, and Case Detail tabs.

This guide explains every screen in OpsOwl in plain English — what it does, who uses it, how it works, and why it is designed the way it is. It is written for investigators, compliance officers, and developers who need to understand the application deeply without reading source code.

---

## Table of Contents

**Part 1 — Administration Section**
1. [User Management](#1-user-management)
2. [Audit Logs](#2-audit-logs)
3. [SLA Policy Engine](#3-sla-policy-engine)
4. [Inbound Events](#4-inbound-events)
5. [Outbound Webhooks](#5-outbound-webhooks)
6. [Scheduled Reports](#6-scheduled-reports)
7. [Case Templates](#7-case-templates)
8. [Investigation Playbooks](#8-investigation-playbooks)
9. [FATF Typology Library](#9-fatf-typology-library)
10. [API Keys](#10-api-keys)
11. [Alert Sources](#11-alert-sources)
12. [Alert Routing Rules](#12-alert-routing-rules)
13. [Auto-Assignment Rules](#13-auto-assignment-rules)
14. [Data Retention](#14-data-retention)
15. [AI Governance](#15-ai-governance)
16. [Settings](#16-settings)

**Part 2 — Workspace Pages**
17. [Dashboard](#17-dashboard)
18. [Cases List](#18-cases-list)
19. [My Tasks](#19-my-tasks)
20. [Documents (Global)](#20-documents-global)
21. [Alert Queue](#21-alert-queue)
22. [Subject Directory](#22-subject-directory)
23. [Subject Detail](#23-subject-detail)
24. [Search](#24-search)
25. [Team Workload](#25-team-workload)
26. [MLRO Dashboard](#26-mlro-dashboard)

**Part 3 — Case Detail Tabs**
- [Case Layout — Persistent AI Copilot](#case-layout--persistent-ai-copilot)
27. [Case Overview](#tab-1-case-overview)
28. [Timeline](#tab-2-timeline)
29. [Tasks](#tab-3-tasks)
30. [Documents](#tab-4-documents)
31. [Findings](#tab-5-findings)
32. [Insights](#tab-6-insights)
33. [AI Logs](#tab-7-ai-logs)
34. [STR Filings](#tab-8-str-filings)
35. [Communications](#tab-9-communications)
36. [Narrative](#tab-10-narrative)
37. [Transactions](#tab-11-transactions)
38. [Case Closure](#tab-12-case-closure)
39. [UBO (Ultimate Beneficial Ownership)](#tab-13-ubo-ultimate-beneficial-ownership)
40. [Notes](#tab-14-notes)
- [Cross-Tab Reference Summary](#cross-tab-reference-summary)

---

# Part 1 — Administration Section

The Administration section is only accessible to users with the **Admin** or **Super Admin** role. It contains all the configuration that governs how the workspace behaves — who can access it, how alerts are processed, how cases are assigned, and what regulatory framework is active. Think of it as the control room for the entire platform.

---

## 1. User Management

**Route:** `/users`

### What this page does

This is where administrators invite new team members into the workspace, assign them roles, and deactivate accounts when someone leaves the organisation. Every person who uses OpsOwl must have an account created here first.

### Roles and what they mean

OpsOwl uses five distinct roles arranged in a hierarchy from lowest to highest access:

- **Viewer** — Can read cases, documents, findings, and the timeline, but cannot create or edit anything. Suitable for auditors or regulators who need read-only access.
- **Analyst** — The standard investigator role. Can create cases, add tasks, upload documents, write notes, log communications, create findings, and use the AI Copilot. Most day-to-day investigation work is done at this level.
- **Manager** — Everything an analyst can do, plus the ability to approve or reject findings, narratives, and STR filings that were submitted by others. Managers also have exclusive access to the STR filing tab in cases (analysts are excluded to prevent tipping-off). They can also approve case closures.
- **Admin** — Full management access including all administration pages. Can manage users, configure the system, and reopen closed cases.
- **Super Admin** — The highest level. All admin capabilities plus the ability to manage other admins.

### Inviting a new user

To add someone to the workspace, an admin clicks the **Invite User** button and fills in the person's full name, email address, initial role, and a temporary password (minimum 8 characters). The system creates the account immediately — there is no email verification step. The new user can log in straight away using the credentials provided.

### Changing a role

Roles can be changed at any time. Hovering over any user's role badge reveals a pencil icon. Clicking it opens an inline dropdown where the admin can select a new role and save. The change takes effect immediately on the user's next page load.

### Deactivating a user

When someone leaves the team, their account is deactivated rather than deleted. This preserves all the audit history — their past actions, approvals, and case contributions remain visible in the system. Deactivation is a two-step inline confirmation to prevent accidental clicks. Administrators cannot deactivate their own account.

### Finding users

The page loads all users at once (up to 200). A search box at the top filters by name or email in real time. Clicking any of the role pills at the top of the page filters the list to show only users with that role, which is useful for seeing how many analysts versus managers are active.

---

## 2. Audit Logs

**Route:** `/audit`

### What this page does

The Audit Logs page is an immutable record of everything that has ever happened in the system — every login, every case created, every document uploaded, every approval given, every configuration change. It exists to satisfy regulatory requirements that financial crime investigation systems maintain a complete, tamper-proof audit trail.

### Who uses it

Primarily used by admins during internal reviews, compliance audits, and incident investigations. If a regulator asks "who approved this STR filing and when?", the answer is found here.

### What each log entry contains

Every log entry records: the type of action that occurred (for example, `str_submitted` or `case_status_changed`), the name and email of the person who performed the action, their IP address at the time, the specific record that was affected (for example, which case or which document), and any additional context (such as what a field was changed from and to). Each entry is also classified by severity — informational, warning, or error — which helps triage unexpected events.

### Browsing logs

Logs are shown newest-first and load in batches of 50 as you scroll down. There is no fixed pagination; the list simply extends as you reach the bottom. You can filter by severity using the buttons at the top (All / Info / Warning / Error), and you can search for specific event types by typing in the event code filter box — for example, typing `str` will show only STR-related events.

### Exporting

The **Export CSV** button downloads a CSV of the full audit log. This is typically used when preparing documentation for a regulator or an internal compliance review.

### Important note on IP addresses

In development or Docker-based deployments, the IP address shown may be an internal Docker network address rather than the investigator's real IP. In production, this reflects the actual client IP.

---

## 3. SLA Policy Engine

**Route:** `/sla`

### What this page does

SLA stands for Service Level Agreement. This page defines time limits for how long a case can remain open based on its priority. If an investigation takes longer than the defined thresholds, the system raises visual warnings across the dashboard and case list to alert the team. This page is where those thresholds are configured.

### Why this matters

Regulatory frameworks like PMLA (India), the EU's AMLD6, and CBUAE (UAE) all have expectations around how promptly suspicious activity is escalated and reported. If a critical-priority case sits uninvestigated for three days, the compliance team needs to be alerted before it becomes a regulatory problem.

### How the thresholds work

Each of the four priority levels — Critical, High, Medium, and Low — has two thresholds:

- **Warning threshold:** The number of hours after which the case turns amber on the dashboard, signalling that time is running short.
- **Breach threshold:** The number of hours after which the case turns red, indicating the SLA has been violated.

There is also a **warn percentage** setting (between 10% and 95%). This defines at what percentage of the breach timeline the warning is triggered. For example, if the breach threshold is 48 hours and the warn percentage is 75%, the warning appears after 36 hours.

### Visual representation

Each priority row shows a visual progress bar. The amber warning line appears at the warn percentage mark; the red breach line appears at the 100% (breach) mark. This makes it easy to understand the relationship between warning and breach thresholds at a glance.

### Who can edit

All users can view SLA policies. Only admins and super admins can edit them. Non-admins will see the values but the edit buttons will not appear.

---

## 4. Inbound Events

**Route:** `/settings/alert-sources`

### What this page does

OpsOwl does not generate alerts itself — alerts come from your organisation's existing AML monitoring tools (such as NICE Actimize, SAS AML, Oracle FCCM, or Featurespace ARIC). This page is where you register those external systems so they are authorised to send alerts into OpsOwl.

### How it works

Each registered source gets a unique **ingest URL** — a web address that the external system posts alerts to. When an alert arrives at that URL, OpsOwl validates that it came from the correct source, processes it, and adds it to the Alert Queue.

To prevent unauthorised systems from injecting fake alerts, each source uses **HMAC-SHA256 signing**. When you create a source, OpsOwl generates a secret key. The external system must include a cryptographic signature in each request header (`X-OpsOwl-Signature`) that proves the alert was sent by the legitimate source and has not been tampered with.

### Creating an alert source

When you click **Add Alert Source**, you provide:
- A name for the source (e.g. "Actimize Production")
- The type of external system (Actimize, SAS AML, Oracle FCCM, Featurespace, or Generic/Custom)
- Whether it sends via webhook or CSV batch
- A description

After creation, OpsOwl shows you the **webhook secret once** in an amber banner. You must copy this and configure it in your external AML system immediately — it is never shown again. The ingest URL remains visible permanently on the source card.

### Enabling and disabling sources

Each source can be toggled active or inactive without deleting it. This is useful if an external system is undergoing maintenance — you can pause ingestion without losing the configuration.

---

## 5. Outbound Webhooks

**Route:** `/settings/webhooks`

### What this page does

While Inbound Events bring data into OpsOwl, Outbound Webhooks push data out. When something important happens inside OpsOwl — a case is escalated, an STR is filed, an SLA is breached — the system can automatically notify an external tool such as a SIEM, a Slack channel, a Microsoft Teams webhook, or a custom internal API.

### How it works

You configure a webhook by providing the destination URL and choosing which events should trigger it. OpsOwl signs each outgoing request with HMAC-SHA256 so the receiving system can verify the payload came from OpsOwl. Each webhook delivery is logged — you can see whether it succeeded or failed, and retry failed deliveries manually.

### Use cases

Common use cases include: notifying a SIEM when a case is escalated, sending a Slack message when an STR is submitted for approval, or triggering a downstream workflow when a case is closed.

---

## 6. Scheduled Reports

**Route:** `/settings/reports`

### What this page does

This page lets administrators configure automated reports that are generated and emailed to specified recipients on a fixed schedule — daily, weekly, or monthly. Reports summarise case activity, SLA compliance, team workload, and STR filing statistics over the reporting period.

### Configuration

Each scheduled report has a name, a report type (case summary, SLA report, filing report, etc.), a recipient email list, and a schedule (daily at a specified time, weekly on a specified day, or monthly on a specified date). Reports are generated in PDF or CSV format depending on the report type.

---

## 7. Case Templates

**Route:** `/settings/templates`

### What this page does

When investigators create a new case, they can optionally select a template. A template pre-fills the case with a standard description, priority level, tags, and a pre-built task checklist. This ensures that similar types of investigations always follow the same starting structure — nothing important gets missed because someone forgot to create a standard task.

### Who manages templates

Managers, admins, and super admins can create and edit templates. Analysts only use them when creating a case — they cannot modify them.

### What a template contains

Each template has:
- A **name** (shown in the case creation dropdown)
- A **description skeleton** — a pre-written description text that gives the investigator a starting point and ensures consistent documentation format
- A **default priority** — the priority level the case is set to when the template is applied
- **Tags** — labels that are automatically attached to the case (for example: "high-value-customer" or "sanctions-related")
- A **task checklist** — a list of tasks that are automatically created inside the case, each with its own title, description, and priority

### Active vs. inactive

Templates can be toggled active or inactive. Inactive templates are hidden from the case creation picker but their configuration is preserved. This is useful if a template is being revised or is temporarily not applicable.

---

## 8. Investigation Playbooks

**Route:** `/settings/playbooks`

### What this page does

Playbooks are more powerful than templates. While a template is a static starting configuration, a playbook is a structured, ordered workflow. When a case of a specific type is created, the playbook automatically generates all its investigation steps as ordered tasks inside the case — complete with due times and dependency rules between steps.

### How playbooks differ from templates

A template gives you a pre-filled description and a task list. A playbook gives you a proper workflow where steps can depend on each other — Step 3 cannot be started until Step 2 is complete. This enforces investigation methodology.

### How it works

Each playbook is linked to a **case type** (for example, "Sanctions Hit" or "Structuring"). When a case of that type is created — either manually or automatically from an alert — the playbook fires and creates all its steps as tasks in the case.

Each step in a playbook has:
- A title and description
- A priority level (critical, high, medium, or low)
- An optional **due hours** setting — how many hours after case creation the step must be completed. For example, if a step has a 24-hour due time, the task's due date is automatically set to 24 hours from when the case was created.
- An optional **dependency** — which earlier step must be completed before this step becomes actionable. This prevents investigators from skipping ahead in the workflow.

### Example

A "PEP Match" playbook might have these steps:
1. Confirm the identity match (no dependency, due within 4 hours)
2. Pull the PEP's public profile and political exposure history (depends on step 1, due within 8 hours)
3. Review all linked accounts and transactions (depends on step 2, due within 24 hours)
4. Escalate or clear — document decision (depends on step 3, due within 48 hours)

This ensures every PEP investigation follows the same rigorous process.

---

## 9. FATF Typology Library

**Route:** `/settings/typologies`

### What this page does

FATF stands for the Financial Action Task Force — the global standard-setting body for anti-money laundering. Over decades, FATF has identified recurring patterns of how criminals move money. These patterns are called typologies. This page is OpsOwl's built-in library of those typologies.

### Who uses it

This page is read-only and accessible to all users. Investigators use it to understand what different money laundering patterns look like. The AI engine uses it to suggest which typologies apply to a given case based on the risk factors it detects.

### What each typology contains

Each typology entry includes:
- A **code** (for example, "STR-001") and a descriptive **name** (for example, "Smurfing / Structuring")
- A **category** — there are six categories: Placement, Layering, Integration, Structuring, Terrorist Financing (TF), and Proliferation Financing (PF). Each category is colour-coded throughout the application.
- A **description** of the pattern in plain English
- A list of **red flags** — observable behaviours that indicate this typology may be present (for example: "Multiple cash deposits just below the reporting threshold made at different branches on the same day")
- A list of **risk keywords** — specific words and phrases the AI engine looks for when scanning case data to suggest this typology automatically
- A **FATF reference** — the specific FATF guidance document that describes this pattern

### How it connects to cases

When investigators work on a case, they go to the Insights tab and can tag which FATF typologies are relevant. The AI analyses the case's risk factors and suggests which typologies match — these are shown with a violet "AI Suggested" badge. Investigators can accept, reject, or add their own typology tags manually.

---

## 10. API Keys

**Route:** `/settings/api-keys`

### What this page does

API Keys allow external systems and scripts to access OpsOwl's data programmatically without using a username and password. For example, a data analytics team might use an API key to pull case statistics into a Power BI dashboard, or a compliance team might use one to automate exports.

### How API keys work

Each key is created with a **name** (so you know what it's for), a set of **permissions** (called scopes), and an optional **expiry date**. The key is a long random string that the external system includes in every API request using the header `X-API-Key: your_key_here`.

The four available permissions are:
- **cases:read** — view cases and their details
- **cases:write** — create and update cases
- **documents:read** — download and view documents
- **audit:read** — read audit log entries

### Security

The full API key value is shown **only once** immediately after creation, in an amber banner at the top of the page. After you close that banner, the key can never be retrieved again — only its first 8 characters are stored for identification. If you lose the key, you must revoke it and create a new one.

Keys can expire automatically after 30, 90, 180, or 365 days, or they can be set to never expire. Expiring keys are safer because they reduce the window of exposure if a key is leaked.

---

## 11. Alert Sources

This page is described in detail under **Section 4 — Inbound Events** above (they are the same page, `/settings/alert-sources`).

---

## 12. Alert Routing Rules

**Route:** `/settings/alert-rules`

### What this page does

When an alert arrives in OpsOwl from an external AML system, the Alert Routing Rules decide what happens to it. Should it go into the alert queue for a human analyst to review manually? Or should it automatically create a case and start an investigation immediately?

This page is where those routing decisions are configured. It is one of the most powerful configuration pages in the system because it determines how quickly threats are escalated.

### How rules work

Rules are evaluated in order, from lowest to highest sort order number. The **first rule that matches** an incoming alert wins. If no rule matches, the alert goes into the queue with a priority based on its risk score.

Each rule can match on:
- **Alert type** — for example, only apply this rule to "Sanctions Hit" alerts, or use the wildcard `*` to match every alert
- **Minimum risk score** — only apply this rule if the alert's risk score is at or above a threshold (0-100)

When a rule matches, it can either:
- **Queue** the alert — it lands in the Alert Queue for manual triage
- **Auto** — a case is created immediately and optionally assigned to a specific user, without any human needing to press a button

### Deduplication

Each rule has a **deduplication window** (between 0 and 8760 hours). If the same entity triggers the same alert type within this window, the duplicate alert is silently discarded rather than creating a second case. This prevents the same suspicious behaviour from flooding the system with redundant cases.

### Example configuration

A well-configured ruleset might look like:
1. Sort order 1 — Alert type: `SANCTIONS_HIT`, any risk score → **Auto** (immediately create a case; sanctions hits must be investigated immediately)
2. Sort order 2 — Alert type: `*`, minimum risk score 80 → **Auto** (any very high-risk alert creates a case automatically)
3. Sort order 3 — Alert type: `*`, minimum risk score 0 → **Queue** (everything else goes into the queue for manual review)

---

## 13. Auto-Assignment Rules

**Route:** `/settings/assignment-rules`

### What this page does

When a new case is created — either manually or automatically from an alert — this page's rules decide who it gets assigned to. Without auto-assignment, every new case starts as "Unassigned" and someone has to manually distribute the workload. Auto-assignment rules remove that manual step.

### How rules work

Like alert routing rules, assignment rules are evaluated in order by priority number. The first matching rule determines the assignment. Rules can match on:
- **Case priority** — for example, only assign critical-priority cases using this rule
- **Case type** — for example, only assign "PEP Match" cases using this rule

Both conditions work together with AND logic — a case must match all the specified conditions for the rule to apply. If you leave both conditions blank, the rule becomes a catch-all that matches every case.

### Assignment targets

Each rule assigns to either:
- **A specific user** — the case goes directly to that person
- **A role (round-robin)** — the case goes to whichever analyst or manager currently has the fewest open cases. This distributes workload evenly across the team.

### Example

You might configure:
- Rule 1: Case type = Sanctions Hit → Assign to the Senior Sanctions Analyst (specific user, because this requires expertise)
- Rule 2: Case priority = Critical → Assign to role Manager (round-robin, so the senior team shares the load)
- Rule 3: (no conditions) → Assign to role Analyst (round-robin, default for everything else)

---

## 14. Data Retention

**Route:** `/settings/retention`

### What this page does

This page controls how long different types of data are kept in the system before they are permanently deleted. Automatic data deletion is required by data protection laws — India's DPDP Act 2023, the EU's GDPR, and similar frameworks all require that personal data is not kept longer than necessary.

### The five data categories

OpsOwl manages retention separately for five different types of data:

- **Cases** — Closed and archived investigation records. The system retains these for the configured number of days (typically 5-7 years for financial crime records) and then permanently deletes them.
- **Documents** — Evidence files and attachments uploaded to cases.
- **Audit Logs** — The record of user actions. Notably, audit logs of the purge process itself are retained indefinitely even if other audit logs are purged.
- **Notifications** — In-app notification records shown to users.
- **Sessions** — Expired user login session tokens.

### How the purge works

A background process runs automatically every 24 hours. It identifies any records that have been in the system for longer than the configured retention period and permanently deletes them. You can see when each category was last purged in the "Last Purged" column.

### Enabling and pausing purges

Each retention category can be individually enabled or paused. Pausing stops automatic deletion for that category without changing the configured retention period. This is useful during a legal hold or audit freeze when you need to preserve data beyond the normal retention window.

### Important

You cannot change the retention period through the UI — only the enable/pause toggle is available. The actual retention duration (in days) is set by the system administrator at deployment time through environment configuration.

---

## 15. AI Governance

**Route:** `/settings/ai-act`

### What this page does

Every time the AI engine generates something in OpsOwl — a risk score, a narrative, an insight, a copilot response — that output must be reviewed by a qualified human before being used in a regulatory filing or case decision. This page provides a governance dashboard showing whether that oversight is actually happening, which outputs are still waiting for human review, and an exportable audit trail for regulators.

The page is called "Responsible AI Governance" and its content adapts based on the regulatory framework of your deployment jurisdiction.

### What regulators require

**India:** The DPDP Act 2023, FIU-IND AML & CFT Guidelines 2023, and RBI IT Governance Directions 2023 together require that the Principal Officer appointed under PMLA Section 12 personally oversees and authorises AI-assisted compliance outputs. India does not have a standalone AI Act — the governance framework is built from these three existing regulations.

**EU:** The EU AI Act (Regulation 2024/1689, in force from August 2025) classifies financial crime investigation as a high-risk AI use case. Articles 13 (transparency) and 14 (human oversight) require that human reviewers can override, correct, or reject any AI decision before it takes effect.

**UAE:** The CBUAE AML/CFT Supervision Standards require Compliance Officers to review and authorise all AI-assisted outputs before any goAML STR submission.

### The Conformity Checklist

The top section of the page shows six controls that together constitute a compliant AI governance posture. Each control shows a green checkmark (active) or amber warning (attention needed):

1. **AI decisions logged with model version** — Every AI action in OpsOwl records which model and version produced it. This is always active.
2. **Human oversight workflow active** — This shows green when at least one AI output has been reviewed by a human. If no AI outputs have ever been reviewed, this turns amber to prompt action.
3. **Data minimisation — prompts use case IDs** — Subject personal information (names, ID numbers) is never sent to the AI inference layer. Only case identifiers and non-PII context are used. Always active.
4. **Audit trail exportable for regulators** — The export function on this page makes the AI activity log available for regulator inspection. Always active.
5. **AI runs fully on-premise — zero data egress** — When using the local Ollama AI engine, all inference happens inside the organisation's own servers. Subject data never leaves the network. Always active for on-premise deployments.
6. **AI feature access is role-gated** — Only analysts and above can trigger AI; only managers and above can approve AI outputs. This enforces segregation of duties. Always active.

A score badge shows how many of the six controls are currently active (for example, "6/6 controls active"). If the oversight control is amber, the score drops to "5/6 controls active" and turns amber overall.

### Pending Human Review queue

The middle section lists every AI output across all cases that has not yet been reviewed by a human. Each row shows the case, what type of AI action it was, which model generated it, and when. Clicking "Review" takes you directly to that case's AI Logs tab where you can approve, reject, or edit the output.

### Export Audit Trail

The bottom section allows you to download a CSV of all AI activity within a date range. The exported file includes: the case ID and number, the type of AI action, which model and provider was used, the human review status, who reviewed it and when, the confidence score, and the timestamp. This CSV is what you would provide to a regulator during an inspection.

The downloaded file is named differently by jurisdiction — `ai-act-audit.csv` for EU deployments (where the EU AI Act applies) and `ai-governance-audit.csv` for India and UAE (which are governed by different frameworks).

---

## 16. Settings

**Route:** `/settings`

### What this page does

This is the main configuration page for the workspace. It is split into sections — some accessible to everyone, others only to admins. It controls everything from individual security settings to the regulatory framework the workspace operates under.

### For all users: MFA (Two-Factor Authentication)

Every user can enable two-factor authentication on their own account. When enabled, logging in requires a 6-digit time-based code from an authenticator app (Google Authenticator, Authy, etc.) in addition to the password. The setup flow shows a QR code to scan, then asks you to enter a code to confirm it worked. A set of one-time backup codes is shown at this point — save them somewhere safe, as each can only be used once if you lose access to your authenticator app.

### For all users: Change Password

A straightforward password change form. Enter your current password, then your new password (minimum 8 characters) with confirmation. Your session remains active after the change.

### For admins: AI Configuration

Shows which AI provider and model is currently active. If the workspace is running in air-gapped mode (using local Ollama), a green "Air-Gapped (Local AI)" badge appears. If using a cloud provider, an amber "Cloud AI" badge appears with a note that data protection agreements are required. This section also shows whether data leaves the network — "No" for air-gapped, "Yes" for cloud AI.

### For admins: Deployment Jurisdiction

A read-only panel showing which regulatory jurisdiction this deployment is configured for. This controls which regulatory references appear throughout the system (FIU-IND for India, Hellenic FIU for Greece, CBUAE for UAE, etc.). The jurisdiction is set at deployment time through environment variables and cannot be changed through the UI.

### For admins: Workspace Terminology

Different organisations call investigation records by different names. An NBFC might call them "cases", a bank might call them "investigations", a fintech might call them "reviews". This section lets you customise 14 labels used throughout the application — for cases, tasks, documents, notes, and risk-related terms — so the application matches your organisation's internal language.

### For admins: Regulatory Filings Configuration

This section configures everything related to STR and CTR filings. The configurable values include:
- What your jurisdiction calls the report (STR, SAR, CTR — the label shown in the UI)
- The name of the regulator to whom you file (FIU-IND, Hellenic FIU, CBUAE, etc.)
- The legal citation that authorises the filing requirement
- How many days you have after detecting suspicious activity to file a report (the default is 30 days)
- The local currency and the threshold amount at which a transaction must be reported as a cash transaction report (CTR threshold)
- Which ID document types are accepted in your jurisdiction (Passport, National ID, Emirates ID, Aadhaar, etc.)

---

# Part 2 — Workspace Pages

Workspace pages are the day-to-day investigative interface used by the entire team. Unlike the Administration section (which is configuration), workspace pages are where active investigation work happens.

---

## 17. Dashboard

**Route:** `/dashboard`

### What this page does

The dashboard is the first page you see after logging in. It gives you a real-time overview of the entire investigation workload — how many cases are active, how many are breaching their SLA, what the risk distribution looks like, and how the team is performing. Think of it as the control tower.

### What it shows

The dashboard is divided into several sections. At the top are summary numbers: total open cases, cases currently breaching their SLA (shown in red), cases at SLA warning (amber), and total cases closed this month. Below that are visual charts showing the breakdown of cases by priority, by status, and by risk level.

The middle section shows a live list of the highest-priority cases sorted by SLA urgency — cases that are closest to or already past their deadline appear first. Each case shows the case number, title, assigned investigator, current status, and SLA status badge.

The bottom of the page shows team workload — a breakdown of how many open cases each investigator currently has. This helps managers identify if someone is overloaded or if cases need to be redistributed.

---

## 18. Cases List

**Route:** `/cases`

### What this page does

This page shows all investigation cases in the system. It is the primary way investigators find and access their cases, and managers use it to monitor the full portfolio.

### Filtering and searching

The list can be filtered by status (draft, open, in progress, pending review, closed, escalated), priority (critical, high, medium, low), case type, date range, and assigned investigator. A text search box filters by case number or title. All filters can be combined.

Cases are displayed in a table showing: case number, title, subject name (if linked), status, priority, assigned investigator, SLA status badge, risk level, and created date. Clicking any row opens the full case detail view.

### Bulk operations

Cases can be selected with checkboxes for bulk operations such as bulk reassignment, bulk status updates, or bulk export. This is useful for managers reorganising workload.

### Creating a new case

The "New Case" button opens a creation form asking for: title, description, case type, priority, due date (optional), template selection (optional), and subject to link (optional). If a template is selected, the description field is pre-filled and tasks are automatically created. Cases start in "Draft" status and must be manually moved to "Open" to begin active investigation.

---

## 19. My Tasks

**Route:** `/tasks`

### What this page does

This page shows all the tasks assigned to the currently logged-in user across every case they are working on. It is a personal to-do list that aggregates task work from potentially dozens of different cases into a single view.

### Why this exists

Investigators often work on multiple cases simultaneously. Without a global task view, they would have to click into each case individually to see what tasks are pending. The My Tasks page solves this by pulling together every task assigned to you regardless of which case it belongs to.

### How it works

Tasks are shown in two tabs: **Active** (open and in-progress tasks) and **Done** (completed tasks). Active tasks can be filtered by priority and by whether they are overdue (past their due date). Each task card shows the task title, which case it belongs to, the due date, and priority badge.

Completing a task gives you a 5-second undo window. A toast notification appears at the bottom of the screen saying the task was completed, with an "Undo" button. This prevents accidental completions from causing irreversible state changes.

---

## 20. Documents (Global)

**Route:** `/documents`

### What this page does

This page shows all documents uploaded across every case in the system. It is primarily used by admins who need to monitor document processing, search for specific files across all investigations, or review the storage footprint of the workspace.

### What it shows

Each document entry shows the file name, file type, which case it belongs to, who uploaded it, when it was uploaded, its processing status (pending, processing, completed, failed), and the number of entities the AI extracted from it.

Documents are searchable by file name and filterable by processing status. Clicking a document navigates to the case it belongs to.

---

## 21. Alert Queue

**Route:** `/alerts`

### What this page does

The Alert Queue is the inbox for all incoming alerts from your external AML monitoring systems. Alerts arrive here after being processed by the Alert Routing Rules — specifically, any alert that did not match an "auto" routing rule lands here for manual triage by an analyst.

### What an alert contains

Each alert represents a suspicious pattern detected by your AML system. It includes: the alert type (for example, "Structuring" or "Velocity Breach"), the entity or account involved, the risk score assigned by the originating system, which alert source it came from, and when it was generated.

### Triaging an alert

Analysts review each alert and decide what to do with it:
- **Convert to Case** — the alert becomes a new investigation case. The alert's details pre-fill the case creation form.
- **Dismiss** — the alert is closed without creating a case, with a required reason documented for the audit trail.
- **Bulk Triage** — multiple alerts can be selected and dismissed or converted together.

### Alert statuses

Alerts move through these states: new (just arrived), under_review (an analyst has opened it), converted (a case has been created), dismissed.

---

## 22. Subject Directory

**Route:** `/subjects`

### What this page does

The Subject Directory is OpsOwl's database of all entities that have appeared in investigations — individuals, companies, and other legal entities. It is the central place to look up a subject's investigation history and current risk profile.

### What a subject record contains

Each subject has a canonical (standardised) name, entity type (individual or company), a risk score, and whether they are flagged as a Politically Exposed Person (PEP) or a potential sanctions match. The record also links to every case the subject has been involved in.

### Searching

The directory is searchable by name, entity type, and risk level. This allows investigators to quickly check whether a new subject of interest has been investigated before or has any existing risk flags.

---

## 23. Subject Detail

**Route:** `/subjects/{id}`

### What this page does

This page shows the complete profile of a single subject — everything OpsOwl knows about them. It is the "360-degree view" of an individual or company and is essential for understanding their investigation history.

### What it contains

The subject detail page shows:
- **Identity information:** All name variants, date of birth (for individuals), ID documents, addresses, phone numbers, and email addresses on record
- **Risk profile:** Current risk score, risk level, PEP status, sanctions flag status, and the history of how the risk score has changed over time
- **Linked cases:** Every investigation case this subject is associated with, with a summary of each case's status and outcome
- **Beneficial ownership** (if the entity is a company): Details of individuals with significant ownership or control

---

## 24. Search

**Route:** `/search`

### What this page does

Global search across the entire OpsOwl workspace. You can search for cases, subjects, documents, and transactions from a single search box. Results are grouped by type and ranked by relevance.

This is the fastest way to find something when you know part of a name, case number, reference number, or document title but do not know where exactly in the system to look.

---

## 25. Team Workload

**Route:** `/workload`

### What this page does

The Team Workload page gives managers a view of how investigation work is distributed across the team. It shows each team member's name, current role, number of open cases assigned to them, number of overdue tasks, and an SLA compliance score.

### Why this matters

Uneven workload distribution is a risk factor for compliance failures. If one analyst has 20 cases and another has 2, the overloaded analyst is more likely to miss SLA deadlines or make errors. This page exists so managers can spot imbalances and reassign cases before problems occur.

### What you can do

Managers can click on any team member to see a detailed breakdown of their cases — which are urgent, which are overdue, and which are approaching SLA breach. Cases can be reassigned directly from this view.

---

## 26. MLRO Dashboard

**Route:** `/mlro`

### What this page does

MLRO stands for Money Laundering Reporting Officer — the senior compliance official responsible for the organisation's AML programme and for signing off on all regulatory filings. This dashboard is designed specifically for that role, providing high-level metrics and a filing status view that other dashboards do not offer.

### What it shows

The MLRO dashboard focuses on:
- **Filing statistics:** How many STRs and SARs have been drafted, submitted, approved, and filed this month and year-to-date
- **Pending approvals:** STR filings waiting for the MLRO's approval
- **Overdue SLA cases:** Cases that have breached their investigation time limit
- **Risk trends:** How the risk distribution of the case portfolio has shifted over time
- **AI oversight metrics:** The percentage of AI-generated outputs that have been reviewed by a human (relevant for regulatory reporting)

---

# Part 3 — Case Detail Tabs

Every investigation in OpsOwl lives inside a **Case**. When you open a case, you see a sub-navigation bar across the top with tabs for different aspects of the investigation. Each tab focuses on a specific type of work or information. This part of the guide explains each tab in detail.

---

## Case Layout — Persistent AI Copilot

Before explaining individual tabs, it is important to understand one element that is present on every tab: the **AI Copilot button**.

A floating button labelled "AI Copilot" with a sparkle icon appears in the bottom-right corner of the screen on every case tab. Clicking it opens a side panel where you can have a conversation with the AI about the current case. The AI has access to the case's data — transactions, findings, subject profile, and insights — and can answer questions, suggest next steps, or help draft text.

The important thing is that the AI Copilot panel stays open as you switch between tabs. If you ask the AI a question while on the Transactions tab and then switch to the Findings tab, the conversation is still there. This is by design — it lets you use the AI as a running assistant throughout your investigation without losing context.

---

## Tab 1: Case Overview

**Route:** `/cases/{id}` (the default landing page when you open a case)

### What this page does

The Case Overview is the front page of an investigation. It shows you everything important about the case at a glance: the current status, who is assigned, the risk score, when it was created, linked subject, open tasks, recent documents, and a live activity feed. It is also where you manage the case's status and core details.

### The three-column layout

The page is split roughly 60/40 between a main content area on the left and a details panel on the right.

**Left side — the investigation workspace:**
- The case title, which can be edited by clicking the pencil icon next to it
- The case description, also inline-editable
- A quick note input — a one-line field for adding short notes directly to the timeline without opening the Notes tab
- A live activity feed showing the most recent events on this case in real time

**Right side — the details card:**
- Status badge (clickable to change the status)
- Priority, case type, assigned investigator (clickable to reassign)
- SLA status indicator (green OK / amber warning / red breached)
- Due date and created date
- Tags attached to this case
- If a subject is linked: their name and risk level, with a link to their profile
- A mini preview of the first 3 open tasks and the 2 most recent documents

### Status transitions — moving a case through its lifecycle

A case moves through these statuses as the investigation progresses:

1. **Draft** — The case has been created but work has not formally begun. Can move to Open.
2. **Open** — The investigation is acknowledged and active. Can move to In Progress or Escalated.
3. **In Progress** — Active investigation underway. Can move to Pending Review or Escalated.
4. **Pending Review** — The investigator has completed their work and submitted the case for a manager to review before closing. Can move to Closed, back to In Progress, or to Escalated.
5. **Escalated** — The case has been escalated for urgent senior attention. Can move back to In Progress.
6. **Closed** — The investigation is complete and the closure has been approved by a manager.

Moving a case to "In Progress" or "Pending Review" happens immediately when you click the status badge — no confirmation is needed because these are routine forward-moving transitions. Moving to any other status opens a dialogue box where you must write a brief reason. This reason is recorded in the timeline for the audit trail.

### The Investigation Nudge Panel

This is an AI-powered proactive assistant that analyses the case and surfaces important warnings at the top of the Overview page. It checks for things an investigator might overlook, such as:

- The SLA deadline has already been breached (critical — shown in red)
- No investigator has been assigned yet (warning)
- The risk score is high (70 or above) but no STR has been filed (warning)
- There are AI-generated outputs waiting for human review (informational — clicking this opens the AI Copilot)
- There are tasks with past-due dates (warning)
- No AI analysis has been run on this case yet (informational)
- No documents have been uploaded (informational)

Each nudge can be dismissed and it will not reappear until you refresh the page. The nudge panel remembers which ones you have dismissed for each case in your browser's session storage.

### Inline editing

Almost every field on the Case Overview can be edited without navigating to a separate page. Click the pencil icon next to any field (title, description, priority, due date, tags) to enter edit mode. When you make changes, a floating "Save changes" bar appears at the very bottom of the page — click Save to commit all changes at once, or Cancel to discard them.

### Exporting a case report

The "Export PDF" button in the top-right of the Overview generates a comprehensive PDF report of the entire investigation — case details, findings, timeline, closure decision — and downloads it automatically as `{case_number}-report.pdf`. This is used when a physical or archived record is needed.

### Assignee and watchers

Clicking the assignee name (or "Unassigned") opens a dropdown listing all team members with their roles. Selecting someone reassigns the case and adds an entry to the timeline. Separately, investigators can "watch" a case without being the primary assignee — watchers receive notifications about significant events on the case.

---

## Tab 2: Timeline

**Route:** `/cases/{id}/timeline`

### What this page does

The Timeline is the complete chronological record of everything that has happened on this case. Every action — whether taken by a human, the system, or the AI — creates a timeline event. This tab is the authoritative audit trail for the investigation.

### What gets recorded

The timeline automatically captures all of these events without anyone manually logging them:
- Case creation, status changes, and reassignments
- Every note added (with the note content)
- Every task created, updated, or completed
- Every document uploaded and when processing finishes
- Every AI action — when the copilot was queried, when insights were generated, when a narrative was created
- Every STR event — draft created, submitted, approved, rejected, filed
- Communications logged
- SLA warnings and breaches

### Live updates

The Timeline uses a live connection to the server (called Server-Sent Events or SSE). New events appear at the top of the list in real time without you needing to refresh the page. A small coloured dot in the tab header shows the connection status: green means live and connected, amber means reconnecting, red means the connection has been lost.

When you are scrolled down the list reading older events and new ones arrive at the top, a floating banner appears saying "N new events — scroll to top". Clicking it scrolls back to the top so you see the new events.

### Filtering

The timeline can be very long on active cases. Two filter controls help narrow it down:

- **Source filter:** Choose whether to see events from All sources, only System events (automated status changes, SLA alerts), only User events (things investigators did manually), or only AI events (copilot queries, analysis runs, narrative generation).
- **Event type filter:** A multi-select dropdown with all 22 specific event types. For example, you could filter to only `str_submitted` and `str_filed` events to see the exact history of a STR filing.

---

## Tab 3: Tasks

**Route:** `/cases/{id}/tasks`

### What this page does

The Tasks tab is the case's internal to-do list. Every action item, investigation step, and checklist requirement for this specific case lives here. Tasks keep the investigation organised and ensure nothing falls through the cracks.

### Active and Done views

The tab has two views. **Active** shows all open tasks — things still to do. **Done** shows completed tasks — the historical record of what has been accomplished. Active tasks can be filtered by priority (critical, high, medium, low) and by whether they are overdue (past their due date). This overdue filter is particularly useful when checking which tasks need immediate attention.

### Creating a task

Clicking "Add Task" opens a form where you fill in:
- **Title** (required) — what needs to be done. This is validated when you move out of the field — if you leave it empty, the field turns red.
- **Description** (optional) — more detail about what the task involves
- **Priority** — defaults to Medium
- **Due date** — must be today or in the future
- **Assigned to** — optionally assign it to a specific team member from a dropdown

### Completing a task with undo

When you check a task as complete, it disappears from the Active list and a toast notification appears at the bottom: "Task completed". This toast has an **Undo** button that stays visible for exactly 5 seconds. If you realise you completed the wrong task, clicking Undo brings it back instantly. After 5 seconds, the completion is permanent.

### Assigning and unassigning

Each task shows an "Assign to me" link if it is unassigned or assigned to someone else. Clicking it assigns the task to yourself immediately — the UI updates right away without waiting for the server to respond (this is called an optimistic update). If the server update fails for any reason, the UI reverts. The same applies to "Unassign".

### Playbook tasks

If the case was created from an Investigation Playbook, the initial batch of tasks was generated automatically by the playbook. These tasks are visually distinguished from manually created tasks with a chain-link icon. Tasks with dependencies (where Task B cannot start until Task A is complete) show a label indicating which task must be finished first.

---

## Tab 4: Documents

**Route:** `/cases/{id}/documents`

### What this page does

The Documents tab is where all evidence files for this investigation are stored — bank statements, identity documents, transaction screenshots, company registration records, court orders, and any other supporting material. Documents uploaded here are automatically analysed by the AI to extract relevant entities such as names, amounts, and account numbers.

### Uploading documents

You can upload documents by dragging and dropping files onto the upload zone, or by clicking the zone to open a file browser. Multiple files can be uploaded in a single batch.

### What happens after upload

After a file is uploaded, it goes through automated processing. The status progresses through four stages:
- **Pending** — The file has been received but processing has not started yet
- **Processing** — The AI is actively reading the document, extracting text, and identifying entities (people, companies, accounts, amounts)
- **Completed** — Processing is done. The extracted entities and text are now available to the AI Copilot and other case tools
- **Failed** — Something went wrong during processing (for example, the file was password-protected or in an unsupported format)

The page automatically checks for status updates every 5 seconds while any document is still processing. This means you do not need to manually refresh — the status badges update by themselves. Once all documents have settled to either Completed or Failed, the automatic checking stops.

### Retrying and deleting

Failed documents have a Retry button that attempts processing again — useful if the failure was due to a temporary system issue rather than a problem with the file itself. Any document can be deleted permanently, which also removes it from any evidence links it was attached to in Findings.

### The status filter

A dropdown at the top lets you filter documents by status (Pending / Processing / Completed / Failed). This is useful on cases with many documents when you want to focus on the ones that need attention.

---

## Tab 5: Findings

**Route:** `/cases/{id}/findings`

### What this page does

Findings are the core of the investigation record. A finding is a formally documented observation of suspicious or relevant activity — it is more structured than a note and more significant than a timeline event. Findings are the building blocks of the final investigation narrative and STR filing. They must go through an approval process before they can be used in formal documents.

### The summary bar

At the top of the Findings tab, four numbers are always visible: the total number of findings, how many have been approved, how many are critical severity, and how many are high severity. This gives investigators and managers a quick assessment of the investigation's findings picture without scrolling through the list.

### Categories of findings

Each finding belongs to one of twelve categories that describe what type of suspicious activity was observed:
- **Structuring** — Breaking up large transactions to stay below reporting thresholds
- **Layering** — Moving money through multiple accounts or entities to obscure its origin
- **Integration** — Reinvesting illicit funds into legitimate businesses or assets
- **Smurfing** — Using multiple people (smurfs) to make deposits at different branches
- **Round Tripping** — Sending money abroad and receiving it back to make it appear legitimate
- **PEP Exposure** — Connections to politically exposed persons
- **Sanctions Evasion** — Activity suggesting circumvention of sanctions
- **Unusual Cash** — Cash activity inconsistent with the customer's profile
- **Rapid Movement** — Money moving through accounts unusually quickly
- **Identity Fraud** — Evidence of false or stolen identity use
- **Account Takeover** — Signs that someone else has gained control of an account
- **Other** — Anything that does not fit the above

### Severity levels

Each finding is classified as Critical, High, Medium, or Low. Severity affects where the finding appears in the list (critical findings appear first) and controls which findings require evidence before they can be submitted for review.

### How narrative weight works

Each finding has a narrative weight — Primary, Supporting, or Contextual. When the AI generates an investigation narrative, it uses this weight to determine how prominent each finding is in the story. Primary findings form the core of the narrative; supporting findings add corroboration; contextual findings provide background that helps explain the primary concerns.

### The approval workflow

A finding goes through the following steps before it becomes part of the official investigation record:

**Step 1 — Draft:** The investigator creates the finding, filling in the title, description, category, severity, and narrative weight. At this point, it is a work in progress and has no formal status.

**Step 2 — Attach evidence (required for High and Critical):** For serious findings, at least one piece of evidence must be linked before the finding can be submitted for review. Evidence can be a document from the case, a note, an AI log entry, a subject record, or an alert. Each evidence link includes a brief citation label and a relevance explanation.

**Step 3 — Submit for review:** The investigator clicks "Submit for Review". The finding moves to the "Under Review" status and is now visible to managers for approval.

**Step 4 — Manager approval:** A manager (who is a different person from whoever submitted — this is the 4-eye rule) reviews the finding and either approves or rejects it.
- If **approved**, the finding becomes part of the official record and can be used in narratives and STR filings.
- If **rejected**, the investigator sees the rejection reason and can edit the finding and resubmit, or formally contest the rejection.

### The 4-eye rule explained

The investigator who submits a finding cannot approve their own finding. This is not a technical limitation added for convenience — it is a fundamental internal control requirement in financial compliance. Having two different people involved in creating and approving a finding prevents errors and reduces the risk of a single investigator misrepresenting what they found. If you open a finding you submitted and try to approve it, the system shows "Awaiting a different approver" instead of action buttons.

### AI-generated findings

The AI Copilot can automatically identify potential findings based on the case data. These appear in the list with a violet "AI Generated" badge and a confidence percentage showing how certain the AI is (for example, "87% confidence"). AI-generated findings still require a human investigator to review and formally submit them — the AI can only suggest, not decide.

---

## Tab 6: Insights

**Route:** `/cases/{id}/insights`

### What this page does

The Insights tab is the AI analysis hub for this case. It presents the AI engine's assessment of the case — a risk score, the factors that drove that score, detected anomalies, recommended next steps, and which FATF money laundering typologies are relevant. It also shows all the entities (people, companies, accounts) the AI has identified.

### Running an analysis

When you first open Insights on a new case, there is no analysis yet. Clicking "Run AI Analysis" sends all the case data — subject profile, transactions, documents, findings — to the AI engine. Analysis happens asynchronously (in the background), so the page polls for results every 3 seconds. When the new results arrive, the page updates automatically. If the analysis takes longer than 3 minutes, the page shows a timeout message and a Retry button.

Once analysis has been run, you can re-run it at any time as the investigation grows — as you add more transactions or documents, the risk score and recommendations may change.

### The risk gauge

The most visually prominent element is the circular SVG risk gauge showing the risk score (0 to 100). The colour of the gauge tells you immediately how serious the case is:
- **Green** — score below 40, low risk
- **Amber** — score between 40 and 74, medium to high risk
- **Red** — score 75 and above, very high or critical risk

The risk level label (LOW, MEDIUM, HIGH, or CRITICAL) appears below the number.

### Risk factors

Next to the gauge is a list of the specific factors that drove the risk score. These are plain-English sentences generated by the AI — for example: "Multiple transactions structured just below the ₹10 lakh reporting threshold", "Subject is a Politically Exposed Person with links to high-risk jurisdictions", "Transaction velocity inconsistent with declared business activity". These factors are also used as the basis for the AI's typology suggestions.

### Anomalies and recommendations

The Anomalies section lists behaviours the AI flagged as inconsistent or suspicious — things that stand out from expected patterns. The Recommended Actions section lists what the AI suggests the investigator should do next — for example "Review all transactions between 01 Jan and 31 Mar", "Request PEP declaration form from subject", or "Consider filing STR if structuring pattern continues".

### FATF Typology tagging

The typology section lets investigators formally tag which FATF money laundering patterns are present in this case. This matters because typologies feed into the pre-flight checklist for STR filings and into AI-generated narratives.

The AI analyses the case's risk factors and suggests which typologies from the FATF library are most likely relevant — these appear with a violet "AI Suggested" badge. Investigators can accept a suggestion by clicking "Tag", or they can manually search for and add typologies themselves using the Tag Typology button, which opens a searchable modal with the full FATF library.

Tagged typologies can be removed at any time.

### Entities section

At the bottom, the AI lists all the individuals, companies, and accounts it has identified from the case documents and transactions. Each entity is shown with a link to their subject profile if one exists in the system.

---

## Tab 7: AI Logs

**Route:** `/cases/{id}/ai-logs`

### What this page does

The AI Logs tab is the complete record of every AI interaction on this case — every copilot query, every narrative generation, every analysis run. It exists for one primary reason: regulators require that AI-assisted decisions in financial crime investigations have a documented human review step. This tab is where that review happens and where the record of who reviewed what, and when, is stored.

### Why human review of AI outputs matters

If an AI generates a case narrative that says a transaction was suspicious, and that narrative is used in an STR filing, the regulator wants to know that a qualified human reviewed and approved that AI output before it was used. Without this, the filing would rest entirely on an algorithm's judgement — which is not acceptable under DPDP 2023 (India), the EU AI Act (EU), or CBUAE standards (UAE). The AI Logs tab is how OpsOwl demonstrates this oversight happened.

### What each log entry shows

Each entry shows:
- What type of AI action it was (copilot query, summary generation, insights analysis, narrative creation, etc.)
- Who triggered it and when
- Which AI model generated the response
- A snippet of what the AI was asked and what it responded
- The current review status: Pending, Approved, Edited, or Rejected

### The 4-eye rule for AI logs

The same person who triggered an AI interaction cannot review their own AI output. This is the same principle as finding approvals — it ensures an independent human has genuinely checked the AI's work rather than just rubber-stamping their own request. If you triggered the AI action, you will see a message saying "You generated this — another investigator must review" instead of action buttons.

### Review actions

For pending AI outputs, a qualified reviewer (a different analyst or manager) can:
- **Approve** — confirming the AI's output is accurate and can be used
- **Reject** — marking the output as unacceptable, with a required reason. A rejected output cannot be used in narratives or STR filings.
- **Edit** — modifying the AI's response before approving it. The edited version is saved alongside the original, and the entry is marked "Edited" so it is clear the output was human-modified. This is important for the audit trail — the regulator can see what the AI originally said and what the human changed it to.

---

## Tab 8: STR Filings

**Route:** `/cases/{id}/str`

### What this page does

This tab is where Suspicious Transaction Reports (STR), Suspicious Activity Reports (SAR), and Cash Transaction Reports (CTR) are drafted, reviewed, approved, and filed with the relevant financial intelligence unit (FIU-IND in India, Hellenic FIU in Greece, CBUAE in UAE, etc.).

### Why analysts cannot see this tab

This tab is restricted to Managers, Admins, and Super Admins. Analysts — even senior ones — cannot access it. This is a legal requirement called the **tipping-off restriction**. AML laws in most jurisdictions make it a criminal offence to tip off a suspect that they are under investigation or that a report has been filed about them. If analysts could see the STR tab, there would be a risk (however small) that an analyst with a personal relationship to the subject could inadvertently or intentionally inform them. Restricting STR access to senior compliance officers mitigates this risk.

### How a filing is created

A filing starts as a **Draft**. The manager fills in:

- **Report type:** STR (suspicious transaction), SAR (suspicious activity), or CTR (cash threshold report)
- **Submission code:** Whether this is a new report (E), a supplement to an earlier report (S), or a correction of a prior report (C)
- **Suspicion date:** The date when the suspicious activity was first identified. This date starts the regulatory deadline clock — the organisation typically has 30 days from this date to file (configurable per jurisdiction).
- **Suspicion reason:** A narrative explaining why the activity is suspicious
- **Transaction list:** The specific transactions being reported, each with date, amount, transaction mode (cash, transfer, wire, deposit, other), description, reference number, location, and the sending and receiving account and institution details
- **Reporting officer and entity details:** Required for XML export — the name, FIU registration number, and contact details of the officer signing the filing, and the institution they represent

### The auto-populate shortcut

Filling in the form manually from scratch on a complex case can take time. The "Auto-Populate from Case" button runs four data fetches simultaneously — subject details, existing narratives, AI insights, and case transactions — and pre-fills as many form fields as possible. Managers should always review and verify the pre-filled data before submitting.

### CTR threshold warnings

If any transaction amount reaches or approaches the Cash Transaction Report threshold (configured per jurisdiction — for example, ₹10 lakh in India), the amount field shows a warning badge:
- A red pulsing **"CTR"** badge when the amount equals or exceeds the threshold
- An amber **"Near-CTR"** badge when the amount is within 10% of the threshold

This helps ensure that cash transactions requiring separate mandatory reporting are not missed.

### The regulatory deadline countdown

Once a suspicion date is entered, a countdown clock appears showing how many days remain before the filing deadline. If the deadline has passed, it shows how many days the organisation is overdue — in red.

### Jurisdiction-specific validation

The form adapts to the deployment jurisdiction:
- **UAE deployments:** Subject ID fields validate Emirates ID format (the 15-digit number formatted as xxx-yyyy-xxxxxxx-x with check digit validation)
- **EU deployments:** Transaction counterparty account fields validate IBAN format and BIC/SWIFT code format

### The pre-flight checklist

Before submitting a filing, managers can expand the Pre-Flight Checklist panel to see how ready the filing is. The checklist has four sections, showing how many of four checks pass:

1. **Structural completeness** — A server-side check that validates the XML structure against the required goAML or FINnet schema. This catches missing mandatory fields before submission.
2. **Narrative quality** — A client-side check that analyses the suspicion reason text. It checks whether the narrative is long enough (at least 100 words) and whether it answers the basic journalistic questions: who was involved, what they did, when it happened, and why it is suspicious.
3. **Transaction risk signals** — A client-side check that looks for concerning patterns in the transaction list: amounts just under the reporting threshold (possible structuring), round number amounts, transactions missing counterparty information, or many transactions on the same day between the same parties.
4. **FATF typology match** — A client-side check that verifies at least one FATF typology has been tagged in the Insights tab. If none have been tagged, this suggests the investigation may not have formally identified the money laundering method.

### The approval process (maker-checker)

Once the draft is ready, the filing goes through an approval process before being filed:

1. **The manager submits the draft** — it moves to "Submitted" status
2. **A different manager approves it** — they cannot be the same person who submitted. If the current user submitted the filing, the approve and reject buttons are replaced with: "You submitted this filing — a different officer must approve."
3. **Once approved, an admin marks it as filed** — entering the FIU reference number confirming the regulatory submission was received

### Exporting the XML

After the filing is approved, it can be exported as a machine-readable XML file for submission to the FIU portal:
- **India:** FINnet 2.0 XML format (via the `/export` endpoint)
- **All other jurisdictions:** goAML XML format (via the `/export/goaml` endpoint)

If the XML export fails because required fields are missing, the server returns a validation error and the checklist panel shows exactly which fields need to be corrected.

---

## Tab 9: Communications

**Route:** `/cases/{id}/comms`

### What this page does

The Communications tab is the log of all direct contact between the investigation team and the subject or related parties. Every phone call, email, field visit, letter, WhatsApp message, and SMS that is part of the investigation should be logged here. This creates an indisputable record of when the organisation reached out, what was said, and how the subject responded.

### Why this matters

Regulators and courts sometimes ask for proof that the organisation attempted to get clarification from a subject before filing a report. The Communications log provides this. It also prevents institutional memory loss — if the investigator who handled a call leaves the organisation, the log ensures the next person can pick up exactly where they left off.

### What a communication log records

Each entry captures:
- **Channel** — how the communication happened: email, phone call, field visit, formal letter, WhatsApp, SMS, or other
- **Direction** — whether it was inbound (the subject or party contacted the organisation) or outbound (the organisation contacted them). Inbound appears with a blue badge; outbound with a purple badge.
- **Contact details** — the name of the person contacted, and their phone number or email
- **Subject and body** — what the communication was about and what was said (for written channels) or a summary of the conversation (for calls)
- **Outcome** — what was agreed or decided as a result
- **Date and time** — when the communication happened (defaults to the current time but can be set to any past time for logging retrospectively)

### AI extraction

After you save a communication, the AI engine reads the body text in the background and automatically extracts:
- **Entities mentioned** — names of people, companies, account numbers, or amounts that appear in the communication
- **Action items** — commitments made in the communication (for example, "customer will provide bank statements by Friday")
- **Sentiment** — the tone of the communication: positive (cooperative), neutral, negative (resistant or unhelpful), or hostile. Hostile communications are shown with a red badge and are a red flag in the investigation.

This extraction appears on the communication card without any extra work from the investigator.

### Automatic refresh

Unlike most tabs which load once, the Communications tab automatically refreshes its list every 10 seconds. This ensures that when the AI finishes extracting entities from a communication you just logged, the extraction results appear on your screen without you needing to manually refresh.

### Deleting communications

Investigators can only delete their own communications — not communications logged by colleagues. This prevents one investigator from erasing another's work from the record.

---

## Tab 10: Narrative

**Route:** `/cases/{id}/narrative`

### What this page does

The Narrative tab is where AI-generated written summaries of the investigation are created, reviewed, and approved. These narratives serve as the formal written record of what happened in the investigation. They are used directly in STR filings, case closure documentation, and as a starting point for investigator-written reports.

### The four types of narrative

OpsOwl can generate four different types of narrative, each with a different purpose:

- **Investigation Summary** — A comprehensive narrative of the entire investigation: who the subject is, what suspicious activity was detected, what the investigation found, and what was done about it. This is the most general narrative.
- **STR Suspicion Basis** — A formal legal narrative specifically structured for inclusion in an STR filing. It explains the legal basis for suspicion in terms that satisfy FIU requirements.
- **SAR Narrative** — Similar to the STR suspicion basis but formatted for a Suspicious Activity Report when the matter is being referred to law enforcement.
- **Closure Summary** — A narrative summarising the outcome of the investigation: what was found, what decision was made, and why the case is being closed.

### How generation works

Click "Generate New Narrative" and select the type you need. An optional "Investigator Notes" text box lets you give the AI additional context or direction — for example, "Focus on the structuring pattern in January and the PEP connection. The subject has been cooperative but explanations are inconsistent."

After clicking Generate, the AI reads the case's approved findings, linked transactions, subject profile, and AI insights, then writes the narrative. This typically takes 30 to 60 seconds. The generation is always logged in the AI Logs tab so it can be reviewed and approved.

### Reviewing and approving a narrative

A freshly generated narrative has "Draft" status. A manager (who must be a different person from whoever generated it — the 4-eye rule again) reviews it and can:
- **Approve** — The narrative is formally accepted and can be used in STR filings and closure documents
- **Reject** — The narrative is not acceptable, with a required reason. The investigator can generate a new one.

### Editing before approval

If the AI narrative is mostly right but needs some adjustments, investigators can edit it inline — click the pencil icon, make changes in the text area, and save. The edited version is saved alongside the original AI-generated text. The narrative card shows an amber "edited" badge so reviewers know the text has been modified from the original. The audit trail preserves both the original AI output and the human-edited version.

### What a narrative card shows

Each narrative card shows the type label, current status badge (Draft / Approved / Rejected), the generated text (collapsed to a preview by default, expandable to full text in a readable serif font), the generation date, who generated it, which AI model was used, the word count, and how many approved findings and transactions were incorporated into it.

Two utility buttons — Copy to clipboard and Download as .txt — make it easy to move the text into other documents or filing systems.

---

## Tab 11: Transactions

**Route:** `/cases/{id}/transactions`

### What this page does

The Transactions tab is where all financial transactions relevant to this investigation are stored. These might be transactions from a bank's core banking system, manually entered by the investigator, or bulk-imported from a CSV export. The tab provides two views: a detailed list view and a visual Flow of Funds diagram.

### The summary bar

Six key numbers are always visible at the top: the total number of transactions, the total monetary value flowing through the case (the "total flow"), the total value of debit transactions (outflows), the total value of credit transactions (inflows), how many transactions have been flagged as suspicious, and how many unique accounts appear across all transactions.

### The transaction list

The list shows all transactions with these details per row:
- Date of the transaction
- Reference number (in monospace font)
- The flow: who sent money and who received it, shown as "From Account → To Account"
- The channel through which it was transferred (NEFT, RTGS, IMPS, UPI, SEPA, SWIFT, cash, card, etc.)
- The amount — debit amounts appear in red, credit amounts in green
- A risk score badge (0-100) showing how suspicious this individual transaction is
- CTR warning badges if the amount is at or near the reporting threshold

Clicking any row expands it to show the full details: the complete from-account and to-account information including institution names, the narration (description), the location, and edit/delete options.

### Searching and filtering

A search box at the top filters the list in real time by reference number, counterparty names, account numbers, or narration text. A "Flagged" toggle filters to show only transactions that have been marked as suspicious.

### Flagging suspicious transactions

Any transaction can be flagged by clicking the flag icon on its row. Flagged transactions appear with a red flag badge and count in the summary bar's "Flagged" total. Flagged transactions are also highlighted in the Flow of Funds diagram and are given extra weight by the AI when generating narratives.

### Adding transactions manually

The "Add Transaction" button opens a form. The form adapts to the deployment jurisdiction:
- In **EU deployments**, account number fields validate IBAN format (a specific international bank account number format starting with a two-letter country code) and institution fields validate BIC/SWIFT code format. Channel options show SEPA, TARGET2, SWIFT, and CHAPS first.
- In **India deployments**, no IBAN or BIC validation is applied. Channel options show NEFT, RTGS, IMPS, and UPI first.

### Bulk CSV import

For cases with many transactions, a three-step import wizard is available. In Step 1 you upload a CSV file (up to 200 rows). In Step 2 the system tries to automatically match your CSV column headers to OpsOwl's field names — it can recognise common naming conventions ("Txn Reference", "Transaction Date", "Amount", etc.) — and shows a mapping interface where you can correct any mismatches. The minimum required columns are the transaction reference and date. In Step 3, the import runs and the results show how many rows succeeded and details of any rows that failed validation.

### The risk score rubric

The risk score badge on each transaction is colour-coded:
- **Green (0–39):** Low risk — normal transaction
- **Amber (40–69):** Medium risk — worth reviewing
- **Orange (70–84):** High risk — should be included in findings
- **Red (85–100):** Critical risk — likely relevant to the investigation's core suspicion

### Flow of Funds diagram

Switching to the Flow of Funds view renders a network graph of all the transactions. Accounts and entities appear as nodes (circles). Money flows appear as arrows between them, labelled with the total amount that flowed along that path.

If the diagram detects circular flows — where money flows from Account A to Account B to Account C and back to Account A — a red banner appears at the top saying "Circular Flow Detected". Circular flows are a classic indicator of layering, which is one of the key money laundering methods in the FATF typology library.

---

## Tab 12: Case Closure

**Route:** `/cases/{id}/closure`

### What this page does

The Closure tab is where the investigation is formally concluded. It ensures that before a case is closed, the investigator has documented what was found, what decision was made about whether to file a report, and why. A manager must then approve the closure before the case status changes to Closed. This two-person sign-off mirrors the maker-checker principle used throughout the system.

### The three-step workflow

**Step 1 — Pre-closure checklist**

Before you can even draft the closure, the system runs an automated checklist to verify the investigation is complete. The checklist checks things like: are all tasks completed, has an STR been reviewed, have all findings been approved, are all UBO records verified (if UBO verification is enabled)?

Each check is either a **blocking** item (the case absolutely cannot close until this is resolved — shown with a red "BLOCKING" badge) or a **non-blocking** item (a warning that should be addressed but will not prevent closure). The step is only complete when all blocking items are resolved.

**Step 2 — Closure form**

Once the checklist passes, you fill in the closure details:

- **Disposition:** The official outcome of the investigation. The six options are:
  - *STR Filed* — A report was filed with the regulator
  - *No STR Warranted* — The investigation was completed but the activity was found not to be suspicious enough to warrant a report
  - *False Positive* — The original alert or concern was found to be incorrect
  - *Closed — Under Monitoring* — The case is closed but the subject is being placed under enhanced monitoring going forward
  - *Escalated to Law Enforcement* — The matter was referred to the police or prosecution authorities
  - *Referred Internally* — The matter was passed to another team within the organisation

- **Reason code:** A more specific categorisation of why this disposition was chosen (for example, "customer provided satisfactory clarification" or "insufficient evidence to proceed")

- **Closure summary:** A written explanation of the investigation's findings and decision (minimum 100 characters — a character counter shows progress). This is the investigator's own words, separate from the AI-generated narrative.

- **STR filed / SAR considered fields:** If the disposition is one of the "no report filed" outcomes (No STR Warranted, False Positive, Closed Under Monitoring), the system requires the investigator to explicitly check a box confirming that they considered filing a report and consciously decided against it. This is a regulatory safeguard — it prevents cases from being closed without someone making an active decision about reporting obligations.

**Step 3 — Submit for manager review**

After saving the draft, you submit it. The closure goes to "Submitted" status and appears in the manager's review queue.

### Manager review

The manager sees the complete closure details and either:
- **Approves** — the case status moves to Closed and the closure is locked permanently
- **Rejects** — the investigator is notified with the rejection reason and must revise the closure and resubmit

As always, the manager cannot approve a closure they themselves submitted.

### What the approved closure looks like

Once approved, the closure section displays as a formal read-only record showing: the disposition, the reason code, the risk score at the time of closure (this is snapshotted so the historical record shows the risk at the time, even if the score changes later), who prepared the closure and when, who approved it and when, and the full closure summary text.

### Reopening a closed case

If a closed case needs to be reopened — for example, new information comes to light — only Admins and Super Admins can do this. Reopening voids the closure (status becomes "Voided"), resets the case to "In Progress", and records the reason for reopening in the audit trail. The voided closure record is preserved so there is a complete history.

---

## Tab 13: UBO (Ultimate Beneficial Ownership)

**Route:** `/cases/{id}/ubo`

### What this page does

When the subject of an investigation is a company or other legal entity, the law requires that investigators identify and verify the real human beings who ultimately own or control that entity — the Ultimate Beneficial Owners. This tab is where that verification is recorded.

### Why this exists

Criminals frequently use complex corporate structures — chains of companies across multiple jurisdictions — to hide their identity behind anonymous ownership. AML frameworks require that investigators "look through" these structures to find the actual people in control. EU AMLR 2024 Article 42 formally requires this verification for any entity with beneficial owners above the threshold (typically 25% ownership or control).

### The ownership threshold

The threshold is configurable per deployment but defaults to 25%. This means any individual owning or controlling more than 25% of the subject entity must be identified and verified. The page subtitle always shows the current threshold so investigators know which owners must be recorded.

### What each UBO record captures

For each beneficial owner, the investigator records:
- The legal entity being investigated (the company name)
- The owner's full name
- Their nationality
- Their ID document type and number
- Their date of birth
- What percentage of the entity they own or control
- The type of control (through share ownership, through voting rights, or through other mechanisms)
- A link to the relevant corporate registry where this ownership can be independently verified

### Verification statuses

Each UBO record has one of four statuses:

- **Pending** (amber spinning icon) — The record has been created but the ownership has not yet been verified against an external source
- **Verified** (green checkmark) — The ownership has been confirmed against the corporate registry or other authoritative source
- **Discrepant** (red triangle) — The declared ownership does not match what the registry shows. A mandatory note explaining the discrepancy is attached.
- **N/A** (grey X) — This record is not applicable — for example, an owner was initially flagged but is below the threshold, or the entity type means UBO rules do not apply

### The closure blocking rule

If any UBO record on the case is still in Pending or Discrepant status when the investigator tries to close the case, the pre-closure checklist blocks the closure. An amber banner at the top of the UBO tab also warns: "Case cannot be closed — UBO records have unresolved status." This ensures investigators cannot close a case while ownership verification is still outstanding.

### Who can manage UBO records

Only Managers and above can add, verify, mark as discrepant, or delete UBO records. This ensures that ownership verification decisions are made by experienced senior investigators rather than junior analysts.

---

## Tab 14: Notes

**Route:** `/cases/{id}/notes`

### What this page does

The Notes tab (the label may be customised in Settings — it might be called "Memos", "Investigation Notes", or another name in your workspace) is for structured, detailed documentation written by investigators. Notes are more formal than the quick notes available on the Overview page and more permanent than messages in the activity feed. They are a core part of the investigation record.

### Internal vs. Formal notes

Notes come in two types:

- **Internal notes** — Working notes visible to all investigators on the team but not included in formal regulatory outputs. Used for thinking out loud, sharing observations with colleagues, or noting things that are interesting but not yet conclusions.

- **Formal notes** — Official investigation memoranda that appear in the PDF case report and in regulatory documentation. These should be written with the same care as any document that might be shown to a regulator or presented in legal proceedings.

### Live collaboration

The Notes tab uses a live connection (Server-Sent Events) so that notes written by one investigator appear on a colleague's screen within seconds — without either person needing to refresh. This allows real-time collaboration on active investigations.

If the connection drops, the system automatically attempts to reconnect, waiting progressively longer between attempts (2 seconds, then 4, then 8, up to a maximum of 60 seconds between retries). A small indicator in the header shows whether the live connection is active.

### Writing a note

The NoteEditor component provides a rich-text writing experience. You can format text (bold, italic, bullet lists), write the note title, choose whether it is internal or formal, and tag colleagues with @mentions. When you mention a colleague, they receive an in-app notification that they have been mentioned in a note on this case.

The note content is stored as HTML (preserving formatting) alongside a plain-text version used for search.

### Loading notes

Notes load in batches of 20, from newest to oldest. As you scroll down you will see older notes — clicking "Load more" at the bottom fetches the next batch. A dropdown filter lets you view only Internal notes or only Formal notes.

### Editing and deleting

Investigators can edit or delete their own notes at any time. Notes from other investigators are read-only. Deleting a note removes it permanently — there is no undo. Editing a note is done inline by clicking the edit icon on the note card.

---

## Cross-Tab Reference Summary

This section summarises the rules and patterns that appear across multiple tabs, so you understand the principles behind the design without having to read every section individually.

### User access levels

Every feature in OpsOwl enforces one of five access levels. Throughout this guide, "analyst+" means an analyst, manager, admin, or super admin. "Manager+" means a manager, admin, or super admin.

| Role | Key capabilities across all case tabs |
|------|--------------------------------------|
| Viewer | Can read everything but cannot create or edit anything |
| Analyst | Can do all investigative work: create tasks, upload documents, write notes, log communications, create findings, run AI analysis, generate narratives |
| Manager | Can approve or reject findings, narratives, and STR filings submitted by analysts; can approve case closures; has exclusive access to the STR tab; can manage UBO records |
| Admin | Everything a manager can do, plus reopening closed cases and managing all administration pages |
| Super Admin | All capabilities |

### The 4-eye (maker-checker) rule

Five features in OpsOwl require that the person who creates or submits something must be different from the person who approves it:

| Feature | What is prevented |
|---------|------------------|
| Findings | The investigator who submits a finding cannot approve it |
| Narratives | The investigator who generates a narrative cannot approve it |
| STR Filings | The officer who submits the filing for approval cannot approve it themselves |
| AI Logs | The person who triggered an AI action cannot review that AI output |
| Case Closure | The investigator who submits the closure cannot approve it themselves |

This rule exists because a single person controlling both the creation and approval of a compliance decision creates an opportunity for error or misconduct. Two people signing off on the same action is standard internal control practice in financial institutions.

### Real-time updates

Not all tabs work the same way when it comes to showing new information. There are three different approaches:

- **Live connection (SSE):** The Timeline and Case Overview use a live server connection. New events appear the moment they happen — no action needed from the user. The Notes tab also uses a live connection with automatic reconnection logic.
- **Automatic polling:** Some tabs check for updates on a regular interval. Documents checks every 5 seconds while processing is ongoing. Communications checks every 10 seconds. Insights checks every 3 seconds while an analysis is running. Once the reason for checking has resolved (all documents processed, analysis complete), the automatic checking stops.
- **Manual refresh:** Most other tabs load fresh data when you navigate to them. They do not automatically check for changes while you are on the page.

### The Closure singleton rule

The Closure tab behaves differently from most resources in the system. There is only ever one closure per case. When you load the Closure tab on a case that has never been closed, the server returns a response with `status: "not_started"` rather than an error. This is intentional — it allows the UI to show the "Start Closure" workflow instead of an error page, and it means the Closure tab always works correctly regardless of whether a closure has been initiated.

---

## Appendix: Quick Reference

### AI model

The default AI model used for insights, narratives, and copilot is `llama3.2:3b` running locally via Ollama. Pull this model before any demo or live session: `ollama pull llama3.2:3b`. The qwen2.5:7b model is available but significantly slower on CPU hardware.

### Role hierarchy reference

```
Super Admin  →  Admin  →  Manager  →  Analyst  →  Viewer
(highest)                                          (lowest)
```

### Workspace terminology

Every label in the application that refers to cases, tasks, documents, notes, and regulatory filings can be customised in Settings → Workspace Terminology. If something in this guide says "Cases" but your deployment calls them "Investigations", those are the same thing. The underlying functionality is identical.

### Jurisdiction-conditional features

Certain features only appear when the deployment is configured for a specific jurisdiction:
- **goAML XML export** — available when `DEPLOYMENT_JURISDICTION` is not `india`
- **UBO verification tab** — available when `UBO_ENABLED=true`
- **GDPR DSAR management** — available when `GDPR_ENABLED=true`
- **IBAN/BIC validation in transactions and STR** — active when jurisdiction is `eu`
- **Emirates ID validation in STR** — active when jurisdiction is `uae`

---

*End of Guide*
