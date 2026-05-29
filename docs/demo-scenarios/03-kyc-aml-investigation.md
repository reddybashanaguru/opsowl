# Demo Scenario 3 — KYC / AML Investigation
### Targeted at: Any NBFC, UCB, or Mid-Sized Bank with a Compliance / FIU Desk

---

## Executive Summary

A mid-sized NBFC with ₹800 crore AUM runs a routine AML transaction monitoring batch on its savings and current account portfolio. The batch flags nine accounts at three different branches that collectively received ₹89.6 lakh in cash deposits over 47 days — all in tranches of ₹9–9.9 lakh, deliberately staying below the ₹10 lakh Cash Transaction Report (CTR) threshold. Four of the nine accounts were opened within the previous 90 days, share an introducer, and show near-zero credit histories. Within 72 hours of each deposit, funds are transferred onward to a single aggregator account at a private sector bank. This demo shows how OpsOwl AI runs the full AML investigation — transaction pattern analysis, KYC gap detection, network mapping, and STR drafting — in a single working day, without a dedicated transaction monitoring platform.

> **Real-world grounding:** Structuring ("smurfing") to avoid CTR thresholds is one of the top patterns reported by FIU-IND. In FY 2022-23, 4,36,287 STRs were filed with FIU-IND — structured cash deposits and dormant-account-activation schemes account for a disproportionate share. This scenario mirrors the "deposit mule network" variant documented in RBI's Master Circular on KYC (2016, updated 2023) and FinCEN advisory SAC-2014-A009 (same pattern, globally common).

---

## Background — AML Obligations for NBFCs

Under the Prevention of Money Laundering Act (PMLA) 2002 and RBI Master Direction on KYC (updated Jan 2023), every NBFC that accepts deposits is required to:

1. **File a CTR** with FIU-IND for cash transactions ≥ ₹10 lakh in a single day
2. **File an STR** within 7 days of forming a suspicion of money laundering — no minimum amount threshold
3. Maintain KYC records with Customer Due Diligence (CDD) documentation
4. Identify the **Beneficial Owner** for any customer with > 25% ownership stake
5. Apply **Enhanced Due Diligence (EDD)** for Politically Exposed Persons and high-risk categories

The absence of an automated transaction monitoring system (common in smaller NBFCs) creates a manual review gap that this scenario exploits — and which OpsOwl partially fills via AI analysis over uploaded transaction data.

---

## The Fraud Pattern — "Structured Deposit Mule Network"

| Element | Detail |
|---------|--------|
| Account cluster | 9 accounts, 3 branches (Andheri West, Thane, Navi Mumbai) |
| Deposit pattern | 27 cash deposits over 47 days; each between ₹9,00,000 and ₹9,90,000 |
| Total deposited | ₹89,60,000 across the cluster |
| CTR threshold avoided | Every single deposit is below ₹10,00,000 — zero CTRs triggered automatically |
| Account age | 4 of 9 accounts opened < 90 days before activity began |
| Common introducer | "Manish Trading Solutions" — same DSA introducer code on all 9 accounts |
| Onward transfer | ₹86,20,000 transferred within 72 hours to a single HDFC current account (AC: XXXXXX7843) |
| Destination account KYC | Rajesh Enterprises Pvt Ltd — no verifiable business presence |
| Account profiles | Salaried individuals aged 22–28; monthly salary credits ₹18,000–₹22,000; inconsistent with ₹9.5 lakh deposit volumes |
| Status at case open | Accounts active; 4 accounts still receiving deposits |

---

## Trigger Event — How the Case Opens

**Date: 18 August 2025, 9:00 AM**

Compliance Officer Arjun Mehta runs the monthly AML batch on the current month's transaction export (CSV from the core banking system). The batch is a simple rule: *flag any account with cumulative cash credits between ₹8 lakh and ₹10 lakh in a rolling 30-day window.*

The rule fires on 9 accounts. Arjun opens OpsOwl.

```
Case: AML-2025-0318
Title: Structured Cash Deposit Pattern — Andheri/Thane/Navi Mumbai Cluster
Priority: High
Assigned to: Arjun Mehta (Compliance Analyst)
SLA: 7 days (STR deadline: Day 7 per PMLA Section 12)
Tags: aml, structuring, mule-accounts, mumbai, fiu-ind
```

---

## Investigation Walkthrough

### Step 1 — Transaction Data Upload

Arjun exports 3 months of transaction data for all 9 flagged accounts from the core banking system and uploads a single CSV (`flagged_accounts_txn_aug2025.csv`, 847 rows).

**AI Document Intelligence processes it in 22 seconds:**
- Identifies 9 distinct account IDs
- Detects 27 cash credit transactions, all between ₹9,00,000 and ₹9,90,000
- Maps 24 outward transfers to the same destination IFSC + account within 72 hours of each deposit
- Reconstructs the deposit timeline across 47 days

**AI Copilot — Arjun's first question:**
> *"What is the distribution of cash deposit amounts across all 9 accounts?"*

**AI Response:**
> *"Across 27 cash credit transactions in the flagged cluster, amounts range from ₹9,00,000 to ₹9,90,000. The mean is ₹9,48,148. Zero transactions exceed ₹10,00,000. The narrow banding (all within a 10% corridor just below the CTR threshold) combined with the volume and frequency is the primary structuring indicator. A legitimate cash business would show natural variance above and below round numbers; this cluster shows systematic avoidance of the ₹10 lakh threshold with statistical confidence."*

---

### Step 2 — Account Profile vs. Transaction Mismatch

Arjun uploads the KYC summary extract for all 9 accounts (`kyc_extract_9accounts.pdf`).

**AI Document Intelligence:**

| Account | Holder | Age | Declared Occupation | Monthly Salary Credit | 30-day Cash Deposits |
|---------|--------|-----|--------------------|-----------------------|----------------------|
| AC-0041 | Rahul Desai | 24 | Data Entry Operator | ₹18,500 | ₹28,40,000 (3 deposits) |
| AC-0042 | Pooja Naik | 22 | Student | ₹0 | ₹19,00,000 (2 deposits) |
| AC-0043 | Suresh Patil | 27 | Delivery Executive | ₹21,000 | ₹9,70,000 (1 deposit) |
| AC-0044 | Anita Rao | 25 | Housewife | ₹0 | ₹19,40,000 (2 deposits) |
| AC-0045 | Kiran Shinde | 28 | Auto Driver | ₹22,000 | ₹9,00,000 (1 deposit) |
| AC-0046 | Deepa Joshi | 23 | Part-time Work | ₹9,000 | ₹9,90,000 (1 deposit) |
| AC-0047 | Vinod Kumar | 26 | Sales Staff | ₹20,500 | ₹9,50,000 (1 deposit) |
| AC-0048 | Meena Gaikwad | 24 | Homemaker | ₹0 | ₹19,30,000 (2 deposits) |
| AC-0049 | Ravi Tiwari | 22 | Fresher/Unemployed | ₹0 | ₹9,90,000 (1 deposit) |

**AI Copilot:**
> *"Nine of nine accounts show a fundamental inconsistency between declared income and deposit volumes. For accounts AC-0041, AC-0042, AC-0044, and AC-0048, the cash deposits in a single month exceed 12 months of declared income. Four accounts (AC-0042, AC-0044, AC-0048, AC-0049) declare zero income — these are the highest-risk profiles for mule account recruitment. Under RBI KYC Master Direction Para 38, this is a Category A suspicious transaction indicator: 'transactions inconsistent with the customer's known legitimate business or personal activities or financial standing.'"*

---

### Step 3 — Introducer and Account Opening Pattern

Arjun uploads account opening records for all 9 accounts (`account_opening_forms_batch.pdf`, 76 pages).

**AI Document Intelligence:**

- All 9 accounts list **"Manish Trading Solutions" (DSA Code: MTS-0892)** as the introducing agent
- Account opening dates:
  - AC-0041, AC-0042, AC-0043, AC-0044 opened on **14 May 2025** (same day, Andheri branch — 4 accounts in one session)
  - AC-0045, AC-0046 opened on **17 May 2025** (Thane branch)
  - AC-0047, AC-0048, AC-0049 opened on **22 May 2025** (Navi Mumbai branch)
- All 9 accounts were opened within an 8-day window
- Address on AC-0042, AC-0044, AC-0048 resolves to the same building (Shivaji Nagar, Thane) with different flat numbers
- Photo IDs on 3 accounts (AC-0043, AC-0046, AC-0049) show the same background — consistent with photographs taken at the same location in a single session

**AI Copilot:**
> *"Batch account opening via a single DSA across three branches within 8 days, combined with shared building addresses and photographic evidence of a single-session KYC photo shoot, is the hallmark of a mule account factory. The DSA 'Manish Trading Solutions' (MTS-0892) should be suspended and referred to the Branch Risk team immediately. This pattern matches FIU-IND advisory FIU/ADV/2023/04 on organised mule account recruitment."*

**Network finding added to Timeline automatically.**

---

### Step 4 — Onward Transfer Destination Analysis

Arjun queries: *"Who is the beneficiary of the outward transfers?"*

**AI Response (from transaction CSV analysis):**
> *"24 of 27 NEFT/IMPS outward transfers route to a single account: HDFC Bank current account XXXXXX7843, registered to Rajesh Enterprises Pvt Ltd, IFSC HDFC0000712 (Mumbai Central branch). The remaining 3 transfers route to two UPI VPA handles (@ybl). The concentration of ₹86.2 lakh (96% of total deposits) into one beneficiary within 72 hours of receipt is a textbook layering indicator — the aggregator account is likely the second hop in a three-stage money laundering cycle: placement (mule cash deposits) → layering (NEFT to aggregator) → integration (unknown final destination)."*

Arjun pulls the MCA filing for Rajesh Enterprises Pvt Ltd and uploads it.

**AI Document Intelligence:**
- Incorporation: March 2025 (2 months old at time of first transfer)
- Directors: Rajesh Subramaniam (DIN: 09124731) — single director, no prior company history
- Registered address: Virtual office, Mumbai Central
- Declared business: "Import-Export of Agricultural Products" — inconsistent with receiving ₹86 lakh NEFT credits from retail savings accounts

---

### Step 5 — Communications Evidence

Arjun obtains WhatsApp messages from one cooperative account holder (Rahul Desai, AC-0041) who was contacted by the branch manager.

**AI Communications Entity Extraction (PDF of exported chat):**

| Entity | Extracted Value |
|--------|-----------------|
| Parties | "Rahul", "Manish bhai", unknown number (+91-98XXXXXXXX) |
| Amounts | ₹5,000, ₹8,000, ₹10,000 (commission references) |
| Instructions | "bas account mein paisa aayega, 3 din mein transfer kar dena", "koi bhi bank ko mat batana", "ek baar aur karo, last baar" |
| Dates | Message dates align with 7 of the 9 deposit events |
| Red-flag phrases | "account use karna hai kuch din ke liye", "tere naam pe kuch nahin hoga", "easy money" |

**AI Copilot:**
> *"'Bas account mein paisa aayega, 3 din mein transfer kar dena' (money will arrive in the account, transfer it within 3 days) combined with commission amounts and 'koi bhi bank ko mat batana' (don't tell any bank) is unambiguous recruitment of a witting or semi-witting mule. The recruiter (Manish bhai) matches the DSA introducer 'Manish Trading Solutions'. Rahul Desai may qualify as a cooperative witness — his statement should be recorded before the STR is filed."*

---

### Step 6 — Risk Re-Score and Recommended Actions

**AI Risk Analysis:**

```
Risk Score:  96 / 100   ●●●●●  CRITICAL
Risk Level:  CRITICAL

Risk Factors:
  [CRITICAL] Structuring confirmed — 27 deposits, all ₹9.0–9.9L, zero above CTR threshold
  [CRITICAL] 96% of deposits transferred to single 2-month-old shell company within 72h
  [CRITICAL] Batch KYC opening — 9 accounts, 1 DSA, 8 days, 3 branches
  [CRITICAL] Account profile mismatch — deposits 12–18× annual declared income
  [HIGH]     Communication evidence confirms mule recruitment by DSA introducer
  [HIGH]     Same-building address across 3 account holders (flat-number variation)
  [HIGH]     Shell company beneficiary — virtual office, no business history
  [MEDIUM]   Photo artifact — same-session photographs on 3 accounts

Recommended Actions:
  1. File STR with FIU-IND immediately (Day 0 suspicion → Day 7 deadline)
  2. Freeze outgoing transactions on all 9 accounts pending investigation
  3. Suspend DSA MTS-0892 (Manish Trading Solutions) and block new account openings
  4. File police complaint under PMLA Section 3 + 4 (money laundering offences)
  5. Report Rajesh Enterprises Pvt Ltd IFSC/account to FIU-IND as suspected aggregator
  6. Interview Rahul Desai (AC-0041) — cooperative witness
  7. Escalate to Principal Officer for mandatory reporting sign-off
  8. Review all accounts opened via DSA MTS-0892 in the last 12 months
```

---

### Step 7 — STR Generation

Arjun's manager, Sunita Krishnan (Principal Officer), clicks **Generate STR** from the case action bar.

**OpsOwl generates a goAML-compatible STR draft:**

```
Suspicious Activity:  Structured cash deposits to avoid CTR threshold —
                      organised mule account network with layering to
                      shell company beneficiary

Reporting Entity:     [NBFC Name], RE Code: [FIU Code]
Report Date:          19 August 2025
Case Reference:       AML-2025-0318
7-Day STR Deadline:   25 August 2025

Subject Accounts:     9 (AC-0041 through AC-0049)
Subject Individual:   Manish Trading Solutions DSA (MTS-0892) — suspected organiser
Beneficiary Entity:   Rajesh Enterprises Pvt Ltd (CIN: U01100MH2025PTC123456)
                      HDFC Bank XXXXXX7843

Total Suspicious Amt: ₹89,60,000 (deposits); ₹86,20,000 (onward transfers)
Period:               02 July 2025 – 17 August 2025 (47 days)

Grounds for Suspicion:
  — 27 structured cash deposits, all below ₹10L CTR threshold (classic smurfing)
  — Accounts opened in coordinated batch via single DSA across 3 branches in 8 days
  — Account profiles inconsistent with deposit volumes (0–22K declared income)
  — 96% of deposits onward transferred to 2-month-old shell company within 72 hours
  — WhatsApp communication evidence of mule recruitment by DSA introducer
  — Same-building address overlap across 3 account holders

Regulatory References:
  — PMLA 2002, Section 12 (Reporting obligation)
  — PMLA 2002, Section 3/4 (Money laundering offence — predicate)
  — RBI Master Direction on KYC 2023, Para 38 (Suspicious transaction indicators)
  — FIU-IND Advisory FIU/ADV/2023/04 (Mule account networks)
```

Sunita reviews, adds the Principal Officer digital signature field, and marks the case: **STR Filed**.

The system records the filing timestamp in the hash-chain audit log, satisfying the PMLA 7-day window.

---

## Investigation Timeline

| Time | Event |
|------|-------|
| T+0h | Case AML-2025-0318 opened; transaction CSV uploaded (847 rows) |
| T+0h 22m | AI flags structuring pattern — 27 deposits all below ₹10L threshold |
| T+1h | KYC extract uploaded; income vs. deposit mismatch confirmed across all 9 accounts |
| T+2h | Account opening forms uploaded; batch DSA + same-session photo artifacts detected |
| T+3h | MCA data uploaded; Rajesh Enterprises confirmed as 2-month-old shell |
| T+5h | WhatsApp chat from cooperative account holder uploaded; mule recruitment confirmed |
| T+6h | AI Risk Score: 96/100 CRITICAL; recommended actions listed |
| T+7h | Manager review — escalated to Principal Officer |
| T+8h | STR drafted (goAML XML + PDF preview) |
| T+10h | All 9 accounts frozen; DSA MTS-0892 suspended |
| T+24h | STR filed with FIU-IND; police complaint referral to local cyber/economic cell |
| T+48h | Branch review of all MTS-0892 originated accounts (last 12 months) initiated |

*PMLA deadline: Day 7. Actual filing: Day 1. Without OpsOwl, the same analysis across 3 branch silos, 847 transaction rows, 76 KYC pages, and an MCA lookup typically takes 4–6 days of analyst time — leaving < 48 hours to draft and approve the STR.*

---

## Demo Talking Points for the Prospect

**For a Compliance Head / Principal Officer at any NBFC:**

> "Your CTR rule fires at ₹10 lakh. The structuring pattern fires at ₹9.9 lakh — nine times, across three branches, over 47 days. Without a system connecting those transactions, each one looks like a routine cash deposit. OpsOwl connects the nine accounts in 22 seconds because it sees the pattern across all the data at once, not one account at a time."

> "The PMLA STR deadline is 7 days from forming a suspicion. Manual investigations in a 3-branch case routinely eat 4–5 of those days just pulling data. OpsOwl compresses that to under a day, leaving you time to actually review and sign off rather than racing the clock."

> "You're on the hook for the STR personally as Principal Officer. OpsOwl doesn't just flag — it drafts the goAML XML with the regulatory references pre-filled. You review and sign. That's the product: it does the analyst work so you can do the compliance work."

---

## OpsOwl Features Showcased

| Feature | Where Used |
|---------|-----------|
| Transaction CSV ingestion + pattern detection | 847-row core banking export — structuring pattern in 22 seconds |
| AI Copilot (Q&A over data) | Deposit distribution analysis, profile mismatch query, beneficiary query |
| Document Intelligence (PDF OCR) | KYC extract, account opening forms (76 pages), MCA filing |
| Structuring / threshold-avoidance detection | All 27 deposits in ₹9.0–9.9L band identified and flagged |
| Account profile inconsistency analysis | Income vs. deposit volume mismatch across 9 profiles |
| Batch KYC opening pattern | Same DSA, same-session photos, shared address detected |
| Network mapping (onward transfer destination) | Single aggregator account identified; shell company confirmed |
| Communications entity extraction | WhatsApp — parties, amounts, instructions, red-flag phrases |
| AI Risk Scoring (96/100 CRITICAL) | Automated risk level + prioritised recommended actions |
| STR Generation (goAML XML) | Multi-account structured deposit STR with all regulatory references |
| Hash-chain audit log | Filing timestamp recorded in tamper-evident audit trail |
| Timeline (auto-populated) | Full 24-hour investigation audit trail visible to Principal Officer |
| SLA countdown | PMLA 7-day deadline visible throughout investigation |
| Case status chain | Open → In Progress → Escalated → STR Filed |
