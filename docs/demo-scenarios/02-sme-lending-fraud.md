# Demo Scenario 2 — SME Lending Fraud
### Targeted at: Lendingkart, Aye Finance, Ugro Capital, Capri Global, Five-Star Business Finance

---

## Executive Summary

A Pune-based textile trading company applies for a ₹75 lakh working capital loan. The credit team approves it based on GST returns showing ₹3.2 crore annual turnover. Three months post-disbursal, the EMI bounces. A fraud analyst pulling the pre-closure review discovers that the borrower's GST filings were inflated — real turnover was ₹38 lakh — and that the same director runs three other entities that collectively received ₹2.1 crore in loans from different NBFCs in the previous six months. This demo shows how OpsOwl AI reconstructs the fraud network, connects the dots across four companies, and prepares both a police complaint annexure and an FIU-IND STR in under a day.

> **Real-world grounding:** Reflects the GST-to-banking discrepancy pattern (Precisa AML research, 2025), multi-lender loan-stacking, and the "director-as-common-thread" shell company technique documented in MSME fraud alerts (Ministry of MSME Fraud Alert, 2021, pattern still active in 2025).

---

## Background — SME Lending NBFC Context

Digital SME lenders like Lendingkart and Aye Finance underwrite loans in 48–72 hours using account aggregator data, GST filings, ITR, and bureau scores — with little or no physical visit. The speed that makes them competitive also creates the attack surface. Fraudsters know that:

1. GST Portal data is self-reported and not real-time verified against actual sales
2. Bank statements can be fabricated or show round-tripped cash to inflate apparent revenue
3. Credit bureaus take 30–45 days to update new loan enquiries — enough time to stack loans across multiple lenders before any bureau warning appears
4. A single director can control 4–6 proprietorship/partnership/private limited entities with different names and PAN numbers

---

## The Fraud Pattern — "Director Stacking"

| Element | Detail |
|---------|--------|
| Primary borrower | Vijay Textiles Trading Pvt Ltd, Pune |
| Director / KMP | Vijay Ramakrishnan (PAN: DXVIJ7834K) |
| Loan applied | ₹75 lakh working capital, tenor 24 months |
| Claimed GST turnover | ₹3.2 crore (GSTR-1 FY25) |
| Actual bank statement credits | ₹38 lakh (annualised from 6 months of statements) |
| Connected entities (same director) | Ramakrishnan Fabrics LLP; VR Exports India; Sai Weave Traders |
| Loans stacked across lenders | ₹2.1 crore (₹40L + ₹65L + ₹80L + ₹26L across 4 entities at 3 NBFCs) |
| Round-tripping evidence | ₹28 lakh transferred from VTTPL to VR Exports, then returned 11 days later as "trade receivable" — inflating VTTPL bank statement credits |
| Status at case open | EMI 1 bounced (Day 91); borrower uncontactable |

---

## Trigger Event — How the Case Opens

**Date: 22 August 2025, 2:15 PM**

Collections analyst Priya Sharma flags EMI bounce for loan LN-2025-PUN-3847. The collections system auto-creates a delinquency record. Per the NBFC's policy, any bounce in the first 3 EMIs triggers a Fraud Review Case. Priya opens OpsOwl.

```
Case: FRD-2025-1194
Title: Vijay Textiles Trading Pvt Ltd — Early EMI Default + Fraud Review
Priority: High
Assigned to: Priya Sharma (Fraud Analyst)
SLA: 5 days
Tags: sme-lending, early-default, gst-mismatch, pune
```

---

## Investigation Walkthrough

### Step 1 — Document Package Upload

Priya uploads the full loan file as a ZIP (auto-extracted by OpsOwl):
- `GSTR1_FY25_VijayTextiles.pdf` (GST returns)
- `BankStatement_6M_VijayTextiles.pdf` (HDFC Bank)
- `ITR_FY24_VijayTextiles.pdf`
- `CreditBureau_CIBIL_VijayTextiles.pdf`
- `KYC_VijayRamakrishnan.pdf` (Aadhaar + PAN)
- `MOA_VijayTextilesTradingPvtLtd.pdf` (Memorandum of Association)

**AI Risk Analysis triggered automatically on upload (48 seconds):**

```
Risk Score:  81 / 100   ●●●●○  HIGH
Risk Level:  HIGH

Preliminary Findings:
  [CRITICAL] GST declared turnover (₹3.2 Cr) vs. bank statement credits
             (₹38 lakh annualised) — 8.4× discrepancy
  [HIGH]     Bank statement shows 3 inward transfers from "VR Exports India"
             totalling ₹28 lakh — entity name matches a known associated firm
  [HIGH]     CIBIL report shows 4 loan enquiries in the last 6 months across
             3 lenders (enquiry dates: 12 Feb, 28 Feb, 15 Mar, 2 Apr 2025)
  [MEDIUM]   ITR declared income (₹18 lakh) inconsistent with loan repayment
             capacity for ₹75 lakh loan
```

---

### Step 2 — GST vs. Bank Statement Deep Dive

**AI Copilot:**
> *"Can you break down the GST turnover vs. bank credits month by month?"*

**AI Response:**
> *"GSTR-1 for FY25 shows monthly sales of ₹22–30 lakh. Bank statement credits for the same period average ₹3.2 lakh per month. The gap is consistent across all 12 months — this is not a seasonal mismatch. Notably, 4 months show zero bank credits despite GSTR-1 reporting sales of ₹24–27 lakh. This pattern is consistent with inflated GST filing without corresponding actual sales receipts."*

**AI Copilot:**
> *"What are the top 5 credit entries in the bank statement and who are the counterparties?"*

**AI Response:**
> *"Top 5 credits:*
> *1. ₹14,20,000 — VR Exports India (07-Nov-2024)*
> *2. ₹8,40,000 — VR Exports India (19-Jan-2025)*
> *3. ₹5,60,000 — VR Exports India (03-Mar-2025)*
> *4. ₹3,80,000 — Sai Weave Traders (12-Feb-2025)*
> *5. ₹2,10,000 — Cash deposit (28-Dec-2024)*
>
> *Three of the top 5 credits are from VR Exports India — also named in the MOA as a related party. These transfers inflate the bank statement balance without representing genuine trade revenue. This is a documented round-tripping indicator."*

---

### Step 3 — Director Network Reconstruction

Priya pulls MCA filings for Vijay Ramakrishnan (Director DIN: 08723419) from the MCA portal and uploads the extracted company master data PDF.

**AI Document Intelligence:**

| Entity | Type | DIN Match | Incorporation | Status |
|--------|------|-----------|---------------|--------|
| Vijay Textiles Trading Pvt Ltd | Private Ltd | Yes | Mar 2022 | Active |
| Ramakrishnan Fabrics LLP | LLP | Yes (Designated Partner) | Jan 2021 | Active |
| VR Exports India | Proprietorship | Yes (same PAN) | Aug 2020 | Active |
| Sai Weave Traders | Partnership | Yes (Partner) | Nov 2019 | Active |

**AI Copilot:**
> *"These four entities share a director/proprietor. VR Exports India and Sai Weave Traders are also counterparties in Vijay Textiles' bank statement. What does this mean for the round-tripping hypothesis?"*

**AI Response:**
> *"When an entity's bank credits predominantly originate from other entities controlled by the same director, those credits represent intra-group transfers rather than genuine sales receipts. The ₹28 lakh from VR Exports India to VTTPL — returned 11 days later as ₹26 lakh to 'Sai Weave Traders' per the bank statement — is a textbook round-trip. The net effect was to show ₹28 lakh of apparent bank credits during the loan underwriting window, inflating the apparent revenue by ~74%."*

---

### Step 4 — Multi-Lender Stacking Analysis

Priya adds a task: *"Request CIBIL detailed report + check bureau for all 4 entities."*

The detailed CIBIL report (uploaded 2 hours later) shows:

| Lender | Entity | Loan Amount | Sanction Date | Status |
|--------|--------|-------------|---------------|--------|
| [This NBFC] | Vijay Textiles Trading Pvt Ltd | ₹75,00,000 | May 2025 | NPA (EMI 1 bounced) |
| Fintech NBFC A | Ramakrishnan Fabrics LLP | ₹65,00,000 | Apr 2025 | Active (DPD 0) |
| Fintech NBFC B | VR Exports India | ₹40,00,000 | Mar 2025 | Active (DPD 0) |
| PSB (Term Loan) | Sai Weave Traders | ₹26,00,000 | Feb 2025 | DPD 30 |
| **Total exposure** | **4 entities, 4 lenders** | **₹2,06,00,000** | **Feb–May 2025** | |

**AI Copilot:**
> *"The 4 enquiries appeared within a 79-day window (12 Feb – 2 May 2025). Bureau data typically takes 30–45 days to reflect new sanctions, creating a blind spot window that this borrower appears to have exploited deliberately. This is a documented loan-stacking technique. Recommend filing STR referencing all 4 entities and sharing intelligence with the other lending institutions per RBI's inter-NBFC fraud information sharing guidelines."*

---

### Step 5 — Communication Evidence

Priya adds a note to the case: *"No response on registered mobile (6 attempts). Courier sent to registered address — returned 'addressee not found'."*

She uploads the rejection note photo. The Timeline records it automatically.

**Collections team note (added via mobile):** *"Field visit to Pune plant address shows premises locked and shuttered. Neighbours say 'company left 2 months ago'."*

**AI Copilot:**
> *"Address abandonment within 91 days of disbursal — before the first EMI — combined with uncontactable borrower and locked premises is consistent with a planned bust-out fraud. The borrower likely never intended to repay. Recommend including this as a 'planned default' indicator in the STR."*

---

### Step 6 — AI Risk Re-Score (Post Full Investigation)

```
Risk Score:  97 / 100   ●●●●●  CRITICAL
Risk Level:  CRITICAL

Updated Risk Factors:
  [CRITICAL] Confirmed GST inflation — 8.4× declared vs. actual credits
  [CRITICAL] Round-tripping confirmed — ₹28L intra-group cycle pre-underwriting
  [CRITICAL] Multi-lender stacking — ₹2.06 Cr across 4 entities, 4 lenders, 79 days
  [CRITICAL] Premises abandoned; borrower uncontactable within 91 days of disbursal
  [HIGH]     Director controls all 4 borrowing entities (single beneficial owner)
  [HIGH]     All 4 CIBIL enquiries preceded by zero-credit-history entity creation

Recommended Actions:
  1. File STR with FIU-IND — organized multi-entity lending fraud
  2. File police complaint (IPC Section 420, 467, 468, 471) with cyber/economic cell
  3. Contact Fintech NBFC A and Fintech NBFC B via IBA fraud alert channel
  4. Initiate recovery proceedings — attach director's personal assets (SARFAESI)
  5. Flag Vijay Ramakrishnan PAN and DIN in internal negative list
```

---

### Step 7 — STR and Police Complaint Draft

**OpsOwl STR (goAML XML):**
```
Suspicious Activity:  Multi-entity director stacking — planned bust-out fraud
Reporting Entity:     [NBFC Name], RE Code: [FIU Code]
Subject Entities:     4 (Vijay Textiles Trading Pvt Ltd; Ramakrishnan Fabrics LLP;
                      VR Exports India; Sai Weave Traders)
Subject Individual:   Vijay Ramakrishnan (PAN: DXVIJ7834K, DIN: 08723419)
Total Suspicious Amt: ₹2,06,00,000
Grounds:              GST inflation (8.4× discrepancy); round-tripping; multi-lender
                      stacking within 79-day bureau blind-spot window; planned default
                      evidenced by premises abandonment before first EMI
```

**AI also generates a Police Complaint Annexure draft** covering:
- Chronology of fraud (Table of dates and acts)
- Documentary evidence list (15 items with document IDs from OpsOwl)
- Loss quantum per IPC Section 420
- Linked entities table

Manager Ravi marks case: **STR Filed + Legal Referral.**

---

## Investigation Timeline

| Time | Event |
|------|-------|
| T+0h | Case FRD-2025-1194 opened; loan file uploaded |
| T+1h | AI flags 8.4× GST discrepancy; CIBIL 4-enquiry pattern |
| T+2h | MCA director data uploaded; 4-entity network confirmed |
| T+4h | Round-tripping pattern confirmed via bank statement counterparty analysis |
| T+6h | Detailed CIBIL confirms ₹2.06 crore multi-lender stack |
| T+8h | Field visit confirms premises abandoned |
| T+10h | AI Risk Score: 97/100 CRITICAL |
| T+12h | STR drafted + police complaint annexure ready |
| T+18h | STR filed; legal team notified; IBA fraud alert raised |

*Typical without a system: 2–3 weeks for the same analysis, by which time recoverable assets are further dissipated.*

---

## Demo Talking Points for the Prospect

**For a Lendingkart / Aye Finance Credit Risk or Fraud Head:**

> "Your underwriting checks GST turnover — but the borrower filed the GST. OpsOwl doesn't just look at the number; it asks whether the bank statement credits match. When they don't by 8×, that's the investigation starting point, not the ending point."

> "The bureau blind spot is 30–45 days. This borrower stacked ₹2 crore across 4 lenders in 79 days. OpsOwl connects the entity network and the director fingerprint in minutes — not the 3 weeks it would take your team to make 6 phone calls and pull 4 separate bureau reports."

> "By the time you build this manually, the director has rented a new office under a fifth entity. The system closes that window."

---

## OpsOwl Features Showcased

| Feature | Where Used |
|---------|-----------|
| Document Intelligence (ZIP auto-extract) | Full loan file package processing |
| AI Risk Scoring on upload | Instant 81/100 HIGH score within 48 seconds |
| GST vs. bank statement comparison | Month-by-month discrepancy breakdown |
| Counterparty entity extraction | Identifying VR Exports India as intra-group |
| Director network reconstruction (MCA) | 4-entity network from uploaded MCA data |
| Round-tripping pattern detection | ₹28L cycle confirmed via AI analysis |
| Multi-lender stacking timeline | CIBIL enquiry window analysis |
| AI Risk Re-Score (97/100 CRITICAL) | Updated after full evidence set |
| STR Generation (goAML) | With multi-entity suspicious activity |
| Police Complaint Annexure | AI-generated chronology + evidence list |
| Timeline + task tracking | Field visit note, collections attempts |
| Case status chain | Fraud Review → Escalated → STR Filed + Legal |
