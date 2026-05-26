# Vendor / Third-Party Risk Assessment
**OpsOwl AI — Investigation Workspace**
Version 1.0 · 2026-05-26

---

## 1. Assessment Scope

All third-party software and services integrated into the OpsOwl AI deployment that may process, store, or transmit personal or financial data.

---

## 2. Vendor Register

### 2.1 Database — PostgreSQL (pgvector/pg17)

| Attribute | Detail |
|-----------|--------|
| Vendor | PostgreSQL Global Development Group (open source) |
| Purpose | Primary data store |
| Data processed | All platform data including PII and STR filings |
| Location | Customer-controlled (self-hosted or managed service) |
| Encryption | At-rest: AES-256 (application layer); In-transit: TLS |
| Data residency | Customer controls — must be within India for PSU banks (RBI 2018) |
| Backup | Nightly pg_dump sidecar; configurable retention |
| Risk | **Medium** — well-established OSS; no vendor data sharing |

### 2.2 Object Storage — MinIO / S3-compatible

| Attribute | Detail |
|-----------|--------|
| Vendor | MinIO (open source) or AWS S3 / GCS (customer choice) |
| Purpose | Document and attachment storage |
| Data processed | Case documents, uploaded files |
| Location | Customer-controlled |
| Encryption | SSE with configurable key; TLS in transit |
| Risk | **Medium** — depends on deployment choice |

### 2.3 AI Inference — Anthropic / OpenAI (optional)

| Attribute | Detail |
|-----------|--------|
| Vendor | Anthropic PBC / OpenAI LP |
| Purpose | AI Copilot, document summarization, investigation insights |
| Data processed | Case narratives, documents (only if AI_SERVICE_URL points to external API) |
| Location | US-based by default |
| Data sharing | Subject to vendor's API ToS and data processing agreements |
| Risk | **High if PII sent** — mitigate by deploying local AI service (Ollama / private LLM) |
| Mitigation | Set `AI_SERVICE_URL` to on-premise inference; review prompts for PII minimisation |

### 2.4 Email — SMTP Provider (optional)

| Attribute | Detail |
|-----------|--------|
| Vendor | Customer-configured SMTP (SendGrid, SES, on-premise) |
| Purpose | Notifications, password reset |
| Data processed | User email addresses; notification content |
| Risk | **Low** — standard transactional email; no financial data |

### 2.5 SIEM / SOAR (optional)

| Attribute | Detail |
|-----------|--------|
| Vendor | Customer-configured (Splunk, QRadar, Microsoft Sentinel, etc.) |
| Purpose | Security event monitoring |
| Data processed | Audit event metadata (actor IDs, event types, IP addresses) |
| Risk | **Low** — no PII sent; CEF/JSON event format |

---

## 3. Risk Matrix

| Vendor | Risk Level | Residual Risk After Mitigations |
|--------|-----------|--------------------------------|
| PostgreSQL | Medium | Low (customer-controlled) |
| MinIO / S3 | Medium | Low (customer-controlled) |
| Anthropic / OpenAI | High | Low (local deployment) / Medium (cloud API) |
| SMTP provider | Low | Low |
| SIEM/SOAR | Low | Low |

---

## 4. Third-Party Software Inventory (key dependencies)

| Library | Purpose | License | Security Notes |
|---------|---------|---------|---------------|
| `github.com/go-chi/chi` | HTTP router | MIT | No known CVEs |
| `golang.org/x/crypto` | bcrypt, TLS | BSD | Updated regularly |
| `github.com/jackc/pgx` | PostgreSQL driver | MIT | Well-maintained |
| `github.com/google/uuid` | UUID generation | BSD | Stable |
| `go.uber.org/zap` | Structured logging | MIT | No data leakage known |
| `github.com/pressly/goose` | DB migrations | MIT | Schema-only |
| `next` (Next.js) | Frontend framework | MIT | XSS mitigations in place |
| `@tanstack/react-query` | Data fetching | MIT | No data storage |

---

## 5. Review and Approval

| Reviewer | Date | Finding |
|---------|------|---------|
| Security Lead | *[Customer]* | |
| DPO | *[Customer]* | |
| Procurement | *[Customer]* | |

---

*Review trigger: new vendor onboarded, annual, or regulatory request.*
