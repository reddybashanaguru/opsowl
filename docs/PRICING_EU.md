# OpsOwl AI — Pricing Sheet (EU Market)
**Effective: June 2026 | On-Premise Deployment | European Union**

---

## Enterprise License

OpsOwl AI is sold as a single enterprise license. There are no feature tiers — every customer gets the full product. Pricing is based on the number of named users.

---

### Annual Software License

| Component | Price |
|-----------|-------|
| **Base platform** (up to 25 named users) | €15,000 / year |
| **Additional users** (per user beyond 25) | €500 / user / year |
| **One-time Deployment & Setup** | €2,500 |

**Effective per-user cost at scale:**

| Named Users | Annual License | Per User / Month |
|-------------|----------------|------------------|
| 25 (base) | €15,000 | €50 |
| 50 | €27,500 | €46 |
| 100 | €52,500 | €44 |
| 200+ | Custom | ~€38–42 |

*Prices are exclusive of VAT. EU VAT applies per member state (e.g. 21% Lithuania, 19% Cyprus). USD equivalent: €15,000 ≈ $16,200 at €1/$1.08 reference rate.*

---

### What Is Included (Full Product, No Feature Gates)

**Case Management**
- Case lifecycle (open → in-progress → pending review → escalated → closed)
- Custom case labels, priority, tags, SLA deadlines
- Bulk assign, bulk status change, bulk CSV import (up to 1,000 cases/batch)
- Case templates — pre-fill standard AML / fraud / sanctions workflows
- Assignment rules engine — auto-assign by source, priority, tags, round-robin

**AI Copilot & Document Intelligence**
- AI Copilot — Q&A over case documents, with PII masking before any external call
- Automated risk scoring and risk level classification per case
- STR / SAR narrative generation (goAML-compatible XML output)
- PDF and image ingestion with OCR; up to 300-page chunked RAG per case
- **Air-Gapped AI by default** — all inference runs on your servers via Ollama; zero data leaves your network

**EU Compliance Suite (Full — No Extras)**
- **goAML XML export** — FNTT (Lithuania), MOKAS (Cyprus), Hellenic FIU (Greece), and all EU FIUs
- **UBO Verification** — EU AMLR 2024 Art. 42–55 (≥25% threshold), closure gate enforced
- **GDPR Erasure Workflow** — Art. 17 right-to-erasure with Art. 77 AMLR retention resolution via pseudonymisation
- **Sanctions Screening** — EU Consolidated List, UN Security Council, OFAC SDN; fuzzy matching via pg_trgm
- **PEP Identification** — integrated with sanctions screening; EDD flag triggered automatically
- **AI Act Compliance Module** — EU AI Act Art. 13 transparency + human oversight logging + CSV export for regulators
- **SLA Engine** — FNTT 3-day STR deadline, MOKAS 5-day STR deadline, configurable per jurisdiction

**Investigation Workspace**
- Live timeline with server-sent events (real-time updates)
- Internal notes (immutable, timestamped), tasks, watchers, @mentions
- Document upload, preview, version history, download
- Hash-chain tamper-evident audit log (SHA-256 chained entries)

**Security**
- TOTP-based MFA (TOTP + 8 backup codes)
- Role-based access control — 5 built-in roles (Super Admin, Admin, Manager, Analyst, Viewer)
- Session limits, concurrent session control, IP allowlisting
- AES-256 encryption at rest (documents)
- Account lockout, password policy enforcement
- SIEM integration — Splunk, QRadar, Wazuh (CEF/JSON)

**Integrations & Automation**
- LDAP / Active Directory (bind-based authentication)
- Outbound webhooks (up to 20 endpoints, HMAC-signed with retry)
- Inbound intake — webhook to auto-create cases from your alert engine
- API key management
- Scheduled reports — auto-email PDF/CSV at daily/weekly/monthly cadence
- Analytics export — aggregate CSV for BI tools (Power BI, Tableau)

**Deployment**
- Docker Compose — single-command startup, runs on any Linux server or private VM
- MinIO object storage (self-hosted, included)
- PostgreSQL 18 + pgvector (self-hosted, included)
- No cloud dependency at runtime — fully air-gapped capable

---

### One-Time Setup Fee — €2,500

| Item | Detail |
|------|--------|
| Remote installation | Docker Compose setup, TLS configuration, DNS |
| LDAP integration | Bind to your Active Directory or OpenLDAP |
| Jurisdiction configuration | FNTT / MOKAS / Hellenic FIU — goAML entity ID, STR deadlines, EU flags |
| Data migration | Import up to 5,000 existing cases from CSV/Excel |
| Admin training | 8 hours (two sessions: admin setup + investigator workflow) |
| Go-live support | 2 weeks of hypercare post-deployment (priority email + call) |

On-site deployment (engineer travels to your data centre): **€1,500 additional** + travel at actuals.

---

### Annual Support & Maintenance

Included in the annual license fee. No separate charge.

| Item | Commitment |
|------|-----------|
| Software updates | Monthly release channel — all features, security patches |
| Critical security patches | Within 72 hours of CVE disclosure |
| Support — P1 (production down) | 4-hour response (9 AM–6 PM CET) |
| Support — P2 (feature broken) | 1-business-day response |
| Support — P3 (query / how-to) | 3-business-day response |
| Dedicated Customer Success contact | Quarterly review call |
| Support channels | Email, Help Desk portal, phone (P1/P2) |

---

### Optional Add-Ons

| Add-On | Price |
|--------|-------|
| **Cloud AI upgrade** — Anthropic Claude or Azure OpenAI instead of Ollama | €1,500 / year + provider API costs (pass-through) |
| **Extra storage** — per 500 GB beyond 500 GB base | €400 / year |
| **On-site training** — full-day, up to 20 participants | €1,200 / day + travel at actuals |
| **Custom integration development** | €120 / hour (minimum 8 hours) |
| **Penetration test facilitation** — scope docs, scheduling, re-test | €2,000 / engagement |
| **Source code escrow** — third-party escrow agent | €1,200 / year |
| **24×7 support upgrade** — extends P1 response to nights and weekends | €3,000 / year |
| **goAML Direct Filing API** — electronic STR submission (where FIU API is available) | €1,500 / year |

---

### Discount Schedule

| Condition | Discount |
|-----------|----------|
| 2-year upfront commitment | 8% off Year 2 |
| 3-year upfront commitment | 15% off Years 2–3 |
| Reference customer (written case study + demo participation) | €1,500 credit on Year 1 invoice |
| Pilot customer (first 5 EU customers) | 20% off Year 1 — design partner rate |
| Emergency deployment during FNTT / MOKAS / CySEC audit | Setup fee waived |

---

### Payment Terms

| Schedule | Terms |
|----------|-------|
| Annual (standard) | 100% of annual license + setup fee invoiced at signing |
| Semi-annual | 50% at signing, 50% at month 6; +3% premium |
| Renewal | Invoice raised 45 days before anniversary; payment due within 30 days |

All invoices in EUR. VAT added per member state. Wire transfer (SEPA or SWIFT) or card accepted.

---

### Proof of Concept (POC)

A 30-day POC is available at no cost for qualified prospects.

- Deployed on your own infrastructure (or a temporary VM provisioned by OpsOwl)
- Full production feature set — EU compliance suite fully enabled
- Up to 5 users during POC
- FNTT / MOKAS goAML config pre-loaded for your jurisdiction
- Setup fee waived if a commercial contract is signed within 60 days of POC start

---

### Indicative Year-1 Investment Examples

| Organisation | Country | Users | Annual License | Setup | **Year-1 Total (excl. VAT)** |
|---|---|---|---|---|---|
| Lithuanian EMI — 15 investigators | Lithuania | 20 | €15,000 | €2,500 | **€17,500** |
| Cypriot FX broker — AML team | Cyprus | 30 | €17,500 | €2,500 | **€20,000** |
| Greek commercial bank — compliance + fraud | Greece | 60 | €32,500 | €2,500 | **€35,000** |
| Pan-EU fintech — multi-jurisdiction team | EU | 100 | €52,500 | €2,500 | **€55,000** |

---

### Competitive Position

| Platform | Typical EU Deal | Notes |
|---|---|---|
| NICE Actimize | €300K–€1.5M / year | Tier-1 banks only; 12–24 month implementation; SaaS (data leaves your network) |
| Napier AI | €80K–€400K / year | Cloud-native; no on-prem option; no EU AI Act module |
| Quantexa | €200K–€800K / year | Entity resolution focus; requires data lake; no case workspace |
| ComplyAdvantage | €30K–€100K / year | SaaS screening only; no case management; no investigation workspace |
| Generic tools (Jira / ServiceNow) | €10K–€50K / year | No audit chain; no goAML; no sanctions; no EU regulatory workflow |
| **OpsOwl AI** | **€17,500–€55,000 / year** | AI-native; on-prem; 1-day deployment; full EU compliance suite; FNTT/MOKAS ready |

---

### Why On-Premise Matters for EU-Regulated Firms

- **GDPR Art. 32** — technical measures must ensure data security. Cloud = shared infrastructure risk.
- **DORA (EU 2022/2554)** — ICT risk management requires knowing exactly where data resides.
- **Bank of Lithuania / CySEC supervisory expectations** — regulators increasingly ask where customer data is processed.
- **AMLR 2024 Art. 77** — 5-year retention obligation; data must be recoverable and auditable at your institution.

With OpsOwl, all data stays on your servers. No API call to our systems at runtime. Your data centre, your control.

---

*All prices in EUR, exclusive of VAT. Prices subject to revision with 60 days' written notice to existing customers.
For a formal quote or to schedule a live demo: **contact@opsowl.dev***
