# Demo Scenario 1 — Gold Loan Fraud
### Targeted at: Muthoot Finance, Manappuram Finance, IIFL Gold, Rupeek-style NBFCs

---

## Executive Summary

A branch-level appraiser at a multi-city gold loan NBFC has been certifying fake and mixed-purity gold ornaments as hallmarked 22-karat gold across three branches for 8 months. Borrowers linked to the appraiser received ₹4.87 crore in loans against fraudulent collateral. The fraud surfaces when an internal auditor at the Hyderabad Regional Office flags unusually high disbursal-to-collateral-value ratios at the Kukatpally branch. This demo shows how an investigator uses OpsOwl AI to connect 23 accounts across three branches, rebuild the appraiser's network, surface the pattern, and file a Suspicious Transaction Report to FIU-IND — all without leaving a single screen.

> **Real-world grounding:** Mirrors the ICICI Bank Nagpur racket (₹23 crore, 152 account holders, 7 appraisers, 2024–25) and the Jajpur SBI case (105 accounts, 31 fake gold, Oct 2024–Jul 2025).

---

## Background — Gold Loan NBFC Context

Gold loan NBFCs disburse loans at 65–75% LTV against pledged gold ornaments. The valuation step — performed by a licensed branch appraiser using a touchstone, acid test, or XRF gun — is the only physical control between the borrower and the cash. When an appraiser is compromised, the entire collateral stack collapses.

**Why this fraud is hard to catch without a system:**
- Each branch runs its own loan book; cross-branch data sits in different Excel exports
- The appraiser's name appears on hundreds of legitimate vouchers — the fraudulent ones look identical
- Borrowers are often real people (real PAN, real Aadhaar) brought in by the racket organiser
- The gold is real at inspection but gets swapped or is low-purity alloy certified as hallmarked 22K

---

## The Fraud Pattern

| Element | Detail |
|---------|--------|
| Appraiser | Rajan Pillai — 6-year employee, Kukatpally branch, licensed by BIS |
| Branches involved | Kukatpally (HYD), Miyapur (HYD), Secunderabad (HYD) |
| Period | October 2024 – June 2025 (8 months) |
| Accounts | 23 borrower accounts; 19 fictitious / recruited; 4 genuine customers used as cover |
| Collateral fraud | Gold ornaments certified as 22K; actual purity 14–16K on external re-assay |
| Loan amount range | ₹8 lakh – ₹38 lakh per account |
| Total exposure | ₹4.87 crore |
| Kickback structure | Rajan receives ₹500–₹1,500 per ₹1 lakh disbursed via UPI to a family member |
| Organiser | Suresh Khatri — unlicensed loan agent with no formal role; connected to 18 of 23 borrowers |

---

## Trigger Event — How the Case Opens

**Date: 15 July 2025, 10:42 AM**

Regional Audit Head Meera Iyer runs the quarterly branch disbursal report. The Kukatpally branch shows an average disbursal-to-ornament-count ratio of 3.8× the regional mean for Q1 FY26. She emails the Fraud Risk team. An analyst, Divya Reddy, opens OpsOwl and creates a new case.

```
Case: FRD-2025-0742
Title: Kukatpally Branch — Anomalous Gold Loan Disbursal Ratio (Q1 FY26)
Priority: High
Assigned to: Divya Reddy (Senior Analyst)
SLA: 7 days
Tags: gold-loan, branch-fraud, appraiser, hyderabad
```

---

## Investigation Walkthrough

### Step 1 — Upload the Audit Report and Let AI Extract the Data

Divya uploads the 47-page Quarterly Branch Audit Report (PDF) to the case.

**OpsOwl Document Intelligence processes it in 38 seconds:**
- Extracts 23 loan account numbers flagged in the anomaly table
- Identifies appraiser name "Rajan Pillai" appearing on 21 of 23 certification vouchers
- Extracts disbursal amounts, ornament weights, and claimed purity for each account

**AI Copilot — Divya's first question:**
> *"What is the average claimed gold purity across the flagged accounts compared to the branch average?"*

**AI Response:**
> *"The 23 flagged accounts show an average claimed purity of 22.0 karats — exactly at the maximum LTV-eligible threshold. The Kukatpally branch baseline across non-flagged accounts is 20.6 karats. Consistently claiming maximum purity is a documented red flag in gold loan fraud (RBI Circular DOR.CRE.REC.No.81/21.04.048/2025). Appraiser Rajan Pillai certified 21 of the 23 accounts."*

---

### Step 2 — Cross-Branch Account Lookup

Divya adds two sub-tasks to the case:

1. Pull all accounts certified by Rajan Pillai across all branches (past 12 months)
2. Cross-check borrower phone numbers and Aadhaar suffixes across those accounts

She exports the account list from the LMS, re-uploads as `rajan_pillai_all_accounts.csv` (312 rows).

**AI Copilot:**
> *"Across 312 accounts certified by Rajan Pillai, 23 share phone numbers with accounts at Miyapur and Secunderabad branches. Specifically: 7 accounts at Miyapur and 4 at Secunderabad have phone numbers or Aadhaar last-4 digits matching accounts at Kukatpally. This is statistically anomalous — the expected cross-branch phone overlap for legitimate customers is near zero."*

**Network finding added to the Timeline automatically.**

---

### Step 3 — Borrower Profile Analysis

Divya uploads KYC scans for the 18 Suresh Khatri-linked borrowers (Aadhaar + PAN PDFs, 36 documents).

**Document Intelligence flags:**
- 9 of 18 Aadhaar cards share the same font rendering artifact in the date field — consistent with a single batch print job
- 4 PAN cards have identical metadata timestamps in the embedded PDF (suggesting the same scanner/device)
- Address on 6 Aadhaar cards resolves to the same building (Sri Nagar Colony, Hyderabad) under different flat numbers

**AI Copilot:**
> *"6 borrowers claim different flat numbers at the same building (H.No. 8-2-293/A, Sri Nagar Colony). This is a known mule recruitment pattern — a single address is used with flat-number variations to make borrowers appear distinct in KYC checks. Recommend cross-referencing this address with your loan origination system for all active accounts."*

---

### Step 4 — Communications Review

Divya uploads WhatsApp chat exports (PDF) seized by HR from Rajan Pillai's work phone following an initial HR inquiry.

**AI Communications Entity Extraction:**

| Entity | Extracted Value |
|--------|-----------------|
| Parties | Rajan Pillai, "SK" (Suresh Khatri), "Mama" (unidentified) |
| Amounts mentioned | ₹1,500, ₹2,000, ₹800, "same rate as last time" |
| Dates referenced | Oct 14, Nov 3, Dec 8, Jan 22, Mar 17 — all days before disbursals |
| Red-flag phrases | "materi ready hai", "fresh piece lao", "same stamp lagao", "client confirm karo pehle" |
| Account numbers | 4 partial account numbers matching flagged loan IDs |

**AI Copilot:**
> *"The phrase 'same stamp lagao' (apply the same stamp) combined with pre-disbursal timing and cash amounts matching Rajan Pillai's estimated kickback rate (₹800–₹1,500 per ₹1 lakh) strongly suggests coordination of collateral certification. 'Materi ready hai' (material is ready) likely refers to ornaments prepared for presentation. This communication pattern is consistent with organized appraiser fraud."*

---

### Step 5 — Risk Scoring

Divya triggers **AI Risk Analysis** from the case header.

**Risk Assessment Output:**

```
Overall Risk Score:   94 / 100   ●●●●●  CRITICAL
Risk Level:           CRITICAL

Risk Factors:
  [CRITICAL] Appraiser-linked cross-branch account cluster — 23 accounts, 3 branches
  [CRITICAL] Aadhaar batch-print artifacts across 9 KYC documents
  [HIGH]     Consistent 22K purity claims at LTV ceiling — statistical outlier
  [HIGH]     Pre-disbursal communication pattern with cash kickback markers
  [HIGH]     Shared residential address across 6 borrowers (mule pool indicator)
  [MEDIUM]   4 PAN cards with identical scan metadata

Recommended Actions:
  1. File STR with FIU-IND under PMLA Section 12 — potential organized fraud
  2. Freeze disbursals from Rajan Pillai appraiser ID pending investigation
  3. Refer 23 accounts for physical gold re-assay (external metallurgical lab)
  4. Escalate to ED / local police for Suresh Khatri's background check
  5. Notify branch managers at Miyapur and Secunderabad for parallel HR action
```

---

### Step 6 — Status Escalation

Divya moves the case status: `In Progress → Pending Review`

Manager Ravi Kumar reviews the timeline, reads the AI summary, and escalates: `Pending Review → Escalated`

**Reason captured:** *"Cross-branch fraud confirmed. Escalating to CISO and Branch Risk Committee. Initiating physical re-assay of all 23 pledged ornaments. Preparing STR."*

---

### Step 7 — STR Generation

Ravi clicks **Generate STR** from the case action bar.

**OpsOwl generates a goAML-compatible STR draft (XML + PDF preview):**

```
Reporting Entity:  [NBFC Name] — RE Code: [FIU Code]
Date of Report:    17 July 2025
Case Reference:    FRD-2025-0742

Subject:           Organized gold loan collateral fraud — appraiser collusion
                   across 3 branches, 23 accounts, ₹4.87 crore exposure

Suspicious Transactions Listed:
  — 23 loan disbursals, Oct 2024 – Jun 2025
  — Appraiser: Rajan Pillai (EMP-2019-0341)
  — Organiser (suspected): Suresh Khatri (PAN: BXYZK4521F)

Grounds for Suspicion:
  — Clustered KYC documents with batch-print artifacts
  — Consistent maximum-purity certification across statistical outlier accounts
  — Pre-disbursal encrypted communications with kickback amounts
  — Cross-branch borrower phone number overlap (6 confirmed matches)

Regulatory References:
  — PMLA 2002, Section 12 (Reporting obligation)
  — RBI Master Direction on KYC, Para 38 (Suspicious transaction indicators)
  — RBI DOR.CRE.REC.No.81/21.04.048/2025 (Gold loan purity documentation)
```

Ravi reviews, adds the Principal Officer's digital signature field, and marks the case status: **STR Filed**.

---

## Investigation Timeline (Compressed)

| Time | Event |
|------|-------|
| T+0h | Case FRD-2025-0742 opened by Divya Reddy |
| T+1h | Audit report uploaded; AI extracts 23 accounts and Rajan Pillai link |
| T+3h | Cross-branch CSV analysis — 23 overlapping accounts confirmed |
| T+6h | KYC document analysis — Aadhaar batch-print artifact detected |
| T+9h | WhatsApp chat analysis — kickback pattern confirmed |
| T+11h | AI Risk Score: 94/100 CRITICAL |
| T+13h | Escalated to CISO and Branch Risk Committee |
| T+24h | STR filed with FIU-IND |
| T+48h | Physical re-assay of 23 pledged ornaments initiated |
| T+72h | Rajan Pillai suspended; HR inquiry opened |

*Without OpsOwl: same investigation typically takes 3–4 weeks across email threads, Excel files, and manual document review.*

---

## Demo Talking Points for the Prospect

**For a Muthoot/Manappuram Fraud Risk Head:**

> "Your appraisers are the single point of failure in your collateral model. OpsOwl doesn't replace your appraiser — it watches the pattern across every appraiser, every branch, every month. When one appraiser's LTV distribution looks like an outlier, the system tells you before the borrowers default or the gold disappears."

> "The Nagpur ICICI case ran for 22 months before it was caught. The same pattern — multiple branches, one mastermind — would have been flagged by OpsOwl in the first quarter based purely on cross-branch phone number overlap and consistent maximum-purity certification."

---

## OpsOwl Features Showcased

| Feature | Where Used |
|---------|-----------|
| Document Intelligence (PDF OCR) | Audit report extraction, KYC batch analysis |
| AI Copilot (Q&A) | Purity ratio analysis, address pattern query |
| Communications Entity Extraction | WhatsApp chat — parties, amounts, red-flag phrases |
| Cross-document pattern detection | Aadhaar print-artifact fingerprinting across 9 docs |
| AI Risk Scoring (94/100) | Automated risk level + recommended actions |
| Cross-branch CSV analysis | 312-row account data cross-referenced |
| STR Generation (goAML XML) | One-click draft with all regulatory references |
| Timeline (live, auto-populated) | Full 24-hour audit trail |
| Case escalation with captured reason | Escalated → STR Filed status chain |
| SLA countdown | 7-day SLA visible on case header throughout |
