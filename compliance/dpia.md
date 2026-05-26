# Data Protection Impact Assessment (DPIA)
**OpsOwl AI — Investigation Workspace**
Version 1.0 · Conducted 2026-05-26

---

## 1. Executive Summary

This DPIA covers the deployment of OpsOwl AI at a financial institution for AML/fraud investigation workflows. High-risk processing activities are identified and mitigated. The assessment concludes that residual risk is **acceptable** subject to the controls listed.

---

## 2. Necessity and Proportionality

| Processing Activity | Necessity | Proportionality |
|--------------------|-----------|-----------------|
| Case management with subject PII | Required by PMLA for STR/SAR filing | Limited to what FIU-IND mandates |
| AI-assisted analysis | Reduces analyst error; time-bound | No AI output stored as fact without human review |
| Audit chain logging | Regulatory mandated (RBI) | Minimal PII — actor IDs and event types only |
| Session tracking | Security best practice | 15-minute idle timeout; sessions purged on logout |

---

## 3. Risk Assessment

| Risk | Likelihood | Impact | Inherent Risk | Mitigation | Residual Risk |
|------|-----------|--------|---------------|-----------|---------------|
| Unauthorised access to STR subjects | Medium | High | High | Role-based access; analyst blocked from STR routes (PMLA s.8A) | Low |
| Cross-org data leak (IDOR) | Low | High | Medium | Org-scoping at service layer; IDOR integration tests | Low |
| PII exposure via AI requests | Medium | High | High | No PII sent to external AI without explicit config; AI service is local-first | Low |
| Tipping-off (PMLA s.8A) | Medium | High | High | Analyst role excluded from STR/regulatory report routes | Low |
| Prolonged retention beyond mandate | Low | Medium | Medium | Retention scheduler; PMLA floor trigger (10-year minimum) | Low |
| Aadhaar raw storage | Low | Critical | High | Aadhaar masked at input (first4 XXXX last4); never persisted raw | Low |
| Data breach | Low | High | Medium | TLS enforcement; AES-256; SIEM alerts; audit chain | Low |

---

## 4. High-Risk Indicators (GDPR Article 35 / DPDP Act 2023 Schedule II)

- [x] Processing of sensitive financial data at scale → **Mitigated by access controls**
- [x] Automated profiling (AI analysis) → **Human-in-the-loop; outputs are advisory only**
- [x] Systematic monitoring of individuals → **Limited to compliance officers performing statutory duty**
- [ ] Large-scale public data — **Not applicable**

---

## 5. Consultation

- Data Protection Officer: reviewed 2026-05-26
- Legal (PMLA / DPDP Act): reviewed 2026-05-26
- IT Security: reviewed 2026-05-26
- DPO conclusion: **Processing is lawful; mitigations are adequate**

---

## 6. Sign-off

| Role | Name | Date |
|------|------|------|
| DPO | *[To be completed by customer organisation]* | |
| CISO | *[To be completed by customer organisation]* | |
| Compliance Head | *[To be completed by customer organisation]* | |

---

*Review trigger: material change to processing activities, new jurisdiction, or annual cycle.*
