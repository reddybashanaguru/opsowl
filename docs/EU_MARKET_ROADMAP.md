# OpsOwl — EU / Greece Market Roadmap

**Branch:** `feature/eu-greece-market-readiness`  
**Initial deployment country:** Greece (`DEPLOYMENT_COUNTRY=gr`, `DEPLOYMENT_JURISDICTION=eu`)  
**Expansion path:** GR → BG → RO → HR → Other EU member states  
**Regulators:** Hellenic FIU (SDOE) · Bank of Greece · HCMC · EU AMLA (from 2025)

---

## Why EU / Greece — Why Now (2026–2027)

### The Regulatory Inflection Point

The EU is undergoing the most significant AML/CFT regulatory overhaul in a generation.
Three regulations now force every obliged entity in Europe to upgrade or replace their
compliance infrastructure by mid-2027:

| Regulation | In Force | Key Obligation |
|-----------|---------|---------------|
| EU AMLR 2024 (Anti-Money Laundering Regulation) | July 2027 (full) | Single harmonised AML rulebook across all 27 member states |
| EU AMLA (Anti-Money Laundering Authority) | July 2025 (phased) | New supranational AML supervisor in Frankfurt; direct supervision of high-risk entities |
| EU AI Act | Aug 2026 (high-risk AI) | AML risk-scoring AI must be explainable, auditable, human-overseen |
| DORA (Digital Operational Resilience Act) | Jan 2025 | ICT incident reporting and resilience for financial entities |
| MiCA (Markets in Crypto-Assets) | Dec 2024 | Crypto exchanges become obliged AML entities across all EU |
| EU Transfer of Funds Regulation (TFR) | Dec 2024 | Travel Rule — full originator/beneficiary data on all crypto transfers |

**No existing AML platform was built with EU AMLR 2024 as the native framework.**
Incumbents (Actimize, Quantexa, Napier) were built for US/UK markets and are retrofitting EU compliance.
OpsOwl can be AMLR-2024 native from day one — that is the disruption vector.

### Why Greece First

1. **Regulatory urgency:** Greece has been under FATF enhanced monitoring (grey-listed) for years.
   Following exit from enhanced monitoring, Bank of Greece has intensified AML supervision.
   Greek FIs are under active pressure to modernise infrastructure NOW.

2. **Market gap:** The Greek market (≈ 40 licensed banks + 400+ payment institutions + 200+ DNFBPs)
   has very few sophisticated, modern AML investigation platforms. Most use spreadsheets or
   legacy systems (Siron, legacy SAS AML).

3. **Price leverage:** Greek FIs are cost-sensitive. OpsOwl's on-premise model eliminates
   cloud data sovereignty concerns (Greek data stays in Greece).

4. **Expansion beachhead:** Greece → Bulgaria → Romania → Croatia is a natural
   progression into the EU's Eastern European growth markets, all post-2007 EU members
   with similar regulatory catch-up dynamics.

---

## What Is Already in `main`

- EU jurisdiction defaults: `DEPLOYMENT_JURISDICTION=eu`, `DEPLOYMENT_COUNTRY=gr`
- `goaml_enabled: true`, `ubo_enabled: true`, `gdpr_enabled: true` (env-configurable)
- EU AMLR 2024 / AMLD6 as law reference
- Hellenic FIU as regulator label (when `DEPLOYMENT_COUNTRY=gr`)
- AFM/ID Card as primary ID type
- EUR currency, €10,000 CTR threshold, €1,000 wire record-keeping threshold
- GDPR section in settings (informational)
- UBO section in settings (informational)
- Retention: 5 years (AMLR 2024 Art. 77)
- UBO threshold: 25% (AMLR 2024 Art. 42)

---

## EU-Specific Features to Build (Delta on Top of main)

### P-EU-1: GDPR Erasure Workflow ⭐ Critical Differentiator

**Priority:** Critical — no competitor handles this correctly  
**Regulatory basis:** GDPR Art. 17 (right to erasure) vs. EU AMLR 2024 Art. 77 (5-year retention)  

The tension between GDPR erasure and AML retention is the single biggest compliance
headache for EU FIs. The resolution is **anonymisation, not deletion**: personal
identifiers are replaced with pseudonyms while the transaction audit trail survives.

**Features:**
- **Data Subject Access Request (DSAR) portal** — inbound requests from individuals
- **Erasure eligibility check** — if case is closed AND retention period has expired → eligible
- **Anonymisation pipeline** — replace PII fields (name, ID, address) with `[ANONYMISED-UUID]`
  while preserving transaction amounts, dates, case narrative, and risk score
- **Audit trail** — who processed the DSAR, when, what was anonymised, legal basis cited
- **Retention expiry monitor** — automated alerts when cases approach retention limit
- **DPO dashboard** — Data Protection Officer view of all active DSARs, deadlines, status

**Legal basis recorded on every anonymisation:** `GDPR Art. 17(3)(b) — AML retention exemption
under EU AMLR 2024 Art. 77` if within 5 years; full erasure if beyond.

**Files:**
- `apps/api/internal/gdpr/` — DSAR model, anonymisation service, handler
- DB migration: `dsar_requests` table
- `apps/web/src/app/(dashboard)/settings/gdpr/` — DSAR management UI

---

### P-EU-2: UBO Verification Workflow ⭐ High Impact

**Priority:** High  
**Regulatory basis:** EU AMLR 2024, Art. 42–55 (beneficial ownership)  

AMLR 2024 mandates that obliged entities verify the UBO (≥25% ownership) of every
legal entity customer BEFORE onboarding and periodically thereafter.

**Features:**
- **UBO capture on case creation** — structured UBO form (name, nationality, ownership %, ID)
- **UBO verification status** — Pending / Verified / Discrepant / Registry Mismatch
- **Registry cross-check links** (deep link to national registries):
  - Greece: GEMI (Γενικό Εμπορικό Μητρώο) — `efiling.gsis.gr`
  - Germany: Handelsregister — `handelsregister.de`
  - France: Infogreffe — `infogreffe.fr`
  - Netherlands: KVK — `kvk.nl`
  - Spain: Registro Mercantil — `registradores.org`
- **Case closure gate** — case cannot be closed if UBO status is still Pending
- **UBO change tracker** — if ownership structure changes mid-investigation, flag for re-verification
- **AMLR 2024 Art. 42 compliance audit report** — exportable per case

**Files:**
- DB migration: `ubo_records` table, FK to `cases`
- `apps/api/internal/ubo/` — UBO service + handler
- `apps/web/src/app/(dashboard)/cases/[id]/ubo/` — UBO tab

---

### P-EU-3: goAML Multi-Country XML Export

**Priority:** High  
**Regulatory basis:** All EU member states — FIU submission format  

~19 of 27 EU member states use UNODC goAML. Unlike UAE (single FIU),
each EU member state has its own goAML portal. The XML schema is the same
(UNODC v4.0) but entity ID, portal URL, and country codes differ.

**Features:**
- Reuse `apps/api/internal/goaml/` package from UAE branch (shared code, correct branch architecture)
- EU-specific validation: IBAN format, Greek AFM (9-digit TIN) check
- Member state FIU portal registry (configurable per deployment via env)
- Batch XML generation for multiple transactions
- XML download + upload to national goAML portal (via GOAML_PORTAL_URL)
- Submission receipt tracking (reference number from FIU)

**Note:** The `goaml` package should eventually be merged into `main` since both UAE and
EU use it. Consider promoting it when the UAE branch is validated.

---

### P-EU-4: EU Sanctions & PEP Screening ⭐ Revenue Driver

**Priority:** High  
**Regulatory basis:** EU Consolidated Sanctions List (EEAS) · UN SC Consolidated List  

**Three lists to screen against:**
1. **EU Consolidated Sanctions List** (EEAS, published at `data.europa.eu/euodp`)
   — Updated daily; must screen all subjects
2. **UN Security Council Consolidated List** (UNSC 1267/1989/2253)
   — Required under AMLR 2024
3. **EU PEP Database** (via commercial data provider: Refinitiv, Dun & Bradstreet,
   or open-source alternatives like `opencorporates`)

**Greece-specific additions:**
- Greek PEP list (current and former Greek political figures)
- Transliteration: Greek → Latin script name normalisation
  (e.g. "Παπαδόπουλος" → "Papadopoulos" — multiple Latin variants exist)
- Greek citizens may appear in lists under both Greek and transliterated names

**Features:**
- Reuse `apps/api/internal/sanctions/` screener from UAE branch (Jaro-Winkler fuzzy matching)
- Add `eu_sanctions.go` provider (EEAS REST API or daily XML file download)
- Add `eu_pep.go` provider (configurable external data provider)
- Greek transliteration normaliser (ISO 843 / ELOT 743)
- Screening triggered on: subject creation, case escalation, case closure
- Batch re-screen when list updates arrive
- Match review workflow: auto-escalate Confirmed Hits, queue Potential Hits for analyst review

**Files:**
- `apps/api/internal/sanctions/eu_sanctions.go`
- `apps/api/internal/sanctions/greek_transliteration.go`
- `apps/api/internal/sanctions/pep_provider.go`

---

### P-EU-5: AI Act Compliance Module ⭐ Unique Differentiator (No Competitor Has This)

**Priority:** High — must comply by August 2026  
**Regulatory basis:** EU AI Act, Annex III (High-Risk AI Systems) — AML risk scoring qualifies  

AML risk scoring AI is classified as a **High-Risk AI System** under the EU AI Act.
This requires:
1. **Explainability** — "Why did this case get a 9.2/10 risk score?"
2. **Human oversight** — every AI-generated risk score must have a logged human review
3. **Bias & fairness monitoring** — model must not discriminate by nationality, gender, etc.
4. **Technical documentation** — must be available for supervisory review

No existing AML platform has built this. OpsOwl can be the first.

**Features:**
- **"Why this score?" panel** — per-case AI risk score explanation in natural language
  (factor attribution: which signals contributed most — transaction pattern 40%,
  sanctions proximity 30%, PEP relationship 20%, etc.)
- **Human oversight log** — every risk score change must be accepted/overridden by
  an analyst, with mandatory reason code (AI Act Art. 14)
- **Model card** — publicly accessible model documentation (training data distribution,
  known limitations, update history)
- **Bias audit dashboard** — aggregate statistics on risk score distribution by
  subject nationality (to detect discriminatory patterns)
- **AI Act readiness checklist** — auto-scored compliance checklist with gap analysis

**Files:**
- `apps/api/internal/ai_act/` — oversight log, compliance checker
- `apps/web/src/components/cases/AIScoreExplainability.tsx`
- DB migration: `ai_oversight_log` table

---

### P-EU-6: DORA Operational Incident Reporting

**Priority:** Medium  
**Regulatory basis:** EU DORA Art. 19 — ICT-related incident reporting  
**In force:** January 17, 2025  

If OpsOwl itself has a service disruption, the deploying FI must report it to
their NCA (National Competent Authority) within defined timeframes:
- **Major ICT incident** → initial report within 4 hours of classification
- **Significant cyber threat** → notify NCA proactively

**Features:**
- Incident classification wizard (based on DORA criteria: scale, duration, data impact)
- Automatic determination: is this "major"? → threshold-based scoring
- Report template generation: initial notification / intermediate / final report
- Submission audit trail
- Bank of Greece (for Greece) as default NCA — configurable via `DORA_NCA_ENDPOINT`

---

### P-EU-7: Cross-Border Investigation Coordination (Forward-Looking 2027)

**Priority:** Medium — differentiator for multi-country FI groups  
**Regulatory basis:** EU AMLR 2024 cooperation requirements; Eurojust referral procedures  

When an investigation spans multiple EU jurisdictions (e.g., a Greek bank with
operations in Cyprus and Bulgaria), analysts need to coordinate.

**Features:**
- **Case linking across deployments** — reference a case in another OpsOwl instance
  (secure ID reference, not data sharing)
- **Eurojust referral template** — structured referral package (case summary, subjects,
  transactions, risk narrative) in the format expected by Eurojust/FIU.net
- **FIU.net integration** (future) — EU's secure FIU-to-FIU information exchange network

---

### P-EU-8: EU AMLA Supervisory Readiness

**Priority:** Medium — forward-looking for 2027 when AMLA starts direct supervision  
**Regulatory basis:** AMLA Regulation (EU) 2024/1620  

The EU Anti-Money Laundering Authority begins direct supervision of selected
high-risk entities (≈ 40 largest EU FIs) in 2027. All EU FIs will be indirectly supervised.

**Features:**
- **AMLA supervisory data package** — export case statistics, risk scores, filing rates in
  the format AMLA will require (standard not yet published; build to current AMLA guidance)
- **Compliance posture dashboard** — real-time % of cases filed on time, % with UBO verified,
  average investigation time, AI oversight rate
- **AMLA guidance tracker** — in-app notifications when new AMLA technical standards are published

---

### P-EU-9: Greek Language Interface

**Priority:** Medium — accelerates adoption in Greece  

**Features:**
- Greek translations for all configurable labels (via `DEPLOYMENT_LANGUAGE=el`)
- Jurisdiction-specific terminology: "ΥΠΕ" (Υπόθεση Παραπόμπης Εγκλήματος) for case, etc.
- Greek date format (DD/MM/YYYY), decimal format (comma as separator)
- Greek AFM (ΑΦΜ) validation — 9-digit TIN with check-digit algorithm
- IBAN format validation for Greek accounts (GR + 25 digits)

---

### P-EU-10: MiCA / VASP AML Module (Expansion Phase)

**Priority:** Low (Q3 2027)  
**Regulatory basis:** MiCA Art. 60–62 + EU TFR (Transfer of Funds Regulation)  

**Features:**
- Crypto transaction import (from VASP internal ledger or blockchain explorer)
- Travel Rule compliance: originator + beneficiary data capture for transfers ≥ €1,000
- VASP counterparty verification (is receiving VASP MiCA-licensed?)
- Crypto risk scoring: chain analysis integration (Chainalysis / Elliptic API)
- Wallet clustering for entity identification

---

## Build Priority & Phasing

### Phase A — Greece Launch (Q2–Q3 2026)

**Goal:** First paying Greek client. Revenue. Reference case.

| Feature | Sprint | Effort |
|---------|--------|--------|
| P-EU-1: GDPR Erasure Workflow | Sprint 1-2 | 8 days |
| P-EU-2: UBO Verification Workflow | Sprint 2-3 | 6 days |
| P-EU-3: goAML EU XML Export | Sprint 3 | 3 days (reuse UAE code) |
| P-EU-4: EU Sanctions Screening | Sprint 3-4 | 5 days (reuse UAE screener) |
| P-EU-9: Greek Language Interface | Sprint 4 | 4 days |

Total: ~26 engineering days

### Phase B — AI Act Compliance (Q3 2026)

**Goal:** Be the only AML platform with certified AI Act compliance before August 2026 deadline.

| Feature | Sprint | Effort |
|---------|--------|--------|
| P-EU-5: AI Act Compliance Module | Sprint 5-7 | 12 days |
| P-EU-6: DORA Incident Reporting | Sprint 7 | 3 days |

Total: ~15 engineering days

### Phase C — EU Expansion (Q4 2026)

**Goal:** Second and third EU clients (Bulgaria, Romania, Croatia).

| Feature | Sprint | Effort |
|---------|--------|--------|
| P-EU-7: Cross-Border Coordination | Sprint 8-9 | 8 days |
| Additional EU country overlays | Sprint 9 | 2 days |

### Phase D — AMLA & Crypto (2027)

| Feature | Sprint | Effort |
|---------|--------|--------|
| P-EU-8: AMLA Supervisory Readiness | Sprint 10-11 | 6 days |
| P-EU-10: MiCA / VASP Module | Sprint 12-14 | 15 days |

---

## Competitive Landscape (EU AML Software Market)

| Competitor | Weakness OpsOwl Exploits |
|-----------|------------------------|
| **Actimize (NICE)** | Legacy, expensive (€500K+/yr), US-centric, no AMLR 2024 native support |
| **Napier AI** | UK-focused, no on-premise for regulated EU clients |
| **Quantexa** | Graph analytics focus, not a full investigation workspace |
| **Oracle AML** | Expensive, complex integration, slow innovation |
| **Siron (Tonbeller/Sopra Banking)** | Used by Greek banks but 15+ years old, no AI |
| **SAS AML** | Analytics platform, not an investigation workspace, expensive |

**OpsOwl's differentiation for EU:**
1. **AMLR 2024 native** — built for the new regulation, not retrofitted
2. **AI Act compliant by design** — explainability + oversight built in, not bolted on
3. **GDPR-AML reconciliation** — the only platform that solves the erasure vs. retention conflict
4. **On-premise** — data stays in Greece/EU, no data sovereignty concerns
5. **Modern UI** — analysts actually want to use it (vs. 15-year-old Siron screens)
6. **Pricing** — 70-80% cheaper than Actimize/SAS for comparable obliged entities

---

## Target Clients (Greece First)

| Segment | Count (Greece) | AML Urgency | OpsOwl Fit |
|---------|---------------|-------------|------------|
| Commercial banks | 12 licensed | Very High (BoG supervision) | ✅ Perfect |
| Cooperative banks | 8 | High | ✅ Perfect |
| Payment institutions | 35 | High (PSP AML obligations) | ✅ Good |
| E-money institutions | 15 | High | ✅ Good |
| DNFBPs (real estate, lawyers, accountants) | 200+ | Medium | 🟡 Phase 2 |
| Crypto exchanges (MiCA-licensed) | 5–10 by 2026 | High | 🟡 Phase D |

**Immediate beachhead:** The 12 Greek commercial banks, all under enhanced Bank of Greece
supervision, all facing AMLR 2024 compliance deadlines, all running legacy platforms.

---

## Regulatory Calendar (EU / Greece 2025–2027)

| Date | Event | OpsOwl Opportunity |
|------|-------|-------------------|
| Jan 2025 | DORA enters full effect | Offer DORA incident reporting |
| Jul 2025 | EU AMLA begins phased operations | Position as AMLA-ready platform |
| Dec 2025 | EU AMLR 2024 secondary technical standards published | Get ahead of competitors |
| Aug 2026 | EU AI Act — High-Risk AI system obligations apply | Be the only compliant AML platform |
| Jan 2027 | AMLR 2024 transition period ends — FIs must comply | All Greek FIs need new/upgraded systems |
| Jul 2027 | EU AMLR 2024 fully in force | Market peak demand — be already deployed |
| 2027 | EU AMLA direct supervision of ~40 largest EU FIs | AMLA readiness module ready |
