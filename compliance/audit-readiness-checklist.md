# Audit Readiness Checklist
**OpsOwl AI — Investigation Workspace**
RBI / PMLA / DPDP Act 2023 — Pre-Audit Verification

---

## How to Use

Work through each section before an external audit. Mark items ✅ (done), 🔶 (partial / in progress), or ❌ (not done). Items marked ❌ should be resolved before the audit date.

---

## Section 1: Access Control and Authentication

- [ ] All user accounts have the minimum necessary role (analyst / manager / admin)
- [ ] No shared accounts — each officer has a unique login
- [ ] MFA is enabled for all manager/admin accounts
- [ ] Password policy enforces ≥12 chars, uppercase, and number (`PASSWORD_MIN_LENGTH=12`)
- [ ] Account lockout is configured (check `MAX_FAILED_ATTEMPTS` env var)
- [ ] Max sessions per user is configured (`MAX_SESSIONS_PER_USER`)
- [ ] All ex-employee accounts are deactivated (`is_active = false`)
- [ ] API keys are reviewed and expired keys are revoked

---

## Section 2: Audit Logging

- [ ] Audit logs are enabled and flowing to SIEM (`SIEM_WEBHOOK_URL` configured)
- [ ] Hash chain integrity is verified — no broken chain entries in `audit_logs`
- [ ] Audit logs cover: login, logout, failed login, case changes, STR actions, user management
- [ ] Audit logs are retained for ≥10 years (verify via `data_retention_policies`)
- [ ] SIEM queue depth is ≥1000 — verify `siem.Forwarder` channel size in code

---

## Section 3: STR / AML Workflow

- [ ] At least two managers/admins are configured (maker-checker requires distinct officers)
- [ ] All STR filings in "submitted" status have a different approved_by from submitted_by
- [ ] No STR filings in draft/submitted status are older than 30 days without action
- [ ] FIU-IND FINnet 2.0 XML export tested and validated for at least one filing
- [ ] Analyst role is confirmed blocked from STR listing and regulatory report routes
- [ ] Aadhaar references in str_filings are masked (pattern: XXXXXXXXXXXXXX — 4 digits + XXXX + 4 digits)

---

## Section 4: Data Retention

- [ ] All PMLA resource types have `retention_days = 3650` in `data_retention_policies`
- [ ] `purge_enabled = false` for PMLA resource types (str_filings, audit_logs, cases)
- [ ] Retention trigger `trg_pmla_retention_floor` exists on `data_retention_policies` table
- [ ] Retention scheduler is running (check pod/process status)
- [ ] No PMLA records have been purged within the 10-year window

---

## Section 5: Encryption and TLS

- [ ] `ENCRYPTION_KEY` is set and decodes to exactly 32 bytes (AES-256)
- [ ] `REQUIRE_TLS=true` in production environment
- [ ] All inter-service communication uses TLS (DB, storage, AI service)
- [ ] No default/weak credentials (`minioadmin`, `password`, `OpsOwl@2024`) in production
- [ ] JWT keys are RSA-2048 or better; private key is not world-readable

---

## Section 6: DPDP Act 2023 Compliance

- [ ] DPA is signed with any sub-processors that handle personal data
- [ ] DPIA is completed and on file
- [ ] Erasure endpoint (`DELETE /api/v1/users/{id}/data`) is functional
- [ ] Erasure is correctly blocked for users in active STR filings (PMLA hold)
- [ ] Aadhaar data is never stored in raw form (verified in `str_filings.subject` JSONB)
- [ ] Data localisation: storage and database are within India (for PSU banks)

---

## Section 7: Incident Response

- [ ] Incident response plan is current and distributed to key personnel
- [ ] Contact list in `compliance/incident-response.md` is populated with real names
- [ ] Last tabletop exercise completed within 12 months
- [ ] RBI nodal officer is registered and contact details are current
- [ ] FIU-IND reporting workflow is tested

---

## Section 8: Documentation

- [ ] `compliance/dpa.md` — DPA is current and signed
- [ ] `compliance/dpia.md` — DPIA is current; residual risks are documented
- [ ] `compliance/rbi-compliance-matrix.md` — all controls are marked ✅ or 🔶 with owner
- [ ] `compliance/pmla-controls.md` — STR workflow and retention controls documented
- [ ] `compliance/incident-response.md` — contact list populated; plan tested
- [ ] `compliance/vendor-assessment.md` — all vendors reviewed; AI vendor risk addressed
- [ ] Change log maintained for all production configuration changes

---

## Pre-Audit Sign-Off

| Signatory | Role | Date | Signature |
|-----------|------|------|-----------|
| | Compliance Head | | |
| | CISO | | |
| | DPO | | |
| | RBI Nodal Officer | | |

---

*This checklist must be completed and signed no later than 5 business days before the scheduled audit date.*
