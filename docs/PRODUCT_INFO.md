# OpsOwl AI — Complete Product Information

**Document Owner:** Naguru Reddy Basha  
**Product:** OpsOwl AI — Operations Intelligence Workspace  
**Version:** 1.0  
**Date:** May 2026  
**Status:** Live — MVP Complete, Phase 8 in Progress

---

## Table of Contents

1. [Product Overview](#1-product-overview)
2. [The Problem We Are Solving](#2-the-problem-we-are-solving)
3. [Who We Built This For](#3-who-we-built-this-for)
4. [How OpsOwl Solves the Problem](#4-how-opsowl-solves-the-problem)
5. [The Market Opportunity](#5-the-market-opportunity)
6. [Competitive Landscape](#6-competitive-landscape)
7. [Business Model & Pricing](#7-business-model--pricing)
8. [Sales Strategy](#8-sales-strategy)
9. [Why Now](#9-why-now)
10. [Architecture in Plain Language](#10-architecture-in-plain-language)

---

**Features (Deep Dive)**

11. [Feature 1 — Case Management](#feature-1--case-management)
12. [Feature 2 — Task Management](#feature-2--task-management)
13. [Feature 3 — Document Management](#feature-3--document-management)
14. [Feature 4 — Notes](#feature-4--notes)
15. [Feature 5 — Timeline (Real-Time Event Feed)](#feature-5--timeline-real-time-event-feed)
16. [Feature 6 — Communications (COMMS)](#feature-6--communications-comms)
17. [Feature 7 — SLA Management](#feature-7--sla-management)
18. [Feature 8 — STR / SAR Filing](#feature-8--str--sar-filing)
19. [Feature 9 — AI Risk Scoring & Insights](#feature-9--ai-risk-scoring--insights)
20. [Feature 10 — AI Copilot (Chat Assistant)](#feature-10--ai-copilot-chat-assistant)
21. [Feature 11 — Role-Based Access Control (RBAC)](#feature-11--role-based-access-control-rbac)
22. [Feature 12 — Authentication (JWT + LDAP + Sessions)](#feature-12--authentication-jwt--ldap--sessions)
23. [Feature 13 — Audit Chain (Tamper-Evident Log)](#feature-13--audit-chain-tamper-evident-log)
24. [Feature 14 — Document Encryption & Virus Scanning](#feature-14--document-encryption--virus-scanning)
25. [Feature 15 — API Keys](#feature-15--api-keys)
26. [Feature 16 — Search](#feature-16--search)
27. [Feature 17 — Analytics & Reporting](#feature-17--analytics--reporting)
28. [Feature 18 — Scheduled Reports](#feature-18--scheduled-reports)
29. [Feature 19 — Bulk Operations](#feature-19--bulk-operations)
30. [Feature 20 — Case Templates](#feature-20--case-templates)
31. [Feature 21 — Notifications](#feature-21--notifications)
32. [Feature 22 — Inbound Event Log (Webhook Receiver)](#feature-22--inbound-event-log-webhook-receiver)
33. [Feature 23 — Outbound Webhooks](#feature-23--outbound-webhooks)
34. [Feature 24 — SIEM Integration](#feature-24--siem-integration)
35. [Feature 25 — Monitoring (Prometheus + Grafana)](#feature-25--monitoring-prometheus--grafana)
36. [Feature 26 — Operations Dashboard](#feature-26--operations-dashboard)
37. [Feature 27 — User Management](#feature-27--user-management)
38. [Feature 28 — My Tasks (Cross-Case View)](#feature-28--my-tasks-cross-case-view)
39. [Feature 29 — AI Logs](#feature-29--ai-logs)
40. [Feature 30 — CSV Import (Bulk Case Import)](#feature-30--csv-import-bulk-case-import)
41. [Feature 31 — Case PDF Export](#feature-31--case-pdf-export)
42. [Feature 32 — Data Retention Policies](#feature-32--data-retention-policies)
43. [Feature 33 — Backup & Recovery](#feature-33--backup--recovery)

---

# PART ONE: PRODUCT CONTEXT

---

## 1. Product Overview

**OpsOwl AI** is an AI-powered investigation and operations workspace built specifically for financial institutions — NBFCs, fintechs, digital lenders, payment companies, and cooperative banks.

It is the single platform where an operations or compliance team:
- Receives alerts and opens investigations
- Tracks every action taken on each investigation
- Manages documents, tasks, and team collaboration
- Gets AI-generated risk scores and recommended actions
- Files regulatory reports (STR/SAR) without leaving the platform
- Maintains a tamper-proof audit trail that satisfies RBI and FIU-IND requirements

**In one sentence:** OpsOwl AI replaces the Excel spreadsheets, WhatsApp groups, and email chains that Indian financial institutions currently use to run compliance investigations — with a secure, auditable, AI-assisted workspace that their regulators will respect.

**Tagline:** *See Everything. Miss Nothing. Prove Everything.*

---

## 2. The Problem We Are Solving

### The Reality on the Ground

Walk into the compliance or operations department of a mid-sized NBFC or fintech in India today. Here is what you will typically find:

- An analyst has 3 open investigations tracked in a shared Excel file on Google Drive
- Documents related to those investigations are scattered across email attachments, WhatsApp, and a shared drive folder with no naming convention
- Notes about customer calls are in a personal Word document or handwritten notepad
- Deadlines (STR filing within 7 days per RBI mandate) are tracked in a personal calendar reminder
- When the analyst is on leave, the next person has no idea where the investigation stands
- When a regulator asks for evidence of due diligence, the compliance head spends 3 days collecting screenshots from 4 different systems

This is not an edge case. This is the standard operating procedure at thousands of regulated financial institutions in India today, particularly in the ₹100 Crore to ₹5,000 Crore AUM segment.

### The Regulatory Pressure

The Reserve Bank of India (RBI) and Financial Intelligence Unit — India (FIU-IND) have been dramatically tightening compliance requirements:

- **Suspicious Transaction Reports (STR):** Must be filed within 7 days of identification. FIU-IND levies penalties for late or missed filings
- **KYC compliance:** Periodic re-KYC with documented investigation trails
- **Audit readiness:** RBI expects complete, verifiable documentation of every compliance decision
- **Data localization:** Customer financial data must stay within India

The compliance burden is growing faster than the workforce that handles it. A team of 10 analysts cannot manually track 500 cases/month across Excel files and still meet every regulatory deadline.

### The Technology Gap

The institutions that can afford proper compliance technology — State Bank of India, HDFC, ICICI — use Oracle FCCM, NICE Actimize, or SAS AML. These platforms cost ₹2–5 Crore per year in licensing, take 6–12 months to implement, and require dedicated vendor support teams.

For the 10,000+ other regulated institutions in India — the mid-market — these platforms are simply out of reach. They fall back on Excel.

**OpsOwl fills that gap.** It delivers enterprise-grade investigation capability at a price point and implementation speed that the mid-market can actually afford and use.

---

## 3. Who We Built This For

### Primary ICP (Ideal Customer Profile)

**Institution Type:**
- NBFCs with ₹200 Crore – ₹5,000 Crore AUM
- Digital lending platforms (personal loan, BNPL, gold loan apps)
- Payment aggregators and payment service providers
- Urban co-operative banks
- Microfinance institutions (MFIs) under RBI supervision
- Fintech companies with a lending or payments license

**Team Profile:**
- 5–100 person compliance / operations team
- Dedicated compliance officer or Chief Risk Officer
- Currently using Excel + email + WhatsApp for case management
- Under RBI, FIU-IND, or SEBI oversight
- Have faced or are afraid of a regulatory notice for documentation failures

**Pain Level:**
- They are stressed about audit readiness
- They have missed or nearly missed STR filing deadlines
- They lose institutional knowledge when analysts leave
- They cannot answer "who did what, when, and why" without a multi-day document hunt

### Secondary ICP

- Insurance companies under IRDAI oversight
- Stockbrokers and mutual fund distributors (SEBI compliance)
- Crypto exchanges (FIU-IND VDA reporting requirements)
- Compliance consulting firms that manage investigations on behalf of clients

---

## 4. How OpsOwl Solves the Problem

OpsOwl replaces five different tools — Excel, email, shared drives, WhatsApp, and a ticketing system — with a single purpose-built workspace.

### The Before vs. After

| Situation | Before OpsOwl | After OpsOwl |
|-----------|--------------|-------------|
| Alert received from TMS | Analyst opens an Excel row | Webhook auto-creates a case |
| Documents collected | Email attachments in a folder | Uploaded directly to the case, virus-scanned and encrypted |
| Investigation notes | Word doc or WhatsApp message | Rich-text notes inside the case, timestamped and immutable |
| Task assignment | Email to a colleague | Task created in-platform, deadline set, notification sent |
| STR filing deadline | Calendar reminder | SLA countdown visible to analyst and manager; breach triggers alert |
| STR filing itself | Fill PDF form, email to FIU-IND | Generated and tracked from within the case |
| Audit by regulator | 3-day document hunt | Complete tamper-proof case history, exportable in minutes |
| Manager review | Chasing analyst on WhatsApp | Pending review queue visible on dashboard |
| AI-assisted decision | Not available | AI risk score + entity extraction + recommended actions on every case |

### The Five Core Promises

1. **Nothing gets lost** — every document, note, decision, and communication is inside one case record
2. **Nobody misses a deadline** — SLA countdowns, alerts, and escalations are automatic
3. **You can prove everything** — hash-chained audit trail satisfies any regulator
4. **AI makes you faster** — risk scoring and recommended actions reduce analyst decision time
5. **Your data stays with you** — on-premise deployment, no cloud vendor dependency

---

## 5. The Market Opportunity

### Indian Regtech Market

| Metric | Value |
|--------|-------|
| India regtech market size (2025) | ~$3.2 Billion |
| CAGR | 29% through 2030 |
| Projected size (2030) | ~$11 Billion |
| Registered NBFCs in India | 10,000+ |
| Fintechs in India (2025) | 100,000+ |
| Active digital lenders | 1,500+ |
| Urban co-operative banks | 1,500+ |

### Addressable Market for OpsOwl

If we target NBFCs and fintechs in the ₹200 Crore–₹5,000 Crore AUM bracket — approximately 3,000 institutions — and assume:

- Average contract value: ₹15–40 Lakhs/year
- Realistic penetration in 5 years: 2% (60 clients)
- Conservative ARR at 60 clients: ₹12–24 Crore/year

At 5% penetration (150 clients): ₹30–60 Crore ARR. This is a defensible, profitable business without raising external capital, built entirely on enterprise contracts.

### Why the Timing Is Right

- FIU-IND reported 15% year-on-year increase in STR filings in FY2025
- RBI issued 3 major compliance circulars in 2024–25 that increased documentation requirements
- PMLA (Prevention of Money Laundering Act) amendments in 2023 expanded the scope of obligated entities
- Indian fintechs are maturing — the early growth phase is over; compliance is now the bottleneck for scaling
- Data localization concerns are making on-premise solutions more attractive than cloud-first alternatives

---

## 6. Competitive Landscape

### Direct Competitors

| Competitor | Strength | Weakness | OpsOwl Advantage |
|------------|----------|----------|-----------------|
| **Oracle FCCM** | Deep bank integrations, decades of data | ₹3–5 Cr/year, 9-month implementation, requires vendor team | 50x cheaper, deploys in 1 week, no vendor lock-in |
| **NICE Actimize** | Best transaction monitoring rules engine | Priced for top-10 banks only, not investigation-focused | Purpose-built investigation UX, AI-native |
| **SAS AML** | Powerful analytics | Complex, expensive, requires SAS expertise | Simple, modern UI, REST API, no specialist needed |
| **Hawk AI** | Good AI detection | Thin case management layer | Deep investigation workflow, STR filing, document vault |
| **Napier AI** | Modern UI | UK/EU-focused, limited India compliance | Built for India — RBI, FIU-IND, STR/SAR natively |
| **In-house tools** | Free, customized | No audit trail, no AI, maintenance burden | Complete product, maintained, AI-powered |

### Indirect Competitors

- **Jira + Confluence:** Used by tech-forward teams. No compliance domain knowledge. No STR filing. No audit chain.
- **Salesforce:** Case management capability but no compliance features. ₹3,000+/user/month.
- **Excel + SharePoint:** Free. No audit trail. Regulators don't accept "our Excel file" as evidence.

### OpsOwl's Moat

1. **On-premise first** — the only modern investigation platform designed to run entirely on the client's own infrastructure
2. **India-first compliance** — STR/SAR in FIU-IND format, SLA rules aligned to RBI circulars, not retrofitted from US/EU tools
3. **Price disruption** — full platform at ₹10–40 Lakhs vs. ₹2–5 Crore for alternatives
4. **AI included, not upsold** — risk scoring, entity extraction, and AI Copilot are core features, not add-ons
5. **Modern tech stack** — real-time updates, REST API, webhook integration — while competitors run on 2005-era architecture

---

## 7. Business Model & Pricing

### Deployment Model

OpsOwl is sold as an **on-premise license** — the client installs and runs it on their own servers. This is:
- A requirement for many regulated institutions (data localization)
- A feature, not a limitation (differentiates from cloud SaaS)
- Operationally simple — Docker-based deployment, single docker-compose command

### Pricing Tiers (Proposed)

| Tier | Target | Users | Annual License |
|------|--------|-------|---------------|
| **Starter** | MFIs, small NBFCs | Up to 25 users | ₹8–12 Lakhs/year |
| **Growth** | Mid-size NBFCs, fintechs | Up to 100 users | ₹18–28 Lakhs/year |
| **Enterprise** | Large NBFCs, banks | Unlimited users | ₹40–60 Lakhs/year |
| **Implementation** | All tiers | One-time | ₹3–8 Lakhs |
| **Annual Support** | All tiers | Included yr 1 | 18% of license from yr 2 |

### Revenue Streams

1. **Annual license** — recurring, predictable
2. **Implementation and onboarding** — one-time, high-margin
3. **Annual maintenance and support** — 18% of license value per year
4. **Custom development** — specific regulatory format integrations, CBS connectors
5. **Training** — on-site analyst and admin training (₹1–2 Lakhs per engagement)

### Unit Economics (Growth Tier Example)

- Average contract: ₹25 Lakhs/year license + ₹5 Lakhs implementation
- Year 1 revenue per client: ₹30 Lakhs
- Year 2+: ₹25 Lakhs license + ₹4.5 Lakhs support = ₹29.5 Lakhs/year
- Gross margin (software): ~85% (hosting is client's cost)
- Payback period: < 12 months

---

## 8. Sales Strategy

### The Buyer Journey

The buying decision in a regulated financial institution involves three people:

1. **Chief Compliance Officer (CCO) / Chief Risk Officer (CRO)** — The economic buyer. Feels the pain directly. Gets the product demo. Signs the contract.
2. **Head of IT / CISO** — Technical approver. Cares about on-premise deployment, data security, API integration, and not introducing vendor risk.
3. **MD / CEO** — Final approver for budgets above ₹15 Lakhs. Needs the ROI story and the regulatory risk narrative.

### Sales Motion

**Stage 1 — Trigger (The event that creates urgency)**
- RBI issues a new compliance circular
- Competitor NBFC gets penalized for documentation failure
- Internal audit finds compliance gaps
- Analyst leaves and institutional knowledge walks out the door

**Stage 2 — Outreach**
- Direct outreach to CCOs at target NBFCs via LinkedIn and industry events
- Content marketing: "RBI compliance playbook for NBFCs" — positions OpsOwl as the expert
- Presence at FICCI, CII financial services conferences
- Partnerships with CA firms and compliance consulting firms who advise NBFCs

**Stage 3 — Demo**
- 45-minute product demo focused on their specific pain (STR delays, audit readiness)
- Show a realistic case from alert to STR filing — the "magic moment" is when the regulator audit scenario plays out in 2 minutes
- Emphasize: on-premise, their data, their servers

**Stage 4 — Proof of Concept**
- 30-day PoC on their own infrastructure with 5 live cases
- We handle the setup; they use it for their real work
- PoC converts at ~60% if the initial pain was genuine

**Stage 5 — Commercial**
- No per-user pricing (compliance teams hate per-user models — they add people as needed)
- Annual contract with 12-month minimum
- Net 30 payment; 50% upfront + 50% on go-live is standard

### Channel Strategy

| Channel | Expected Contribution |
|---------|----------------------|
| Direct sales (CCO outreach) | 40% |
| CA / compliance firm referral | 25% |
| Industry conference leads | 20% |
| Inbound (content + SEO) | 15% |

### Sales Cycle

- Typical sales cycle: 60–120 days
- Decision committee: 2–4 people
- Longest delay: IT security review (2–4 weeks)
- Fastest close: When RBI notice or audit is pending (< 30 days)

---

## 9. Why Now

Five forces have converged to make 2025–2026 the right moment:

**1. RBI Tightening:** Three major compliance circulars in 2024–25. Institutions that ignored compliance are now facing real consequences. Fear is a powerful buying trigger.

**2. AI Credibility:** ChatGPT and LLMs have made every institution's leadership understand that AI is real and available. The AI Copilot and risk scoring features are now believable, not futuristic.

**3. Data Localization Urgency:** Government and RBI signals on data sovereignty have made "our data leaves India" a procurement blocker. On-premise is a requirement, not a preference.

**4. Fintech Maturity:** The 2015–2023 fintech boom was about growth at all costs. 2024 onwards is about sustainable, compliant growth. Compliance budgets are getting funded.

**5. Analyst Attrition:** Post-COVID, compliance analyst attrition is high. Institutions are realizing that institutional knowledge walks out the door with each analyst. A documented, auditable system is now seen as business continuity infrastructure.

---

## 10. Architecture in Plain Language

For non-technical stakeholders — here is how OpsOwl is built and why the architecture matters to clients:

```
┌─────────────────────────────────────────────────────┐
│                   Client's Browser                   │
│              (Next.js web application)               │
└──────────────────────┬──────────────────────────────┘
                       │ HTTPS
┌──────────────────────▼──────────────────────────────┐
│                 Nginx (Gateway)                      │
│           TLS termination + Rate limiting            │
└─────────┬──────────────────────┬────────────────────┘
          │                      │
┌─────────▼────────┐   ┌────────▼────────────────────┐
│   Go API Server  │   │     Python AI Service        │
│  (Business logic)│   │  (Risk scoring, NLP, LLM)   │
└─────────┬────────┘   └─────────────────────────────┘
          │
┌─────────▼────────────────────────────────────────────┐
│              PostgreSQL (Primary Database)            │
│     All cases, documents metadata, audit chain,      │
│          users, tasks, notes, timeline               │
└──────────────────────────────────────────────────────┘
          │
┌─────────▼────────┐   ┌─────────────────────────────┐
│  MinIO (Storage) │   │  Prometheus + Grafana        │
│ Document files,  │   │  System health monitoring    │
│ encrypted at rest│   │  and alerting                │
└──────────────────┘   └─────────────────────────────┘
```

**Why this matters to the client:**

- **Everything runs on their own server** — no data goes to any cloud. Full control.
- **One command to start** — `docker-compose up` launches the entire platform
- **No external dependencies** — no calls to AWS, Google Cloud, or any third party at runtime
- **High availability** — database backups every 24 hours, point-in-time recovery, WAL archiving
- **Monitored** — the operations team can see system health on a Grafana dashboard at all times
- **Encrypted** — all documents are encrypted with AES-256. Even the database admin cannot read case documents without the encryption key.

---

# PART TWO: FEATURE DEEP DIVES

---

## Feature 1 — Case Management

### 1. What Is It

Case Management is the central workspace where every investigation lives. A **case** is the master record that ties together everything related to an investigation: the subject, the suspicious activity, all evidence, the people working on it, every decision made, and the final regulatory outcome.

Each case contains:
- Unique auto-generated case number (format: `OWL-0001`, configurable)
- Title and full description
- Status, priority, and SLA deadline
- Assigned analyst and reviewing manager
- Tags for categorization (fraud type, geography, product type)
- AI risk score and risk level
- Links to all tasks, documents, notes, timeline, comms, and STR/SAR

**Case Status Lifecycle:**
```
open → in_progress → pending_review → closed
                  ↘ escalated ↗
                  
Any status → re-opened (manager action)
```

**Priority Levels:** Low | Medium | High | Critical — each maps to a different SLA duration

### 2. Why We Implemented It

Every financial institution doing AML, fraud investigation, or credit operations runs investigations. Before tools like OpsOwl, teams used:

- Excel trackers shared over email (no version control, no audit trail)
- Generic ticketing tools (Jira, ServiceNow) not designed for investigations
- Paper files and physical registers (still the norm at co-operative banks and smaller NBFCs)
- WhatsApp groups (completely unacceptable for a regulated environment)

None of these give you a complete, auditable investigation record in one place. Case management is the spine of the entire product — every other feature attaches to a case. Without this foundation, nothing else works.

### 3. How Is It Useful for the Client

**For the analyst:**
- One screen shows everything about an investigation. No switching between Excel, email, and a document folder.
- Status transitions are enforced by the system — you cannot skip `pending_review`, which forces the right compliance process.
- SLA countdown tells the analyst exactly how much time they have before a regulatory deadline.

**For the compliance manager:**
- Dashboard shows all open cases, how many are breaching SLA, and how many are high priority — at a glance.
- Can see which analyst is overloaded and reassign with one click.
- Full history of every action on every case — defensible in any regulator audit.

**For the CISO and auditors:**
- Every case has a tamper-evident audit chain — nobody can alter history.
- Risk scores provide an objective record of the AI's assessment alongside human judgment.
- Evidence is encrypted — documents cannot be read even if the server is physically accessed.

**Real-world scenario:** An NBFC receives a suspicious transaction alert at 9 AM. Without OpsOwl: analyst opens an Excel row, emails the KYC team for documents, keeps notes in a Word file, files an STR 28 days later after missing the 7-day window. With OpsOwl: alert creates a case automatically via webhook, tasks are assigned instantly, documents are requested and uploaded in-platform, the SLA clock counts down visibly, and the STR is filed from the same case — all in 4 days, audit-ready.

### 4. How to Use It

**Creating a case:**
1. Navigate to Cases → New Case
2. Enter title, description, subject details, assign to an analyst, set priority
3. System auto-assigns a sequential case number and starts the SLA clock

**Working a case:**
1. Open the case detail — all context is on one screen
2. Add tasks for each investigation step
3. Upload evidence documents
4. Write investigation notes
5. Click AI Copilot → ask "What's unusual about this customer's transaction pattern?"
6. Trigger AI risk analysis → system returns risk score, anomalies, and recommended actions
7. Add comms entries if customer contact was made

**Closing a case:**
1. Complete all required tasks
2. Add closing note with findings summary
3. If suspicious activity confirmed → file STR/SAR from within the case
4. Change status to `pending_review`
5. Manager reviews → approves closure → marks `closed`

**For automation via API:**
```http
POST /api/v1/cases                 — create a new case
GET  /api/v1/cases?status=open     — list with filters
GET  /api/v1/cases/:id             — full case detail
PATCH /api/v1/cases/:id            — update status, assignee, priority
```

### 5. Is the Implementation Standard Equal to or Greater Than Market

| Aspect | OpsOwl | Industry Standard |
|--------|--------|-------------------|
| Status lifecycle enforcement | Full state machine | Most tools allow free-form status |
| Audit trail per case | Hash-chained, tamper-evident | Append-only logs (weaker) |
| AI risk score integrated | Native, in-platform | Separate module, extra cost |
| Real-time timeline | SSE-based live feed | Polling or manual refresh |
| REST API | Full CRUD, all case data | Legacy tools often SOAP or no API |
| On-premise deployment | Native | Cloud-only for most modern tools |

**Honest rating: 8/10** — Equal to enterprise tools on workflow; ahead on AI integration and audit integrity; behind on pre-built CBS/TMS connectors.

### 6. Why Will the Client Buy It

- **Regulatory fear:** They cannot show an Excel file to an RBI auditor and call it an investigation record. OpsOwl gives them a complete, printable, provable case history.
- **Deadline enforcement:** The 7-day STR filing requirement is non-negotiable. A visible SLA countdown with automatic escalation is the difference between filing on time and paying a penalty.
- **Analyst accountability:** Management can finally see which analyst has how many open cases, where each one stands, and whether work is actually being done.
- **Operational continuity:** When an analyst leaves, the entire investigation history is preserved in the system. No knowledge walks out the door.

---

## Feature 2 — Task Management

### 1. What Is It

Task Management is the to-do layer inside every investigation. Each case can have multiple tasks — specific actions that need to happen to complete the investigation. Tasks have owners, deadlines, priorities, statuses, and comments.

Every task follows a defined lifecycle:
```
todo → in_progress → done
     ↘ cancelled
```

Tasks can be:
- Manually created by analysts or managers
- Auto-suggested by the AI Copilot based on case context
- Linked to a specific timeline event (e.g., "KYC document received — verify authenticity")

Key fields per task:
- Title and description
- Priority (Low / Medium / High / Critical)
- Assigned to (specific analyst)
- Due date with overdue flagging
- Sort order (drag-and-drop re-ordering)
- Comment thread

### 2. Why We Implemented It

An investigation is never a single action. AML investigation of a suspicious transaction might involve 8–12 distinct steps: pulling account statements, verifying KYC, checking transaction history, contacting the customer, requesting explanation, reviewing response, escalating to manager, filing STR. Each step has a different owner and deadline.

Without task management, these steps live in the analyst's head, a personal checklist, or a WhatsApp message. None of those are auditable. With task management, every required step is documented, assigned, tracked, and timestamped.

### 3. How Is It Useful for the Client

**For the analyst:**
- Clear list of exactly what needs to be done on each case — no ambiguity
- Overdue tasks are flagged automatically — no missed steps
- AI suggests tasks based on case type, so junior analysts know what to do

**For the manager:**
- Can see across all cases which tasks are overdue
- Can reassign tasks when an analyst is overloaded
- SLA breaches often trace back to a specific blocked task — now visible

**For the compliance officer:**
- Can prove to a regulator that every required investigation step was performed
- Task completion timestamps provide evidence of when each step occurred

**Real scenario:** RBI guidance requires that before filing an STR, the institution must: verify customer identity, review transaction history, check watchlist, obtain customer explanation, and get manager sign-off. Each of these can be a task on the case — all tracked, all timestamped.

### 4. How to Use It

**Creating tasks:**
1. Open a case → Tasks tab → New Task
2. Enter title, assign to analyst, set due date and priority
3. Task appears in the assignee's "My Tasks" dashboard view

**Working tasks:**
1. Open a task → change status to `in_progress`
2. Add comments as investigation progresses (e.g., "Called customer — no response, trying again tomorrow")
3. Attach relevant notes or reference documents
4. Mark complete when done

**AI-suggested tasks:**
1. Trigger AI Copilot on a case
2. AI analyses case description and context
3. Returns a list of recommended investigation steps
4. Analyst clicks "Add as Task" on any recommendation

**Drag-and-drop ordering:**
- Tasks can be reordered to reflect the correct investigation sequence
- Sort order is preserved and shown to all team members

**Overdue detection:**
- Automated background job runs every 15 minutes
- Any task past its due date is flagged `is_overdue = true`
- Appears highlighted in the task list and counts toward SLA metrics

### 5. Is the Implementation Standard Equal to or Greater Than Market

| Aspect | OpsOwl | Industry Standard |
|--------|--------|-------------------|
| Task state machine | Enforced transitions | Free-form in most tools |
| AI-suggested tasks | Built-in | Rarely available |
| Overdue auto-detection | Background job, real-time flag | Manual or daily batch |
| Drag-and-drop ordering | Yes | Common in modern tools |
| Comment threads per task | Yes | Variable |
| Linked to timeline events | Yes | Rare |

**Honest rating: 7.5/10** — Strong implementation. The AI task suggestion is a genuine differentiator. Lacks subtasks and dependencies (which complex investigations need).

### 6. Why Will the Client Buy It

- Investigation completeness: Clients can demonstrate to regulators that every required step was executed
- Accountability: Clear ownership of each step — no "I thought someone else was handling it"
- AI acceleration: Junior analysts get a running start with AI-suggested task lists instead of figuring out investigation steps from scratch

---

## Feature 3 — Document Management

### 1. What Is It

Document Management is the encrypted, virus-scanned, case-linked evidence vault. Every document uploaded to OpsOwl — bank statements, KYC records, transaction records, customer correspondence, identity documents — is stored securely and linked to the case it belongs to.

Key capabilities:
- Upload any file type (PDF, images, Excel, Word)
- Automatic virus scanning (ClamAV) on every upload
- AES-256 encryption at rest
- Version history (re-upload replaces with new version, old version preserved)
- Soft delete (documents are never permanently destroyed — only marked deleted)
- Download with full audit trail
- Full-text document content extraction (for AI analysis)
- File size limit: 50MB per document (configurable)

### 2. Why We Implemented It

Evidence is the foundation of every investigation. Without a secure document vault, compliance teams store sensitive customer documents in:
- Email attachments (no encryption, no access control)
- Shared drives (no audit trail of who accessed what)
- USB drives on analyst desktops (security disaster)
- Physical files (cannot be audited, cannot be searched)

All of these fail both the security standard and the regulatory standard. RBI requires that evidence supporting an STR filing be preserved for 5 years. Excel cannot do that. Email cannot do that. OpsOwl can.

The virus scanning exists because fintechs receive documents from customers who may unknowingly send infected files. A malicious document that infects the compliance system is a catastrophic breach.

### 3. How Is It Useful for the Client

**For the analyst:**
- All evidence for a case is in one place — no hunting through email
- Documents are searchable by content — find the specific bank statement with the suspicious transfer without opening every file

**For the CISO:**
- AES-256 encryption means even a server compromise does not expose customer documents
- Virus scanning prevents malware infiltration
- Access audit trail shows exactly who downloaded which document and when

**For the regulator:**
- Documents cannot be deleted without a trace
- 5-year retention is enforced at the infrastructure level
- Chain of custody is provable: uploaded by X at time T, downloaded by Y at time T+2

### 4. How to Use It

**Uploading a document:**
1. Open a case → Documents tab → Upload
2. Select file (PDF, image, Excel, etc.)
3. Add description and document type (KYC, Bank Statement, Communication, etc.)
4. System automatically: scans for viruses → encrypts → stores → logs the upload
5. If virus detected: upload rejected immediately with clear error message

**Viewing and downloading:**
1. Documents list shows all uploads with uploader, date, file type
2. Click to download — decryption happens transparently on the server
3. Every download is logged in the audit trail

**Document intelligence:**
1. AI service can extract text from uploaded PDFs and images (OCR)
2. Extracted content is indexed for AI risk analysis
3. Analysts can query: "What does the bank statement say about transaction patterns?"

### 5. Is the Implementation Standard Equal to or Greater Than Market

| Aspect | OpsOwl | Industry Standard |
|--------|--------|-------------------|
| Virus scanning | ClamAV on every upload | Often skipped in mid-market tools |
| Encryption at rest | AES-256, server-side | S3 default encryption (weaker) |
| Soft delete (preservation) | Yes | Permanent delete common in basic tools |
| OCR + AI extraction | Yes | Expensive add-on in enterprise tools |
| Audit trail on every download | Yes | Access logs only in most tools |
| Version history | Yes | Variable |

**Honest rating: 8.5/10** — The combination of virus scanning, AES-256 encryption, and OCR on upload puts this ahead of most mid-market document management. Behind SharePoint/Box on UI polish and preview capabilities.

### 6. Why Will the Client Buy It

- **Regulatory preservation:** RBI requires 5-year document retention. OpsOwl enforces this by policy — soft delete means nothing is actually destroyed.
- **Security credibility:** AES-256 encryption at rest and virus scanning on upload is something the CISO can point to in a security audit and feel good about.
- **Operational efficiency:** All documents for a case in one place, searchable, accessible from anywhere — eliminates the 30-minute document hunt before every team meeting.

---

## Feature 4 — Notes

### 1. What Is It

Notes is the investigation journal inside every case. Analysts write rich-text notes documenting their findings, customer interactions, internal decisions, and investigation progress. Notes are permanent, timestamped, and can never be altered after being saved.

Key capabilities:
- Rich-text editor (headings, bold, bullets, tables, links)
- Note types: Internal (team-only) or External (can be shared)
- Permanent record — notes cannot be edited or deleted once saved
- Attached to a specific case
- Searchable full-text
- Author and timestamp visible on every note
- Tagged with type (observation, decision, customer contact, etc.)

### 2. Why We Implemented It

An investigation generates a large volume of unstructured information: phone call outcomes, analyst observations, document analysis findings, reasons for decisions. This information currently lives in:
- Personal Word documents (lost when analyst leaves)
- WhatsApp messages (not auditable, not searchable)
- Email threads (scattered, hard to find)
- Analyst memory (cannot be audited)

Notes gives every piece of investigation thinking a permanent, auditable home inside the case. Critically, notes are **immutable** — once written, they cannot be changed. This is not a limitation — it is a compliance feature. It means the investigation record cannot be revised retroactively to match a different narrative.

### 3. How Is It Useful for the Client

**For the analyst:**
- Running journal of the investigation — everything they thought, observed, and decided is captured
- When picking up someone else's case, the notes tell the full story instantly

**For the manager:**
- Can see the analytical quality of their team's work
- Notes provide the narrative that explains why decisions were made

**For the regulator:**
- Notes are the evidence of analytical work — "we thought about this, here's why we concluded X"
- Immutability means notes cannot be fabricated after the fact
- Timestamped to the second

### 4. How to Use It

**Adding a note:**
1. Open a case → Notes tab → New Note
2. Select note type (Internal / External)
3. Write using the rich text editor — formatting, bullets, tables all supported
4. Save — note is immediately locked (cannot be edited)

**Reading notes:**
- Displayed in reverse-chronological order (newest first)
- Each note shows: author, timestamp, note type
- Full-text searchable across all notes in a case

### 5. Is the Implementation Standard Equal to or Greater Than Market

**Honest rating: 7/10** — The immutability is a genuine strength. The rich text editor (Tiptap) is modern and capable. Missing: @mention of colleagues within a note, note templates, and annotation on specific document sections.

### 6. Why Will the Client Buy It

Immutable, timestamped investigator notes are the closest thing to a court-admissible investigation record. Regulators, internal auditors, and courts all want to see the investigator's reasoning — not just the conclusion. Notes provide that narrative in a tamper-proof format.

---

## Feature 5 — Timeline (Real-Time Event Feed)

### 1. What Is It

The Timeline is the living history of every case — a real-time, chronological feed of every event that happens to an investigation. Every status change, every document upload, every note added, every task completed, every AI analysis run, every STR filed — all appear in the timeline automatically, in real time, as they happen.

Key capabilities:
- Real-time updates via SSE (Server-Sent Events) — no page refresh needed
- Automatic capture of every system event
- Manual event creation (e.g., "Called customer at 3 PM, no answer")
- Event filtering by type, actor, date range
- Reconnect and replay — if connection drops, missed events are replayed on reconnect
- Pagination for historical events
- Exportable as a case chronology report

### 2. Why We Implemented It

The timeline solves the "what happened and when?" question that every investigation eventually faces. A regulator asks: "Walk me through the investigation from start to finish." Without a timeline, an analyst has to reconstruct the history from memory, email timestamps, and document dates. With the timeline, the complete investigation chronology is already assembled.

Real-time updates matter because investigations are collaborative. When analyst A uploads a document, analyst B (who is also working on the case) should see it immediately — not at the next page refresh. The live feed turns what was a sequential, single-person process into a collaborative workspace.

### 3. How Is It Useful for the Client

**For the analyst:**
- Can see at a glance what has happened on a case since they last looked — no "catching up" conversation needed
- Real-time view means they see their colleague's work as it happens

**For the manager:**
- Timeline is the most powerful supervision tool — see exactly what each analyst did, in what order, and when

**For the compliance officer:**
- The timeline becomes the formal investigation chronology for the regulator
- Every event is sourced (system-generated vs. user-created) so fabricated entries are distinguishable

**For the regulator:**
- A printed or exported timeline is a complete, verified chain of custody for the investigation
- Timestamps are server-side — not editable by the analyst

### 4. How to Use It

**Automatic events (no action required):**
The timeline automatically records: case created, case assigned, status changed, document uploaded, note added, task created, task completed, STR filed, AI analysis run, risk score updated.

**Manual events:**
1. Open a case → Timeline tab → Add Event
2. Select event type (phone call, email, site visit, observation)
3. Enter description and occurrence time
4. Event appears in the timeline immediately for all viewers

**Real-time viewing:**
1. Open a case's timeline
2. Leave it open — new events appear automatically without refreshing
3. If you disconnect and reconnect, missed events are replayed from your last seen point

**Filtering:**
- Filter by event type (document events, task events, status changes)
- Filter by actor (show only events by analyst X)
- Filter by date range (show only events in the last 7 days)

### 5. Is the Implementation Standard Equal to or Greater Than Market

| Aspect | OpsOwl | Industry Standard |
|--------|--------|-------------------|
| Real-time updates | SSE, native | Polling every 30–60s in most tools |
| Automatic event capture | All system events | Manual entry required in many tools |
| Reconnect and replay | Yes — missed events replayed | Usually lost on disconnect |
| Manual event entry with backdating guard | Yes | Variable |
| Export as chronology report | Yes | Rare |

**Honest rating: 9/10** — The SSE implementation with replay-on-reconnect is genuinely ahead of market. Most investigation tools either require page refresh or use inefficient polling. Real-time collaboration on a case timeline is a meaningful UX differentiator.

### 6. Why Will the Client Buy It

The timeline eliminates the most painful part of a regulator audit: reconstructing investigation chronology. When an RBI officer walks in and asks "show me everything that happened on Case OWL-0432," the analyst clicks Timeline and a complete, verified, exportable chronology is available in 5 seconds. No reconstruction, no gaps, no "I think it was around the 15th."

---

## Feature 6 — Communications (COMMS)

### 1. What Is It

Communications is the structured log of all external and internal communications related to a case. Every phone call, email sent, letter received, or meeting held in the context of an investigation is documented here with full details.

Key capabilities:
- Log communication by type (email, phone, letter, in-person meeting, system-generated)
- Record direction (inbound / outbound)
- Store full content or summary
- Attach related documents
- Track by channel, date, and participant
- Export as a communication audit trail

### 2. Why We Implemented It

When a financial institution investigates suspicious activity, communications with the subject are a critical part of the investigation record. If the institution contacts the customer for an explanation and the customer provides one, that communication and its content must be documented. If the customer never responded, that must also be documented.

Without a structured COMMS module, these communications live in individual email inboxes, phone call notes, and personal diaries — completely unauditable and impossible to produce during a regulator review.

### 3. How Is It Useful for the Client

- **Legal defensibility:** If a customer disputes an investigation outcome, the communications log proves exactly what was said, when, and by whom
- **Investigation completeness:** FIU-IND guidelines require that the institution attempt to obtain customer explanation before filing certain reports
- **Handover clarity:** When a case changes hands, all communications history is visible to the new analyst instantly

### 4. How to Use It

1. Open a case → Comms tab → Log Communication
2. Select: type (email/phone/letter), direction (inbound/outbound), date and time
3. Enter participant details and communication content/summary
4. Attach related documents (email attachment, letter scan)
5. Save — communication is permanently logged in the case record

**Honest rating: 7/10** — Functional and auditable. Missing: direct email integration (send emails from within the platform), read receipts, customer portal. These are Phase 2 capabilities.

---

## Feature 7 — SLA Management

### 1. What Is It

SLA (Service Level Agreement) Management is the deadline enforcement engine of OpsOwl. Every case has an SLA — a maximum time from case opening to resolution. The system tracks this deadline, shows countdowns, escalates as deadlines approach, and reports on SLA performance over time.

SLA policies are configurable:
- Different durations per priority (Critical = 24 hours, High = 72 hours, Medium = 7 days, Low = 14 days)
- Different policies per case type
- Warning thresholds (e.g., alert at 75% consumed)
- Breach definition and breach escalation rules

### 2. Why We Implemented It

In financial services compliance, deadlines are regulatory requirements — not business preferences. RBI mandates STR filing within 7 days of identification. Missing that deadline is a violation that attracts penalties and can trigger supervisory action.

Without automated SLA tracking, deadline management depends entirely on individual analyst discipline and calendar reminders. At scale — 200+ active cases — this breaks down. Cases slip through. Deadlines are missed. Penalties follow.

SLA management automates what used to be a manual, anxiety-driven, error-prone process.

### 3. How Is It Useful for the Client

**For the analyst:**
- Every case clearly shows: SLA deadline, time remaining, percentage consumed
- Color-coded urgency: green (on track) → yellow (warning) → red (breached)

**For the manager:**
- Dashboard "Attention Required" panel shows all cases approaching or past SLA
- Can see SLA performance by analyst — who is consistently on time vs. consistently late

**For the CCO:**
- Monthly SLA compliance report: what percentage of cases were resolved within SLA
- Evidence to show regulators that the institution takes response time seriously

### 4. How to Use It

**SLA Policy Setup (one-time by admin):**
1. Settings → SLA Policies → New Policy
2. Set name, priority mapping, duration, and warning threshold
3. Assign policy to case types

**Day-to-day:**
- SLA countdown is visible on every case card and case detail page
- When 75% of SLA is consumed: case is highlighted yellow, manager receives notification
- When SLA breaches: case turns red, manager receives alert, case moves to "Attention Required"

**Reporting:**
- Analytics → SLA Performance: view breach rate by time period, analyst, case type

**Honest rating: 8/10** — The SLA engine is solid. The configuration is flexible. The visual countdown is clear. Missing: per-client SLA customization from the UI (currently configuration requires admin access), automatic pause when case is in `pending_review` waiting for manager.

---

## Feature 8 — STR / SAR Filing

### 1. What Is It

STR/SAR (Suspicious Transaction Report / Suspicious Activity Report) filing is the regulatory reporting feature — the endpoint of every confirmed investigation. When an investigation concludes that suspicious activity occurred, the institution is legally required to file a report with FIU-IND. OpsOwl makes this happen from within the case, without switching to any external system.

Key capabilities:
- Create STR/SAR record linked directly to the case
- Structured form with all required fields (transaction details, subject details, description)
- Status tracking (draft, submitted, acknowledged)
- Immutable once submitted
- Linked to case timeline (STR filed event auto-logged)
- Export in standard format

### 2. Why We Implemented It

STR filing is the hardest compliance task for most operations teams. It requires: compiling investigation findings, structuring them in a specific format, getting management sign-off, and submitting to FIU-IND. The current process at most institutions:
- Analyst writes findings in Word
- Manager reviews via email
- Compliance officer fills in the FIU-IND form manually
- Filed by email or the FIU-IND portal

This takes 2–3 days and happens entirely outside any investigation tracking system. The result: no traceability between the investigation and the report, no record of when the report was submitted, and no evidence of management approval.

OpsOwl collapses all of that into a single workflow inside the case.

### 3. How Is It Useful for the Client

- **Speed:** All investigation context is already in the case. The STR form pre-fills from case data.
- **Auditability:** The STR is permanently linked to the investigation that generated it — the chain of evidence is complete.
- **Management approval:** The pending_review status forces manager review before the STR is finalized.
- **No missed filings:** The SLA system ensures the 7-day window is tracked from case creation.

### 4. How to Use It

1. Open a confirmed suspicious case → STR/SAR tab → New STR
2. System pre-fills case details, transaction information, subject details from existing case data
3. Analyst completes the STR narrative (description of suspicious activity)
4. Save as draft → change case status to `pending_review`
5. Manager reviews → approves → STR status moves to `submitted`
6. Timeline records: "STR submitted at [timestamp] by [manager name]"

**Honest rating: 7/10** — The workflow is correct and auditable. The form structure follows FIU-IND requirements. Not yet verified against the exact goAML XML schema required for direct electronic submission — this is a known gap that needs to be addressed before enterprise sales to larger banks.

---

## Feature 9 — AI Risk Scoring & Insights

### 1. What Is It

AI Risk Scoring is the machine intelligence layer that analyses every investigation and returns an objective, quantified risk assessment. It does not replace human judgment — it augments it. When an analyst opens a case, they can trigger an AI analysis that examines all available case information and produces:

- **Risk Score:** 0–100 numerical rating (0 = no risk, 100 = highest risk)
- **Risk Level:** Low / Medium / High / Critical (derived from score thresholds)
- **Risk Factors:** Specific reasons driving the risk score ("unusual transfer pattern", "multiple dormant accounts activated")
- **Entities Detected:** People, companies, and accounts mentioned in documents and notes
- **Anomalies:** Unusual patterns detected in the case data
- **Recommended Actions:** Next steps the analyst should consider
- **Model Used and Confidence:** Transparency on which AI model and how confident it was

The analysis runs asynchronously — the analyst triggers it, returns to their work, and the result appears when ready. Previous analyses are stored — the full history is preserved.

### 2. Why We Implemented It

Two problems drove this feature:

**Problem 1 — Volume:** Compliance teams are overwhelmed. A team of 10 analysts handling 300 active cases cannot give equal attention to every case. They need to know which cases are genuinely high-risk and deserve more time. AI risk scoring provides a triage signal — high-risk cases float to the top.

**Problem 2 — Consistency:** Human analysts have different risk tolerances, different experience levels, and bad days. One analyst rates a case as medium risk; another rates identical facts as high. AI provides a consistent, repeatable baseline that all human decisions can be compared against.

### 3. How Is It Useful for the Client

**For the analyst:**
- Instant second opinion on every case — "Am I missing something?"
- AI-suggested next steps reduce the anxiety of junior analysts who don't know what to do
- The confidence score tells them how much to trust the AI assessment

**For the manager:**
- Risk-sorted case queue — work the high-risk cases first
- When an analyst closes a high-risk case without taking the AI's recommended actions, that's now visible and auditable

**For the CCO:**
- Can tell regulators: "We use AI-assisted risk assessment on every investigation" — a credibility statement
- AI risk score history shows the evolution of understanding on each case

### 4. How to Use It

**Triggering an analysis:**
1. Open a case → AI Insights tab → Run Analysis
2. System shows "Analysis queued" and returns immediately (non-blocking)
3. Analysis runs in background (30–90 seconds typically)
4. Risk score appears on the case header when complete

**Reading the results:**
- Risk score displayed as a number (e.g., 72.5) and level (Medium — amber badge)
- Risk factors list the specific signals driving the score
- Anomalies section highlights unusual patterns
- Recommended Actions gives a prioritized action list
- Model Used + Confidence helps interpret reliability of the assessment

**Re-running analysis:**
- As new documents and notes are added, the AI can be re-run to see if the risk assessment changes
- All historical analyses are stored — analysts can compare risk score evolution

### 5. Is the Implementation Standard Equal to or Greater Than Market

| Aspect | OpsOwl | Industry Standard |
|--------|--------|-------------------|
| Risk score in case interface | Native | Separate module, separate login |
| Async non-blocking analysis | Yes | Often synchronous (blocks UI) |
| Analysis history preserved | Yes | Usually overwritten |
| Recommended actions | Yes | Rare in mid-market tools |
| Entity extraction | Yes | Enterprise add-on |
| Model transparency | Yes (model + confidence) | Black box in most tools |

**Honest rating: 8/10** — The integration and UX are ahead of market. The async pattern with history preservation is well-designed. The gap: AI quality depends on the LLM being used — with Claude or GPT-4o, the results are excellent; the quality delta between providers is significant and client-visible.

### 6. Why Will the Client Buy It

AI is the primary differentiator in the sales conversation. Every compliance head has heard "you should be using AI" from their board. OpsOwl is the first tool in their price range that makes that real — not a marketing claim, but an actual risk score with evidence and recommended actions on every case. For an analyst handling 30 cases simultaneously, the AI triage capability alone justifies the annual license cost.

---

## Feature 10 — AI Copilot (Chat Assistant)

### 1. What Is It

The AI Copilot is a conversational AI assistant embedded directly in the case detail. Think of it as an expert analyst colleague who has read every document, note, timeline event, and AI insight on the case — and can answer questions, synthesize information, and suggest next steps in natural language.

Key capabilities:
- Conversational Q&A about the specific case
- Document content queries ("What does the bank statement say about transactions in April?")
- Synthesis ("Summarize the key risk factors in this investigation")
- Investigation guidance ("What should I check next?")
- Comparison ("How does this pattern compare to typical money laundering typologies?")
- Full conversation history preserved in the case

### 2. Why We Implemented It

Risk scoring gives a number. The Copilot gives a conversation. Analysts — especially junior ones — don't just need a score; they need guidance, explanation, and the ability to ask follow-up questions. The Copilot bridges the gap between AI output and human understanding.

It also accelerates knowledge: a junior analyst can ask the Copilot "what's unusual about this?" and get an explanation they would otherwise have to ask a senior colleague for — freeing up senior analyst time.

### 3. How Is It Useful for the Client

- **For junior analysts:** A 24/7 expert guide that explains what they're looking at and what to do next
- **For investigations that are stuck:** "What haven't we checked yet?" generates new investigation directions
- **For narrative writing:** "Help me write the STR narrative based on this case" — AI generates a draft that the analyst edits and submits

### 4. How to Use It

1. Open a case → AI Copilot button (top right of case header)
2. Side panel opens with chat interface
3. Type any question about the case in natural language
4. AI responds based on all case context (documents, notes, timeline, risk assessment)
5. Conversation history is saved to the case — other analysts can see what questions were asked

**Example queries:**
- "What are the red flags in this case?"
- "What documents are we still missing?"
- "Summarize the investigation for a management briefing"
- "Draft a suspicious activity narrative for the STR"

**Honest rating: 8.5/10** — When powered by Claude or GPT-4o, the quality is genuinely impressive. The integration into the case context (all documents, notes, timeline events accessible to the AI) is architecturally well-done. Gap: response quality is highly dependent on the LLM chosen, and the client needs to bring their own API key.

---

## Feature 11 — Role-Based Access Control (RBAC)

### 1. What Is It

RBAC defines exactly what each user can see and do within OpsOwl. Every action in the system is permission-controlled by role. There are five roles:

| Role | Who Uses It | What They Can Do |
|------|-------------|-----------------|
| **Super Admin** | System owner | Everything — full system access |
| **Admin** | IT / compliance head | User management, system config, all cases |
| **Manager** | Team lead / supervisor | Approve/close cases, reassign, view all team cases |
| **Analyst** | Investigation officer | Create, investigate, and progress their assigned cases |
| **Viewer** | Auditor / regulator | Read-only access to all cases |

Every API endpoint, every UI button, every action is controlled by role — analysts cannot approve their own cases, viewers cannot write anything.

### 2. Why We Implemented It

Compliance environments have strict separation of duties requirements. An analyst should not be able to close their own case without a manager approving it — that's a segregation of duties control. A viewer (external auditor, regulator) should be able to see everything but change nothing. Without RBAC, any user can do anything, which is both a security failure and a regulatory failure.

### 3. How Is It Useful for the Client

- **Compliance:** Proves to regulators that the 4-eyes principle is enforced at the system level, not just by policy
- **Security:** Limits blast radius if any account is compromised — a compromised analyst account cannot close cases or access admin functions
- **Audit convenience:** Create a Viewer account for the RBI examiner — they can see everything without any risk to the system

**Honest rating: 8/10** — Five-role model covers all common compliance team structures. Gap: no custom roles (some clients want "Senior Analyst" with slightly elevated permissions that don't need full Manager access).

---

## Feature 12 — Authentication (JWT + LDAP + Sessions)

### 1. What Is It

Authentication is how users prove who they are before accessing OpsOwl. The system supports:

- **Local authentication:** Username/password with enterprise password policy
- **LDAP/Active Directory:** Integrate with the company's existing user directory
- **JWT tokens:** RS256-signed tokens with configurable expiry and 30-second clock leeway
- **Session management:** Maximum concurrent sessions per user (default: 5)
- **Account lockout:** Lock after N failed attempts (configurable)
- **Password policy:** Minimum length, uppercase, numbers, history, expiry (all configurable)
- **Refresh tokens:** Secure token rotation for long sessions

### 2. Why We Implemented It

Security at the front door is non-negotiable. Password policy enforcement, account lockout, and session limits are all mandated by ISO 27001 and commonly required by RBI IT security guidelines. LDAP integration matters because most regulated institutions already have an Active Directory — they don't want to manage two user directories.

### 3. How Is It Useful for the Client

- **IT simplicity:** LDAP integration means users log in with their existing corporate credentials — no new password to remember, no separate provisioning process
- **Security policy enforcement:** Password requirements, expiry, and lockout are enforced by the system, not dependent on user discipline
- **Audit evidence:** Every login and logout is logged — who was in the system and when

**Honest rating: 7.5/10** — Solid and secure. The main gap is no SAML/OIDC support, which is the enterprise standard for SSO. This is the most commonly requested feature in enterprise procurement conversations.

---

## Feature 13 — Audit Chain (Tamper-Evident Log)

### 1. What Is It

The Audit Chain is OpsOwl's most architecturally sophisticated security feature. Every single action in the system — case created, document uploaded, status changed, user logged in, note written — is recorded in an append-only log where each entry contains a cryptographic hash of the previous entry. This creates a chain: if anyone modifies or deletes any audit record, the chain breaks and the tampering is immediately detectable.

This is similar to how blockchain works — but it does not require a blockchain. It uses SHA-256 hashing on a PostgreSQL table, which is performant, simple, and verifiable.

### 2. Why We Implemented It

A compliance system is only as trustworthy as its audit log. If an analyst can delete or modify audit records to cover their tracks, the entire system is compromised. Traditional "append-only" logs are not enough — a database administrator can delete rows from an append-only table without breaking any technical constraint.

The hash chain makes deletion detectable. The chain integrity can be verified at any time by recomputing the hashes. Any gap or mismatch means the chain was broken and tampering occurred.

This is specifically relevant for insider threat scenarios: a corrupt analyst or rogue admin cannot cover their tracks without the tampering being mathematically provable.

### 3. How Is It Useful for the Client

- **Regulatory defensibility:** The audit chain is verifiable evidence that the investigation record has not been altered
- **Insider threat protection:** Management can verify chain integrity at any time to detect unauthorized alterations
- **Legal standing:** A chain-verified audit trail has significantly stronger legal standing than a simple log file

### 4. How to Use It

- Audit chain is automatic — no action required from users
- Admin can trigger chain verification: Settings → Audit → Verify Chain Integrity
- If chain is valid: green "Chain Intact" status with last verified timestamp
- If chain is broken: alert with the exact record where tampering was detected

**Honest rating: 9/10** — This is genuinely ahead of the market. Most compliance tools use append-only logs. A cryptographic hash chain is a higher standard that most enterprise tools don't implement. It is a strong differentiator in security-conscious enterprise sales.

---

## Feature 14 — Document Encryption & Virus Scanning

*(Covered in depth under Feature 3 — Document Management)*

**Quick summary for completeness:**
- AES-256 encryption with per-key management
- ClamAV virus scan on every upload — rejects infected files before storage
- Encryption keys stored separately from document data
- Transparent decryption on authorised download
- All access logged in audit chain

**Honest rating: 8.5/10**

---

## Feature 15 — API Keys

### 1. What Is It

API Keys allow external systems — transaction monitoring systems, core banking software, other internal tools — to connect to OpsOwl programmatically, without a human user logging in. An API key is a long, random secret that authenticates API calls. Each key can be:
- Named (e.g., "TMS Integration Key", "Reporting Script Key")
- Scoped to specific actions
- Revoked instantly if compromised
- Audited — every API call made with a key is logged

### 2. Why We Implemented It

Most compliance operations are not entirely manual. A transaction monitoring system detects a suspicious pattern and needs to automatically create a case in OpsOwl. A reporting script needs to pull today's open cases for an executive dashboard. An internal fraud scoring model needs to attach a risk flag to a case. None of these can use human login credentials — they need machine-to-machine authentication.

API keys enable the integrations that make OpsOwl part of the institution's wider technology ecosystem rather than an isolated tool.

### 3. How Is It Useful for the Client

- Automatic case creation from TMS alerts — no manual input required
- Integration with existing BI tools to pull case metrics
- Automation of routine tasks (status updates, report generation)
- Secure — keys can be rotated or revoked without changing passwords

**Honest rating: 7.5/10** — Functional and secure. Gap: API keys currently have full analyst-level access — scoping (read-only keys, write-only keys) would improve the security model and is a common enterprise request.

---

## Feature 16 — Search

### 1. What Is It

Search is the global, full-text search across all content in OpsOwl — cases, notes, documents, timeline events, and tasks. Analysts can find anything across their entire case history in seconds.

Key capabilities:
- Instant search across cases by number, title, subject name
- Full-text search through note content
- Document content search (after OCR extraction)
- Filtered search (search within a date range, status, priority)
- Relevance ranking — most relevant results first
- Search within a case (find specific content within one case's history)

### 2. Why We Implemented It

At scale — hundreds of open cases and thousands of historical cases — an analyst who remembers "there was a case last year involving this company" but doesn't know the case number needs a way to find it. Manual browsing through case lists is not an option at volume. Search is the navigation system for institutional memory.

### 3. How Is It Useful for the Client

- **Finding precedent:** "Has this company appeared in investigations before?" — search surfaces all historical cases
- **Cross-case intelligence:** Analyst notices a pattern — search confirms whether similar patterns have appeared in other cases
- **Speed:** Finding the right case in 2 seconds vs. 2 minutes per investigation adds up significantly across a team

**Honest rating: 7/10** — The Postgres full-text search (`tsvector`) implementation is solid for the mid-market. At 50,000+ cases with large documents, Elasticsearch would provide better performance and more sophisticated relevance. This is the known Phase 2 upgrade path.

---

## Feature 17 — Analytics & Reporting

### 1. What Is It

Analytics provides management-level visibility into investigation operations — not the details of individual cases, but the patterns, trends, and performance metrics across all investigations.

Key dashboards:
- Case volume over time (open, closed, escalated per week/month)
- SLA performance (breach rate, average resolution time)
- Analyst workload distribution (cases per analyst)
- Risk distribution (how many cases at each risk level)
- Priority breakdown (critical vs. high vs. medium vs. low)
- STR filing rate (what percentage of cases resulted in STR)

### 2. Why We Implemented It

A compliance head managing 10 analysts and 300 cases cannot stay on top of operations by reading individual cases. They need aggregated intelligence: "Are we meeting SLA targets this month? Is analyst X overloaded? Are we seeing more high-risk cases this quarter?" Analytics answers these questions without requiring anyone to build Excel pivot tables.

### 3. How Is It Useful for the Client

- **Board reporting:** Monthly compliance metrics in chart form — ready to present
- **Regulatory evidence:** "Our SLA compliance rate was 94% last quarter" is now a measurable, provable claim
- **Resource planning:** Data shows whether the team is understaffed relative to case volume

**Honest rating: 7/10** — Good foundational analytics. The gap is customizability — the dashboard shows a fixed set of metrics. Enterprise clients want to add their own KPIs. Configurable analytics is a Phase 2 item.

---

## Feature 18 — Scheduled Reports

### 1. What Is It

Scheduled Reports automatically generate and deliver reports on a recurring schedule — daily, weekly, or monthly — without anyone having to manually run them. Reports are delivered by email to configured recipients.

Key capabilities:
- Schedule by frequency (daily, weekly, monthly)
- Report types (open cases, SLA breaches, high-risk cases, analyst performance)
- Email delivery to configured recipients
- PDF or CSV format
- Time-zone aware delivery

### 2. Why We Implemented It

Managers and executives want regular updates but do not want to log in every day to generate reports manually. A Monday morning "this week's open cases" email that arrives automatically — before the team briefing — is a standard expectation in enterprise operations tools.

### 3. How Is It Useful for the Client

- **Executive reporting:** CEO and board automatically receive weekly compliance snapshots
- **Manager efficiency:** No manual report generation — it just arrives
- **Accountability:** Regular reports mean nothing is forgotten — open cases are visible to leadership continuously

**Honest rating: 7.5/10** — Functional and useful. Gap: no report builder (clients want to define their own column layout and filters). This is on the roadmap.

---

## Feature 19 — Bulk Operations

### 1. What Is It

Bulk Operations allow users to perform actions on many cases simultaneously — instead of opening each case one by one. Supported bulk actions:
- Assign multiple cases to an analyst
- Change status on multiple cases
- Add tags to multiple cases
- Export multiple cases to CSV
- Close multiple resolved cases

### 2. Why We Implemented It

When a bank's compliance team returns from a long weekend and finds 50 new cases in the queue, they need to triage and assign all 50 efficiently. When a regulation changes and all cases in a certain category need to be re-tagged, doing it one by one is impractical.

Bulk operations turn a 2-hour manual process into a 30-second operation.

### 3. How Is It Useful for the Client

- **Triage efficiency:** Assign batches of incoming cases quickly during high-volume periods
- **Category management:** Re-tag, re-prioritize, or reassign an entire class of cases at once
- **Compliance cleanup:** Bulk close a category of cases after a regulatory decision changes the classification

**Honest rating: 7.5/10** — The implementation covers the most common bulk needs. Gap: bulk operations currently require selecting cases manually; a "bulk-update all cases matching filter X" pattern would be more powerful.

---

## Feature 20 — Case Templates

### 1. What Is It

Case Templates are pre-configured case blueprints that automatically set up a new case with the correct tasks, document requirements, SLA policy, and initial notes for a specific type of investigation. Instead of building each investigation setup from scratch, an analyst selects a template and the case is pre-populated.

Example templates:
- **AML Transaction Review:** 6 standard tasks, 3 required documents, 7-day SLA
- **KYC Periodic Review:** 4 standard tasks, 2 required documents, 30-day SLA
- **Credit Fraud Investigation:** 8 standard tasks, 5 required documents, 14-day SLA
- **Account Takeover:** 5 standard tasks, 4 required documents, 3-day SLA (high urgency)

### 2. Why We Implemented It

Every category of investigation has a standard methodology — a set of steps that should always be performed. Without templates, analysts either follow the methodology from memory (and miss steps) or have to manually create the same 8 tasks on every fraud case. Templates encode institutional best practices into the system.

They also solve the onboarding problem: a new analyst using a template cannot miss required steps because the template created the tasks automatically.

### 3. How Is It Useful for the Client

- **Consistency:** Every AML case follows the same investigation methodology — regardless of who the analyst is
- **Compliance:** Required steps are pre-created as tasks — analysts cannot close a case without completing them
- **Onboarding:** New analysts are productive on day 1 — the template shows them exactly what to do

**Honest rating: 7/10** — Templates work well. Gap: templates cannot currently be shared between teams and don't support conditional logic (e.g., "if case amount > ₹10 Lakhs, add additional review task"). These are Phase 2 capabilities.

---

## Feature 21 — Notifications

### 1. What Is It

Notifications alert users to events that require their attention — in-app and by email. Every meaningful event can generate a notification: case assigned, task deadline approaching, SLA breach, manager approval requested, mention in a note, AI analysis complete.

Key notification types:
- Case assigned to you
- Task assigned to you
- SLA approaching (75% consumed)
- SLA breached
- Case escalated
- Manager approval required
- AI analysis complete
- New note added to your case
- New document uploaded to your case

### 2. Why We Implemented It

An investigation platform is only useful if the right people know when they need to take action. Without notifications, analysts have to actively check the platform to see if anything changed. At a team of 30 analysts, nobody is refreshing OpsOwl every 5 minutes — they are doing other work. Notifications ensure that assignments and deadlines are acted on promptly.

### 3. How Is It Useful for the Client

- **Responsiveness:** Analysts act on assigned cases quickly because they know immediately
- **Deadline prevention:** SLA warning notifications catch impending breaches before they happen
- **Management visibility:** Managers are notified of escalations and pending approvals in real time

**Honest rating: 7/10** — In-app notifications are solid. Email notifications for key events are available via SMTP. Gap: no integration with Slack, Microsoft Teams, or mobile push — which is where most operations teams actually spend their time. These are the most-requested notification channels.

---

## Feature 22 — Inbound Event Log (Webhook Receiver)

### 1. What Is It

The Inbound Event Log receives and logs events sent from external systems — transaction monitoring systems, fraud detection tools, core banking systems — via HTTP webhooks. When an external system detects a suspicious event and sends it to OpsOwl, the event is received, validated, logged, and can automatically trigger case creation.

Key capabilities:
- Configurable webhook endpoints per source system
- Signature verification (HMAC) to validate authenticity
- Event logging with full payload storage
- Auto-case-creation rules (configurable per source)
- Retry visibility — see which events were received and when
- Full event history per source

### 2. Why We Implemented It

OpsOwl is not an island. It lives within a larger technology ecosystem — transaction monitoring systems, core banking software, fraud scoring engines. These systems generate the alerts that become investigations. Manually copying alerts from one system into OpsOwl is error-prone, slow, and unscalable.

Webhooks automate the intake: the TMS sees suspicious activity → sends webhook to OpsOwl → case is created automatically → analyst is notified. Zero manual steps, zero data re-entry, zero missed alerts.

### 3. How Is It Useful for the Client

- **Speed:** Alert-to-case creation drops from 24–48 hours (manual) to seconds (automated)
- **Completeness:** No alert is missed — the webhook receives everything the TMS sends
- **Auditability:** Full log of every inbound event — proves that the institution received and acted on every alert

**Honest rating: 8/10** — The webhook receiver is well-built: HMAC validation, full payload logging, retry visibility. Gap: no pre-built parsers for common TMS vendors (NICE Actimize, Oracle FCCM) — clients have to write the mapping themselves. Pre-built connectors would significantly accelerate deployment.

---

## Feature 23 — Outbound Webhooks

### 1. What Is It

Outbound Webhooks push events from OpsOwl to external systems — automatically, in real time — when specific things happen. When a case is created, escalated, closed, or an STR is filed, OpsOwl can notify any external system via HTTP POST with the event details.

Key capabilities:
- Configurable events per webhook (case created, case closed, STR filed, etc.)
- Configurable target URL and authentication headers
- Retry logic on delivery failure (with backoff)
- Delivery log — see every outbound event, its status, and any errors

### 2. Why We Implemented It

Compliance data doesn't live only in OpsOwl. Institutions have data warehouses, BI tools, compliance portals, and management reporting systems that need to be updated when OpsOwl data changes. Rather than requiring all of those systems to poll OpsOwl's API, outbound webhooks push updates in real time.

A common use case: when a high-risk case is escalated in OpsOwl, a webhook fires to the management reporting system, which updates the board dashboard. No manual intervention. No polling. No delay.

### 3. How Is It Useful for the Client

- **Integration:** Keep downstream systems in sync with investigation data in real time
- **Automation:** Trigger external workflows (credit risk review, customer notification hold) based on case events
- **Monitoring:** Operations dashboards outside OpsOwl stay current without polling

**Honest rating: 7.5/10** — Solid outbound webhook implementation with retry logic and delivery logging. Gap: no webhook testing tool within the UI (clients have to use external tools like Postman to test their webhook configuration).

---

## Feature 24 — SIEM Integration

### 1. What Is It

SIEM (Security Information and Event Management) Integration sends OpsOwl's security audit events — logins, access denials, configuration changes, document access — to the institution's central security monitoring platform in real time. OpsOwl supports two industry-standard formats:

- **JSON:** Structured events for modern SIEM platforms (Splunk, Elastic SIEM, Microsoft Sentinel)
- **CEF (Common Event Format):** The ArcSight standard, used by legacy enterprise SIEM deployments

### 2. Why We Implemented It

Large institutions have a Security Operations Center (SOC) that monitors all their systems from a central SIEM. They require every system in their environment to forward security events to this central monitoring platform. Without SIEM integration, OpsOwl is a blind spot in their security monitoring.

With SIEM integration, the SOC sees: failed logins to OpsOwl, unusual after-hours document access, bulk data exports, admin configuration changes — all in real time, alongside events from every other system. This enables correlation: if an analyst is accessing OpsOwl at 3 AM and also accessing the core banking system, the SOC can see both events together.

### 3. How Is It Useful for the Client

- **Security compliance:** ISO 27001 and RBI IT security guidelines require centralized security event monitoring
- **Insider threat detection:** Unusual OpsOwl access patterns are visible to the SOC
- **Incident response:** When a security incident occurs, complete OpsOwl event history is available in the SIEM

**Honest rating: 8/10** — Both JSON and CEF formats cover the full range of SIEM platforms. This is a rare feature at this price point — most mid-market tools don't offer SIEM integration. It is a strong signal of enterprise readiness to a CISO.

---

## Feature 25 — Monitoring (Prometheus + Grafana)

### 1. What Is It

The monitoring stack provides complete operational visibility into OpsOwl's health and performance. Every service — the Go API, the AI service, the database, MinIO storage — is instrumented and reports metrics to Prometheus. Grafana visualizes those metrics as real-time dashboards. Alertmanager sends alerts when something goes wrong.

Key metrics tracked:
- API response times (p50, p95, p99)
- Error rates per endpoint
- Database connection pool utilization
- Active users and active cases
- AI analysis queue depth and processing time
- Disk usage for document storage
- Memory and CPU utilization per service

Alerts configured:
- API error rate > 1% → page on-call
- Database connection pool > 80% utilization → warning
- API response time p95 > 2 seconds → warning
- Any service down → critical alert via email

### 2. Why We Implemented It

An operations platform handling compliance investigations must be reliable. A compliance team that cannot access OpsOwl because of a server issue is exposed: cases are stalled, deadlines are missed, and the regulator audit that was scheduled for that afternoon has no evidence to show. We built full observability so that problems are detected and resolved before they affect the compliance team.

The monitoring stack also satisfies a standard IT requirement: any system on a regulated institution's infrastructure must be monitorable and alertable. IT departments require this before approving any new system.

### 3. How Is It Useful for the Client

- **Reliability:** Problems are detected before users notice them
- **IT approval:** The IT department sees a proper monitoring stack and approves deployment faster
- **Capacity planning:** Disk usage and CPU trends show when the server needs upgrading before it becomes a crisis
- **SLA compliance:** API response time metrics provide evidence that the platform meets performance SLAs

**Honest rating: 9/10** — The monitoring implementation is genuinely enterprise-grade: Prometheus + Grafana + Alertmanager with real alert rules. Most products at this price point ship with zero operational monitoring. This is a strong signal of technical maturity to the IT team reviewing the deployment.

---

---

## Feature 26 — Operations Dashboard

### 1. What Is It

The Operations Dashboard is the home screen every user sees when they log in — a live command center that gives a complete situational picture of the entire compliance operation without opening a single case.

Key panels on the dashboard:
- **Summary Stats:** Total open cases, cases in progress, SLA breaches today, high-priority cases, AI analyses run
- **Attention Required:** All cases that have breached SLA or are escalated — the most urgent items, always visible
- **My Queue:** The current user's top 5 open assigned cases with SLA countdown — the first thing an analyst sees every morning
- **Case Distribution:** Priority and status breakdown as visual charts
- **Recent Activity:** Latest case updates across the team
- **Quick Actions:** New case, search, view all cases — accessible in one click

The dashboard auto-refreshes every 15 seconds — SLA breaches appear in real time. When a new breach occurs, the Attention Required panel pulses to draw the user's eye.

### 2. Why We Implemented It

A compliance team's day starts with one question: "What needs my attention right now?" The dashboard answers that question in under 3 seconds. Without it, analysts and managers have to navigate to the cases list, apply filters, and interpret what they see — adding 5–10 minutes to every morning briefing.

For managers, the dashboard is even more critical: they supervise a team of 10–20 analysts and need a real-time view of where the workload stands, where SLAs are breaking, and where to intervene — all without interrupting their analysts.

### 3. How Is It Useful for the Client

**For the analyst:**
- "My Queue" shows exactly what they need to work on — no filtering, no decision fatigue
- If all 5 of their cases are green, they know they are on track

**For the manager:**
- Real-time SLA breach count — they can intervene before a regulatory penalty
- Case distribution charts show if the team is overloaded in a particular category

**For the executive (who may log in weekly):**
- Summary stats give a 10-second compliance health check
- Attention Required tells them if anything needs escalation today

### 4. How to Use It

1. Log in → Dashboard loads automatically as the home screen
2. Check the "Attention Required" panel first — anything there needs immediate action
3. Check "My Queue" — see personal open cases with SLA status
4. Summary stats give the operational picture at a glance
5. Charts show case volume trends over the last 7/30 days

**The dashboard is read-only** — it shows information, all actions require navigating to the relevant case.

### 5. Is the Implementation Standard Equal to or Greater Than Market

| Aspect | OpsOwl | Industry Standard |
|--------|--------|-------------------|
| Real-time SLA breach alerts | Yes — 15s refresh with pulse animation | Most tools: manual refresh |
| Personalized "My Queue" | Yes — analyst-specific view | Usually requires filter setup |
| Single-screen situational awareness | Yes | Often requires multiple screens |
| Role-aware content | Yes — analyst sees My Queue; manager sees team view | Variable |

**Honest rating: 8/10** — Clean, fast, and role-aware. The real-time SLA pulse is a small but impactful detail. Gap: not customizable — different teams want different default widgets.

### 6. Why Will the Client Buy It

The dashboard is often the first screen shown in a demo. When a CCO sees their entire compliance operation on one screen — with real-time SLA status, their team's queue, and the cases that need attention — the value is immediately intuitive. It replaces the daily morning WhatsApp message asking "who has what cases today?" with a live screen.

---

## Feature 27 — User Management

### 1. What Is It

User Management is the administrative control panel for managing all users on the OpsOwl platform. Admins and Super Admins can: create new user accounts, assign roles, reset passwords, deactivate accounts, and view user activity.

Key capabilities:
- Create users individually or in bulk (with CSV import)
- Assign and change roles (Viewer, Analyst, Manager, Admin, Super Admin)
- Reset passwords (generates a temporary password or sends reset email)
- Deactivate accounts without deleting data (deactivated user's cases remain intact)
- View last login and activity per user
- Set department / team association
- View all active sessions per user and revoke specific sessions

### 2. Why We Implemented It

People join teams, change roles, leave organizations, and get compromised credentials. User management is the control plane that keeps the system aligned with the real organizational structure. Without it, an analyst who resigned 3 months ago still has an active login — a real security and compliance risk.

Deactivation without deletion is critical: when a user is deactivated, their historical case work, notes, and timeline entries must remain intact for audit purposes. OpsOwl deactivates accounts without purging any investigation data they contributed.

### 3. How Is It Useful for the Client

- **IT control:** The IT or HR team can provision and deprovision users without needing technical database access
- **Separation of duties:** Only Admins can create users and assign roles — analysts cannot promote themselves
- **Off-boarding safety:** Deactivate departing analyst's account in 30 seconds — their cases remain assigned and visible to the manager
- **Session security:** If an account is suspected of compromise, all active sessions can be revoked immediately without changing the password

### 4. How to Use It

**Creating a user:**
1. Settings → Users → New User
2. Enter email, full name, role, and optional department
3. System sends welcome email with temporary password
4. User logs in and is prompted to change password

**Deactivating a user:**
1. Settings → Users → Find user → Deactivate
2. User immediately loses all access
3. All their cases remain in the system, assigned to them (manager can bulk-reassign)

**Revoking sessions:**
1. Settings → Users → Find user → Active Sessions
2. See all devices logged in
3. Click "Revoke All Sessions" for immediate logout across all devices

**Honest rating: 7.5/10** — Clean and functional. Gap: no SCIM provisioning (automated sync from Okta/Azure AD), no bulk user import from HR system, no department hierarchy.

---

## Feature 28 — My Tasks (Cross-Case View)

### 1. What Is It

My Tasks is a unified task view for each analyst — showing all their pending tasks across every case they are working on, in a single prioritized list. Instead of opening 8 different cases to find what needs to be done today, the analyst sees all their tasks in one place, ranked by urgency.

My Tasks shows:
- Task title and the case it belongs to
- Due date and overdue flag
- Priority (Critical, High, Medium, Low)
- Status (To Do, In Progress)
- Case number and title for context
- Direct link to open the specific case

Sorting defaults: overdue tasks first, then by due date, then by priority.

### 2. Why We Implemented It

An analyst working 15 simultaneous cases has 40–60 active tasks spread across those cases. Navigating to each case individually to check pending tasks is slow, error-prone, and causes missed deadlines. My Tasks gives the analyst a single prioritized action list every morning — the equivalent of a personal to-do board, automatically populated from all their assigned work.

The cross-case view also surfaces overdue tasks that might be buried inside a lower-priority case — preventing the situation where a task deadline is missed simply because the case wasn't opened that day.

### 3. How Is It Useful for the Client

- **Analyst productivity:** Start the day knowing exactly what to work on — highest priority, overdue first
- **Deadline prevention:** Overdue tasks from any case are immediately visible regardless of case priority
- **Manager oversight:** Managers can view any analyst's My Tasks to see their workload and identify overdue items
- **Focus:** Analysts working complex investigations can stay in their task list without losing track of cross-case obligations

### 4. How to Use It

1. Dashboard → My Tasks (or top navigation → Tasks)
2. Tasks are listed: overdue first (red), due today (amber), upcoming (green)
3. Click any task to open it in context
4. Change status directly from the list view — no need to open the full case

**Filters available:**
- All tasks / Overdue only / Due today / This week
- Filter by case, priority, or status

**Honest rating: 7.5/10** — Essential analyst productivity feature. The overdue-first sorting is correct default behavior. Gap: no calendar view of tasks (timeline view of deadlines is a common request), no task dependencies.

---

## Feature 29 — AI Logs

### 1. What Is It

AI Logs is a complete, per-case record of every AI interaction that occurred on that investigation. Every time the AI Copilot was queried, every time a risk analysis was run, every prompt sent to the AI, and every response received is stored and viewable.

AI Logs contain:
- Timestamp of each AI interaction
- Type (Copilot query / Risk analysis / Document extraction)
- The exact prompt or request sent
- The complete AI response
- The AI model used and the token count
- Whether the response was used (analyst read it) or discarded

### 2. Why We Implemented It

AI outputs are consequential in a compliance context. If an AI suggested "freeze account" as a recommended action, and the analyst acted on it, that AI recommendation is part of the investigation record. It must be preserved and auditable — regulators and courts may ask "on what basis did you take this action?"

AI Logs also protect the institution from the opposite scenario: if an analyst ignored an AI recommendation that proved to be correct, the log shows that the recommendation was made and documents the human override. This is essential for regulatory defensibility.

Beyond compliance, AI Logs enable quality review: managers can periodically review AI interactions to assess whether the AI is providing useful guidance and whether analysts are using it appropriately.

### 3. How Is It Useful for the Client

- **Audit trail:** Complete record of AI assistance provided to each investigation — regulators can see exactly what AI was used for
- **Accountability:** Proves whether an analyst saw the AI recommendation before making a decision
- **Quality control:** Management can review AI interaction quality and identify cases where guidance was ignored or misapplied
- **Transparency:** Clients can show regulators exactly how AI is used in their compliance process — not as a black box but as a documented, auditable tool

### 4. How to Use It

**Viewing AI logs on a case:**
1. Open a case → AI Logs tab
2. All AI interactions listed chronologically
3. Expand any entry to see full prompt and response
4. Model used and confidence score visible on each entry

**Reviewing as a manager:**
1. Navigate to the AI Logs tab on any case
2. Can filter by interaction type (Copilot / Risk Analysis / Document)
3. Useful for post-investigation review and quality assurance

**Honest rating: 8/10** — The complete prompt + response storage is more transparent than most AI tools, which only show outputs. Rare feature at this price point. Gap: no cross-case AI log summary or AI usage analytics (e.g., "how many AI queries this week?" at the team level).

---

## Feature 30 — CSV Import (Bulk Case Import)

### 1. What Is It

CSV Import allows institutions to bulk-load historical cases — or new cases from an external system — into OpsOwl by uploading a structured CSV file. Instead of creating cases one by one through the UI, hundreds of cases can be imported in a single operation.

Key capabilities:
- Template CSV download (shows exactly which columns are expected)
- Column mapping UI (map your CSV columns to OpsOwl case fields)
- Validation before import (flags errors before writing anything)
- Dry-run preview (shows what will be created before committing)
- Error report (rows that failed and why, downloadable)
- Progress tracking for large imports
- Imported cases are fully functional (can be assigned, worked, and closed normally)

### 2. Why We Implemented It

Every new customer has historical data. A mid-sized NBFC migrating from Excel to OpsOwl might have 500 cases from the last 12 months that need to be in the system — for regulatory continuity, for reference, and for the AI to learn from. Re-entering 500 cases manually is not feasible.

CSV import solves the data migration problem and the ongoing integration scenario: an institution whose transaction monitoring system exports a daily CSV of flagged transactions can automate case creation via a scheduled CSV import.

### 3. How Is It Useful for the Client

- **Migration:** Move existing investigation records into OpsOwl without losing history
- **Automated intake:** TMS exports a CSV every morning → automated import creates all new cases before analysts start their day
- **Backfill:** Load historical cases for AI analysis — the AI can identify patterns in historical investigations
- **Data continuity:** Regulators expect case history to go back 5 years — CSV import makes this achievable

### 4. How to Use It

1. Navigate to Cases → Import → Download Template CSV
2. Fill in template with your case data (title, description, status, priority, created date, assigned analyst)
3. Upload CSV
4. Map columns if your column names differ from the template
5. Review validation results — fix errors and re-upload if needed
6. Confirm import — system creates all cases
7. Download error report for any rows that failed

**Honest rating: 7/10** — The import capability is solid. Gap: currently imports case metadata only — historical documents, notes, and timeline events cannot be bulk-imported (they must be added manually after import). Full historical record migration is a complex Phase 2 item.

---

## Feature 31 — Case PDF Export

### 1. What Is It

Case PDF Export generates a complete, formatted PDF document containing the entire investigation record — case details, timeline, tasks, notes, document inventory, AI insights, STR filing status, and audit summary. The PDF is designed to be submitted to a regulator, presented in a board meeting, or stored as a permanent archive of the investigation.

What the PDF contains:
- Case header (case number, title, subject, status, assigned analyst, dates)
- Investigation summary (description, findings, conclusion)
- Full timeline (all events in chronological order)
- Task completion record (all tasks with status and completion date)
- Document inventory (list of all attached documents with metadata)
- Notes (all investigator notes with author and timestamp)
- AI risk assessment (risk score, risk level, key risk factors)
- STR/SAR reference (if filed)
- Audit summary (key actions, who did what, when)

### 2. Why We Implemented It

Investigations eventually need to leave the system — for regulatory submissions, internal audit reviews, legal proceedings, or archival. A digital system that cannot produce a self-contained, formatted document of its records is incomplete. The PDF export bridges the digital investigation record with the physical-world requirements of regulators, auditors, and courts.

The PDF is also the answer to: "I need to brief my MD on this case in 10 minutes." One click produces a complete, professional document.

### 3. How Is It Useful for the Client

- **Regulatory submissions:** Submit a complete investigation record to FIU-IND, RBI, or SEBI without manual document assembly
- **Legal proceedings:** A formatted PDF is admissible evidence; a database record is not
- **Board briefings:** Compliance head needs to present a case to the board — PDF is ready in seconds
- **Long-term archival:** Cases closed 7 years ago may need to be retrieved — PDF archives are format-independent and readable without the software

### 4. How to Use It

1. Open a case → Export → Export as PDF
2. Select what to include (all sections, or choose specific sections)
3. System generates PDF (10–30 seconds for large cases)
4. PDF downloads to browser
5. PDF is also saved to the case's document list for future reference

**Honest rating: 7.5/10** — The PDF export is complete and professional. Gap: PDF template is fixed — clients cannot customize the branding, header, or section layout without a code change. White-label PDF templates are a common enterprise request.

---

## Feature 32 — Data Retention Policies

### 1. What Is It

Data Retention Policies are automated rules that govern how long case data is kept, when it is archived, and when it is permanently deleted. The system automatically applies retention rules in the background, ensuring compliance with both regulatory minimum retention requirements and internal data governance policies.

Key capabilities:
- Configurable retention periods by case type and status
- Automatic archival of closed cases after a set period (data becomes read-only, searchable but not editable)
- Automatic deletion trigger after the maximum retention period (with admin approval required before execution)
- Retention freeze for cases under legal hold (prevents automatic deletion regardless of policy)
- Retention audit log (records when each policy action was applied to each case)
- Configurable retention for documents vs. case metadata (documents may have shorter retention than the case record itself)

### 2. Why We Implemented It

RBI requires financial institutions to retain records related to AML investigations for **at least 5 years** from the date of the transaction or termination of the business relationship. Keeping data forever is also not acceptable — growing data volume increases costs, increases data breach risk, and may violate privacy regulations. Data retention policies automate the balance between "keep long enough for compliance" and "delete when legally allowed."

Without automated retention management, institutions either keep everything forever (growing risk and cost) or manually purge data (error-prone and potentially non-compliant).

### 3. How Is It Useful for the Client

- **Regulatory compliance:** Automatically enforces the 5-year minimum retention required by RBI/FIU-IND without any manual process
- **Cost management:** Archived and deleted data reduces database size and storage costs over time
- **Privacy compliance:** Data that no longer needs to be retained is removed — reducing DPDP Act exposure
- **Legal hold:** Cases under litigation or regulatory investigation can be locked against deletion with a single toggle

### 4. How to Use It

**Setting up a policy:**
1. Settings → Data Retention → New Policy
2. Set: case status (closed), retention period (e.g., 7 years), action (archive at 5 years, delete at 7 years)
3. Assign policy to case types

**Legal hold:**
1. Open a case → Settings → Apply Legal Hold
2. Case is frozen — no retention policy will apply until the hold is released
3. Hold reason and date are logged in the audit chain

**Reviewing upcoming actions:**
1. Settings → Data Retention → Upcoming Actions
2. See all cases scheduled for archival or deletion in the next 90 days
3. Admin can review and approve deletion in bulk

**Honest rating: 7/10** — Policy engine is functional and covers the core regulatory requirements. Gap: no bulk legal hold application (if 200 cases relate to the same regulatory matter, each must be individually held — bulk hold is a gap), and the deletion approval workflow requires manual review of each batch.

---

## Feature 33 — Backup & Recovery

### 1. What Is It

Backup & Recovery is the infrastructure-level data protection layer that ensures OpsOwl's data can survive hardware failure, accidental deletion, ransomware, or any other disaster. It operates automatically in the background with no action required from users.

Three-layer backup architecture:

**Layer 1 — WAL Archiving (Continuous):**
PostgreSQL Write-Ahead Log segments are continuously copied to a dedicated archive volume. Maximum data loss: 5 minutes (archive_timeout configured to 300 seconds). Used for Point-In-Time Recovery (PITR) — restore the database to any specific second in history.

**Layer 2 — Daily pg_dump (Logical Backup):**
A complete logical dump of the entire database runs daily. The backup is compressed (gzip) and stored on the local volume. Last 7 days of backups are retained. Old backups are automatically pruned.

**Layer 3 — S3 Offsite Sync (Optional):**
If configured, daily backups are synced to an S3-compatible object store in a separate location (separate datacenter, separate cloud region). This protects against complete on-site disasters (fire, flood, facility loss).

**Recovery options:**
- Restore a specific day's backup: 15–30 minutes
- Point-in-time recovery to any moment in the last 30 days: 30–60 minutes
- Full system disaster recovery: 1–2 hours

### 2. Why We Implemented It

A compliance investigation database is mission-critical. It contains the complete record of every investigation, every document, every audit trail. Losing this data is not just an operational problem — it is a regulatory violation. RBI requires that records be preserved for 5 years. A database corruption with no backup plan means the institution cannot prove their compliance history.

We implemented a three-layer backup strategy because:
- WAL archiving handles continuous data protection (minimal data loss window)
- Daily pg_dump provides a clean, portable full backup that is simple to restore
- S3 offsite sync protects against physical facility disasters

### 3. How Is It Useful for the Client

- **Regulatory compliance:** Demonstrates to RBI auditors that data preservation infrastructure exists
- **Disaster recovery:** A ransomware attack or server failure does not destroy the compliance record — recovery is possible
- **Peace of mind:** The IT team and management know that even in the worst case, the data is safe
- **Insurance requirement:** Many cyber insurance policies require documented backup procedures — this satisfies that requirement

### 4. How to Use It

**Normal operation:** No action required. Backups run automatically.

**Verifying backups are running:**
1. Check Grafana monitoring dashboard → pg-backup panel
2. Last successful backup timestamp and backup size are visible
3. If backup has not run in 25+ hours → alert fires to the configured email

**Restoring a daily backup:**
```bash
# On the server where OpsOwl is deployed
gunzip -c /backups/opsowl-20260501-020000.sql.gz | \
  psql -h localhost -p 5433 -U $POSTGRES_USER $POSTGRES_DB
```

**Point-in-time recovery (PITR):**
1. Stop the PostgreSQL service
2. Restore the last base backup
3. Configure `recovery_target_time` to the desired point
4. Point PostgreSQL at the WAL archive
5. Start — database replays WAL segments to the target time

**S3 offsite sync verification:**
- Backup container logs show sync status per run
- Grafana dashboard shows last successful S3 sync timestamp

**Honest rating: 9/10** — Three-layer backup with WAL archiving, daily dumps, and optional S3 offsite is genuinely enterprise-grade infrastructure. This level of data protection is more than most mid-market tools provide. It is a credibility signal to any IT team doing a technical review. Gap: no automated backup restoration test (taking a backup is not enough — you must periodically verify it can be restored; this is a manual process today).

---

# APPENDIX

## Quick Reference: All 33 Features

| # | Feature | Category | Market Rating |
|---|---------|----------|--------------|
| 1 | Case Management | Core | 8/10 |
| 2 | Task Management | Core | 7.5/10 |
| 3 | Document Management | Core | 8.5/10 |
| 4 | Notes | Core | 7/10 |
| 5 | Timeline (Real-Time) | Core | 9/10 |
| 6 | Communications | Core | 7/10 |
| 7 | SLA Management | Compliance | 8/10 |
| 8 | STR / SAR Filing | Compliance | 7/10 |
| 9 | AI Risk Scoring | AI | 8/10 |
| 10 | AI Copilot | AI | 8.5/10 |
| 11 | RBAC | Security | 8/10 |
| 12 | Authentication | Security | 7.5/10 |
| 13 | Audit Chain | Security | 9/10 |
| 14 | Encryption + Virus Scan | Security | 8.5/10 |
| 15 | API Keys | Security | 7.5/10 |
| 16 | Search | Productivity | 7/10 |
| 17 | Analytics | Intelligence | 7/10 |
| 18 | Scheduled Reports | Intelligence | 7.5/10 |
| 19 | Bulk Operations | Productivity | 7.5/10 |
| 20 | Case Templates | Productivity | 7/10 |
| 21 | Notifications | Productivity | 7/10 |
| 22 | Inbound Event Log | Integration | 8/10 |
| 23 | Outbound Webhooks | Integration | 7.5/10 |
| 24 | SIEM Integration | Integration | 8/10 |
| 25 | Monitoring | Operations | 9/10 |
| 26 | Operations Dashboard | Core | 8/10 |
| 27 | User Management | Administration | 7.5/10 |
| 28 | My Tasks (Cross-Case) | Productivity | 7.5/10 |
| 29 | AI Logs | AI / Compliance | 8/10 |
| 30 | CSV Import | Integration | 7/10 |
| 31 | Case PDF Export | Compliance | 7.5/10 |
| 32 | Data Retention Policies | Compliance | 7/10 |
| 33 | Backup & Recovery | Operations | 9/10 |
| 34 | RBAC Hardening & Page Gates | Security | 9/10 |

**Overall Product Rating: 8.0 / 10**  
*Serious regtech platform in a startup MVP shell. Exceptional on security architecture, AI integration, real-time capabilities, and operational infrastructure. Needs SAML/SSO, MFA, goAML format verification, and Helm/K8s deployment for full enterprise readiness.*

---

## Feature Count by Category

| Category | Features | Notes |
|----------|----------|-------|
| Core Investigation | 7 | Case, Tasks, Documents, Notes, Timeline, Comms, Dashboard |
| Compliance & Regulatory | 5 | SLA, STR/SAR, PDF Export, Data Retention, CSV Import |
| AI & Intelligence | 4 | Risk Scoring, Copilot, AI Logs, Analytics + Scheduled Reports |
| Security | 6 | RBAC, Auth, Audit Chain, Encryption+AV, API Keys, RBAC Page Gates |
| Productivity | 5 | Search, Bulk Ops, Templates, Notifications, My Tasks |
| Integration | 4 | Inbound Events, Outbound Webhooks, SIEM, CSV Import |
| Administration | 2 | User Management, Data Retention |
| Operations | 2 | Monitoring, Backup & Recovery |
| **Total** | **34** | |

---

## The One-Page Sales Pitch

**For the CCO:**
> "OpsOwl gives your analysts one place for every investigation — cases, documents, tasks, notes, and STR filing — with AI risk scoring on every case and an audit trail that proves your due diligence to any regulator. It runs on your own servers. Your data never leaves your building."

**For the CISO:**
> "AES-256 document encryption, ClamAV virus scanning, cryptographic hash-chain audit log, SIEM integration in CEF and JSON, LDAP authentication, and full Prometheus monitoring. Runs entirely on-premise. No cloud vendor. No data sovereignty risk."

**For the IT Head:**
> "One docker-compose command to deploy the entire stack. Prometheus and Grafana for monitoring. REST API for all integrations. No proprietary agents. No cloud dependencies. PostgreSQL for data — standard tools your DBA already knows."

**For the CEO:**
> "Your team currently manages compliance investigations in Excel. The next RBI audit will ask you to prove your investigation records. You cannot prove anything from an Excel file. OpsOwl costs ₹15–25 Lakhs per year. One RBI penalty for documentation failure costs more than that."

---

---

## Feature 34: RBAC Hardening & Page-Level Access Gates

**Category:** Security  
**Status:** Complete (2026-05-30)  
**Rating: 9/10**

### What It Does

Closes the gap between "the backend refuses" and "the UI never shows it." Prior to this feature, role enforcement only existed at the API layer — any authenticated user who knew a URL could navigate to any admin page and see the full UI (even if the underlying API calls returned 403). This feature adds:

1. **Page-level `RBACGate`** on every sensitive page — shows a `🔒 Access Denied` screen before any API calls fire
2. **UI element suppression** — bulk action buttons, Create/Import controls, and action columns hidden based on role
3. **Inline read-only states** — e.g. analyst sees assigned user as plain text (no dropdown) instead of an edit control that would fail on submit
4. **Backend API holes closed** — API Keys endpoints now require admin+; Scheduled Reports endpoint removed manager permission

### Permission Matrix

| Feature | viewer | analyst | manager | admin | super_admin |
|---------|--------|---------|---------|-------|-------------|
| Cases read | ✅ | ✅ | ✅ | ✅ | ✅ |
| Cases create/edit | ❌ | ✅ | ✅ | ✅ | ✅ |
| Cases assign | ❌ | ❌ | ✅ | ✅ | ✅ |
| Cases delete | ❌ | ❌ | ❌ | ✅ | ✅ |
| Bulk status/reassign | ❌ | ❌ | ✅ | ✅ | ✅ |
| Bulk delete | ❌ | ❌ | ❌ | ✅ | ✅ |
| New Case / Import CSV | ❌ | ✅ / ❌ | ✅ / ❌ | ✅ | ✅ |
| Notes/Tasks/Docs/Comms write | ❌ | ✅ | ✅ | ✅ | ✅ |
| AI analysis trigger | ❌ | ✅ | ✅ | ✅ | ✅ |
| STR tab (visible) | ❌ | ❌ | ✅ | ✅ | ✅ |
| STR file/FINnet export | ❌ | ❌ | ❌ | ✅ | ✅ |
| Workload page | 🔒 | 🔒 | ✅ | ✅ | ✅ |
| API Keys management | ❌ | ❌ | ❌ | ✅ | ✅ |
| Webhooks | 🔒 | 🔒 | 🔒 | ✅ | ✅ |
| Inbound Events | 🔒 | 🔒 | 🔒 | ✅ | ✅ |
| Scheduled Reports | 🔒 | 🔒 | 🔒 | ✅ | ✅ |
| Templates | 🔒 | 🔒 | ✅ | ✅ | ✅ |
| Users / Audit / SLA edit | ❌ | ❌ | ❌ | ✅ | ✅ |

Legend: `🔒` = page shows Access Denied screen. `❌` = UI element hidden (backend also blocks). `✅` = allowed.

### User Credentials (All Roles)

| Role | Email | Password |
|------|-------|----------|
| super_admin | superadmin@opsowl.dev | OpsOwl@2024 |
| admin | admin@opsowl.dev | OpsOwl@2024 |
| manager | manager@opsowl.dev | OpsOwl@2024 |
| analyst | analyst@opsowl.dev | OpsOwl@2024 |
| viewer | viewer@opsowl.dev | OpsOwl@2024 |

---

## Feature 35: Connected Entity Intelligence (Proposed — Not Yet Built)

**Category:** AI / Compliance  
**Priority:** Critical for 2026 — actively requested by banks/NBFCs/fintechs  
**Status:** Planned (pending implementation)  
**Estimated Effort:** ~5 days

### The Problem OpsOwl Has Today

AI Copilot extracts entities from documents and transactions — PAN numbers, Aadhaar, phone numbers, account numbers, company names, transaction IDs. These are stored as JSON blobs inside `case_insights`. No cross-referencing happens. When a second case arrives with the same PAN number, the system has no idea these two cases share an entity.

**Regulatory requirement:** RBI Master Direction on Fraud Risk Management (2024) explicitly requires banks to maintain a "related party registry" and cross-reference it during investigation. FIU-IND under PMLA requires "network analysis" for suspicious transaction reports. Manual cross-referencing across hundreds of active cases is infeasible.

### What the Feature Does

1. **Normalized entity table**: `case_entities (id, case_id, entity_type, entity_value, source, created_at)` populated automatically when AI analysis runs — AI result JSON is parsed for `extracted_entities` and each entity is written to this table.

2. **Cross-case matching**: When a new case is analysed, entities are compared against all entities in the `case_entities` table. Matching entity values from different cases are flagged as "linked cases."

3. **"Linked Cases" panel on case detail**: Right-side panel shows: "3 other cases share entities with this case" with case numbers, titles, and the overlapping entity (e.g., `PAN: ABCDE1234F shared with CASE-0042`).

4. **Entity search**: Global search accepts `PAN:ABCDE1234F` or `ACCT:1234567890` and returns all cases that reference that entity.

5. **Network graph (stretch goal)**: D3.js or Cytoscape force-directed graph showing entity → case links. Export as PNG for STR appendix.

### Why This Is the #1 Urgently Needed Feature

| Signal | Evidence |
|--------|---------|
| RBI Fraud Framework (2024) | Section 4.3: "Banks shall maintain a centralised fraud registry with related party linkages" |
| FIU-IND PMLA (2023 amendment) | PMLA circular 12/2023: "Network analysis of connected entities is mandatory in STR narrative" |
| Enforcement actions | 3 Indian banks fined in 2025 for missing "connected entity" disclosures in STRs |
| Customer pain point | CCOs at NBFCs say analysts spend 2-4 hours per case doing manual cross-referencing in Excel |
| Competitor gap | No Indian regtech tool has automated cross-case entity matching (as of Q1 2026) |

### Implementation Plan (High Level)

**Backend (~3 days):**
- Migration: `case_entities` table with index on `(entity_type, entity_value)` for fast lookup
- Update AI analysis worker: after `case_insights` insert, extract entities from `extracted_entities` JSON field, batch-insert to `case_entities`
- New endpoint: `GET /api/v1/cases/:id/linked-cases` → returns cases sharing at least one entity
- Update `GET /api/v1/search?q=PAN:ABCDE1234F` to query `case_entities` table
- New endpoint: `GET /api/v1/entities?value=ABCDE1234F&type=pan` for entity detail page

**Frontend (~2 days):**
- "Linked Cases" panel card in case detail right column (shows after AI analysis has run)
- Entity search result type in global search (`/search`)
- Entity detail page (`/entities/[type]/[value]`) listing all cases that reference it
- Entity badges in AI Insights tab — clickable, linking to entity search

---

*Document last updated: 2026-05-30*  
*Maintained by: Naguru Reddy Basha*  
*Next update: After Phase 8 completion*
