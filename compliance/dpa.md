# Data Processing Agreement (DPA)
**OpsOwl AI — Investigation Workspace**
Version 1.0 · Effective 2026-05-26

---

## 1. Parties

| Role | Party |
|------|-------|
| **Data Controller** | The financial institution ("Customer") deploying OpsOwl AI |
| **Data Processor** | OpsOwl AI / Reddybashanaguru Technologies ("Processor") |

---

## 2. Subject Matter and Duration

The Processor processes personal data on behalf of the Controller for the purpose of operating the OpsOwl AI investigation workspace, for as long as a valid service agreement is in force.

---

## 3. Nature and Purpose of Processing

- **Purpose:** Case investigation, compliance monitoring, fraud detection support, AML/STR workflow management
- **Processing activities:** Collection, storage, analysis, access control, retention, erasure
- **Legal basis:** Performance of contract (Article 6(1)(b)); legal obligation (Article 6(1)(c)) — PMLA 2002, RBI Master Directions

---

## 4. Categories of Data Subjects and Personal Data

| Category | Data Elements |
|----------|---------------|
| Platform users (compliance officers, analysts) | Name, email, role, activity logs, session data |
| Investigation subjects | Name, PAN, account references (stored in case records and STR filings) |

Aadhaar numbers are **never stored raw** — only a masked reference per DPDP Act 2023 s.4.

---

## 5. Controller Instructions

The Processor shall process personal data only on documented instructions from the Controller, except where required to do so by applicable law.

---

## 6. Sub-processors

| Sub-processor | Purpose | Location |
|---------------|---------|----------|
| PostgreSQL (self-hosted or managed) | Primary data store | Configurable — must be within India for PSU banks |
| MinIO / S3-compatible | Document storage | Configurable |
| Anthropic / OpenAI (optional AI features) | AI analysis | Data minimisation applied; no PII sent without explicit config |

---

## 7. Security Measures

| Control | Implementation |
|---------|---------------|
| Encryption at rest | AES-256 column encryption (ENCRYPTION_KEY) |
| Encryption in transit | TLS 1.2+ enforced (REQUIRE_TLS=true in production) |
| Access control | Role-based (analyst / manager / admin / super_admin) |
| Audit logging | Hash-chain tamper-evident audit log |
| Session security | 15-minute idle timeout; max sessions per user configurable |
| Password policy | Minimum 12 characters, uppercase, number required |

---

## 8. Data Retention

| Category | Retention Period | Legal Basis |
|----------|-----------------|-------------|
| PMLA records (cases, audit_logs, str_filings, ai_logs, documents) | 10 years (3650 days) | PMLA 2002 s.12 |
| Notifications | As configured (default 90 days) | Operational need |
| Session tokens | Until logout or 15-minute idle timeout | Security |

---

## 9. Right to Erasure

Erasure requests are processed via `DELETE /api/v1/users/{id}/data`. Erasure is **blocked** for users referenced in STR filings within the last 10 years per PMLA s.12 / DPDP Act 2023 s.17.

---

## 10. Data Breach Notification

The Processor shall notify the Controller within **72 hours** of becoming aware of a personal data breach affecting Controller data.

---

*Document owner: Data Protection Officer | Review cycle: Annual*
