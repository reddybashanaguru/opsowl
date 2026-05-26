# PMLA Controls Reference
**OpsOwl AI — Prevention of Money Laundering Act 2002 Controls**
Version 1.0 · 2026-05-26

---

## Applicable Sections

| PMLA Section | Requirement | OpsOwl Control |
|-------------|------------|----------------|
| s.3 | Offence of money laundering | N/A — platform does not adjudicate offences |
| s.8A | Tipping-off prohibition | Analyst role excluded from STR creation, listing, and regulatory reports |
| s.12 | Record maintenance — 10 years | Retention trigger enforces ≥ 3650 days for all PMLA record types |
| s.12(1)(a) | Account/transaction records | Cases, documents, AI logs, audit logs — all at 3650 days |
| s.12(1)(b) | STR / SAR / CTR filings | str_filings table with purge_enabled=false |
| s.13 | FIU-IND reporting | FINnet 2.0 XML export; maker-checker dual authorisation |

---

## STR Dual-Authorization Workflow

The PMLA requires that reporting by a Reporting Entity to FIU-IND is accurate and authenticated. OpsOwl enforces a **maker-checker** pattern:

```
Compliance Officer A (Maker)         Compliance Officer B (Checker)
        │                                       │
   Create Draft ──────────────────────────────▶ │
        │                                       │
   Submit for Review                            │
        │                                       │
        │ ◀────────────────── Approve/Reject ───┤
        │                           │
   [If Approved]                    │
        │                           │
   Admin marks as Filed             │
        │                           │
   Export FINnet 2.0 XML ──────── FIU-IND
```

**Controls:**
- `submitted_by ≠ approved_by` enforced by DB CHECK constraint
- Same enforcement in application layer WHERE clause
- Maker-checker violation returns HTTP 403 with code `MAKER_CHECKER_VIOLATION`
- Analyst role is fully excluded from all STR routes (PMLA s.8A)

---

## Record Retention Implementation

The database trigger `trg_pmla_retention_floor` (migration 00024) fires on INSERT and UPDATE to the `data_retention_policies` table and **raises an exception** if any PMLA-covered resource type is set below 3650 days:

```sql
-- PMLA resource types protected by the floor trigger:
-- 'audit_logs', 'cases', 'documents', 'ai_logs', 'str_filings'
```

The scheduler (`retention.Scheduler`) runs nightly but respects the trigger — it cannot reduce retention below the statutory floor regardless of configuration.

---

## FIU-IND FINnet 2.0 XML Format

OpsOwl generates FINnet 2.0 compliant XML at `GET /api/v1/cases/{id}/str/{str_id}/export`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<FINnet2Report xmlns="urn:fiu-ind:finnet2:report:v2.0"
               ReportType="STR"
               ReportRefNo="OWL-xxxxxxxx"
               FilingDate="2026-05-26">
  <FilingEntity>...</FilingEntity>
  <Subject>
    <Identifiers>
      <PAN>ABCDE1234F</PAN>
      <AadhaarHash>1234XXXX9012</AadhaarHash>   <!-- masked, never raw -->
    </Identifiers>
  </Subject>
  <Transactions>...</Transactions>
  <SuspiciousActivity>
    <SuspicionType>FIN_01</SuspicionType>       <!-- keyword-mapped code -->
    ...
  </SuspiciousActivity>
  <Certification>
    <SubmittedBy>Officer A</SubmittedBy>
    <ApprovedBy>Officer B</ApprovedBy>          <!-- must differ from SubmittedBy -->
    ...
  </Certification>
</FINnet2Report>
```

**Aadhaar masking** (`hashIfPresent`): Stores `first4 + "XXXX" + last4` only — raw Aadhaar is never persisted, per DPDP Act 2023 s.4.

---

## Suspicious Activity Codes (FIN_xx Mapping)

| Keyword in Suspicion Basis | FINnet Code |
|---------------------------|-------------|
| "cash" | FIN_01 |
| "structured" | FIN_02 |
| "unusual" | FIN_03 |
| "round" | FIN_04 |
| "fraud" | FIN_05 |
| "terror" | FIN_06 |
| "sanctions" | FIN_07 |
| *(other)* | FIN_99 |

---

*Review trigger: any change to STR workflow, retention policies, or FINnet schema updates from FIU-IND.*
