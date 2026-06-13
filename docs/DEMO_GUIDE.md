# OpsOwl AI — Complete Product Demo Guide
### Audience: CTO / Head of Compliance / Decision Maker (Non-Technical)
### Purpose: Full end-to-end product walkthrough for sales demo
### Prepared by: Product Team · Version 2.0 · June 2026

---

## HOW TO USE THIS GUIDE

This guide is written for the person doing the demo — a product manager or sales engineer showing OpsOwl to a potential client's CTO or Head of Compliance.

**The CTO you are presenting to:**
- Runs a mid-sized NBFC, bank, or fintech with 5–200 compliance staff
- Is responsible for regulatory compliance (RBI, FIU-IND, PMLA)
- Has probably seen a few compliance software tools before
- Will ask hard questions about data security, integration, and ROI
- Makes decisions based on: "Will this reduce my risk? Will this save my team time? Can I defend this to my board?"

**For every feature shown, this guide gives you:**
1. A simple, jargon-free explanation of what the feature does
2. The exact steps to demonstrate it in the application
3. A plain-English explanation of the business problem it solves
4. Real numbers to back up the value claim
5. The exact objection the CTO will raise — and how to respond

**Demo environment:**
```
Web app:  http://localhost:3001
Admin:    admin@opsowl.dev    / OpsOwl@2024
Manager:  ravi.kumar@opsowl.dev / OpsOwl@2024
Analyst:  divya.reddy@opsowl.dev / OpsOwl@2024
```

**Total demo time:** 60–90 minutes (or do it in 30 minutes using only the starred ★ sections)

---

## BEFORE THE DEMO — SET THE STAGE (5 minutes)

Before you open the laptop, ask the CTO these three questions. Their answers will tell you which parts of the demo to spend the most time on.

**Question 1:** "How many compliance alerts does your transaction monitoring system generate per month?"
- If they say 500+ → spend more time on Alert Queue and Bulk Operations
- If they say <100 → spend more time on AI features and STR filing quality

**Question 2:** "Have you ever had an RBI observation or penalty related to STR filing delays?"
- If yes → heavily emphasise SLA tracking and Audit Logs
- If no → emphasise it as prevention, not cure

**Question 3:** "When your analysts finish an investigation today, how do they write the STR narrative?"
- If they say "manually in Word" → the AI Narrative feature will be the most impressive
- If they say "we have a template" → ask how long it takes — then show the AI doing it in 30 seconds

---

## THE OPENING STORY (5 minutes)

**Do not open the app yet. Tell this story first.**

> "I want to describe a typical Thursday at a compliance team that doesn't use OpsOwl. This is a real scenario based on what we hear from compliance heads every week.
>
> It's 2pm. Your transaction monitoring system generates an alert. Account holder Kiran Patel has made 11 cash deposits over 3 months, all just below ₹10 lakhs — the level that triggers a Currency Transaction Report. The total is ₹58.5 lakhs. All of it transferred out to three companies within 48 hours of deposit.
>
> Your analyst, Divya, gets the alert in her inbox. She opens Excel. She starts copying transaction data from the CBS export. She emails the KYC team asking for Kiran Patel's documents. She waits. The KYC team replies the next morning. She downloads the attachments, reads through 50 pages of bank statements manually. She takes notes in a Word document on her desktop.
>
> She discovers that the three receiving companies were all incorporated in the last 6 months. She searches the internet for each company. No website, no GST registration, no directors visible on MCA. She writes up her findings in the Word document.
>
> She sends it to her manager, Ravi. Ravi is in meetings all day. He reads it Friday morning. He has a question about one of the transactions — he emails Divya. Divya is now working on 4 other cases. She replies Monday.
>
> It's now been 6 days. FIU-IND requires an STR within 7 days of forming suspicion. Ravi approves the Word document and asks Divya to enter everything manually into the goAML portal. Divya spends another hour typing the same information again.
>
> Total time: 6 days, 3 people, no central record of what happened, no audit trail, data entered twice, and the whole thing sitting on a Word document on Divya's desktop that nobody else can access.
>
> Now: what if Divya's colleague investigated the same Kiran Patel account 6 months ago? Nobody knows. That investigation is also in a Word document on a different computer.
>
> That is the problem. Let me show you what happens with OpsOwl."

**Now open the browser.**

---

# ★ PART 1 — THE DASHBOARD
### "Your Morning Briefing in 10 Seconds"

**Click path:** Log in as admin@opsowl.dev → You land on the Dashboard automatically.

---

### What you see on the Dashboard

The Dashboard has three sections. Walk through each one slowly.

---

**Section A: The Four Stat Cards (top of the page)**

Point to each card and explain what it means:

**Card 1 — Total Cases**
> "This is the total number of investigations in your system right now. Every alert that became a formal investigation, every internal referral, every regulatory inquiry — all counted here. If your CBS sends 200 alerts a month and your team converts 60 of them to formal investigations, this number grows by 60 each month."

**Card 2 — Open Cases**
> "These are cases that are actively being worked. Not closed, not sitting in draft. Your compliance team is right now actively investigating these. If this number is very high and your team is small, that tells your manager something is wrong — either cases are piling up or SLA breaches are coming."

**Card 3 — SLA Breached**
> "This is the most important number on the page. These are cases where the investigation deadline has already passed. In plain language: your team was supposed to finish these investigations by a certain date, and they did not. For AML cases, this can mean you missed the 7-day STR filing window required by FIU-IND. Every number here is a potential regulatory violation. If this number is not zero, your MLRO should be making phone calls right now."

**Card 4 — Attention Required**
> "This captures all cases that are either already breached or about to breach — plus any cases that have been escalated as urgent. Think of this as your 'fire list' for the day. These are the cases that need a human being to act on them today, not tomorrow."

---

**Section B: Attention Required Panel**

Point to the list of cases in this panel:

> "Each row here is a case that is in trouble — either the SLA has passed or a manager has escalated it. The panel shows: the case number, what it's about, who owns it, and how far past the deadline it is.
>
> This panel refreshes every 10 seconds automatically. You do not press a button. You do not run a report. If a case breaches SLA at 3pm while your analyst is on a call, this panel shows it by 3:00:10pm.
>
> Why does this matter? Today, how does your MLRO find out that a case has breached SLA? Probably because someone tells them. Or because it's Monday and they check the Excel sheet and notice the date has passed. By then it's too late. OpsOwl tells you the moment it happens."

---

**Section C: My Queue**

Point to the list of 5 cases at the bottom:

> "This section is personalised. It shows the 5 cases assigned specifically to the person who is logged in right now, sorted by how urgent their SLA is.
>
> When an analyst starts their day, this is the first thing they see. They don't need to scroll through 200 cases to find their work. They don't need to ask their manager what to prioritise. The system tells them: these are your 5 most urgent cases, work on them in this order.
>
> This alone saves your analysts 20–30 minutes every morning."

---

**Why the Dashboard matters — the business case:**

> "Before OpsOwl, getting this information required: a daily standup meeting, a manually updated Excel dashboard, or a manager walking the floor asking everyone for updates. That's 30–60 minutes of management time every day just to know what's happening.
>
> The OpsOwl dashboard gives you all of this information live, in 10 seconds, without a single person having to update anything. The data comes directly from case activity. If an analyst completes a task, the count updates. If a case breaches SLA, it appears in the Attention panel. No human reporting required."

**Common objection:** *"Our CBS already shows us alerts and case counts."*

> "Your CBS shows you transaction data — amounts, accounts, timestamps. This is different. This is the investigation status of those transactions. Who is investigating? How long have they been working on it? Are they going to make the regulatory deadline? Will they be done by Friday? These are compliance workflow questions, not transaction questions. Your CBS was not built to answer them."

---

# ★ PART 2 — THE CASES LIST
### "Your Team's Complete Investigation Register"

**Click path:** Click **Cases** in the left sidebar.

---

### What is a Case?

Before showing the list, explain what a case is:

> "In OpsOwl, a 'Case' is a formal investigation record. Every suspicious activity your team investigates becomes a case. Every alert from your transaction monitoring system that is serious enough to look into — becomes a case. Every internal referral from your relationship manager saying 'I think something is wrong with this customer' — becomes a case.
>
> A case is the container for everything related to one investigation: all the documents, all the notes your analyst writes, all the tasks they complete, every phone call they log, every finding they document, and ultimately the STR report they file with FIU-IND.
>
> Nothing about an investigation should live in someone's email inbox or on their desktop. It lives in the case."

---

### The Cases List — Column by Column

Point to each column in the table and explain it:

**Case Number (e.g., INV-0052)**
> "Every case gets a unique, permanent number the moment it is created. This number is used in all communications, in your STR filing, in your audit logs, and in regulatory correspondence. It never changes and is never reused. If an auditor calls and mentions 'case INV-0052', your team can find it in 2 seconds."

**Title**
> "A human-readable name for the investigation. Your analyst names it when creating the case. You'll see titles like 'Suspicious Cash Deposits — Kiran Patel' or 'Round-Tripping Pattern — ABC Enterprises'. The title helps your team quickly identify what a case is about without opening it."

**Status**
> "Where the case is in the investigation workflow right now. The statuses are:
> - **Draft** — case created but investigation not started
> - **Open** — investigation has started
> - **In Progress** — analyst is actively working it
> - **Pending Review** — analyst is done, waiting for manager to review
> - **Escalated** — something urgent needs management attention immediately
> - **Closed** — investigation complete, decision made
>
> The entire lifecycle of every investigation is visible here. Your manager can tell at a glance: 30 cases in Pending Review means 30 cases waiting for me to act on."

**Priority**
> "How urgent this investigation is. Critical, High, Medium, or Low. Your SLA countdown — the investigation deadline — is tied to priority. A Critical AML case might have a 24-hour deadline. A Medium Fraud case might have 5 days. Priority is set when the case is created and can be changed if the risk profile changes."

**Assignee**
> "The name of the analyst who owns this investigation. In a team of 10 analysts, your manager can see immediately who is working on what. If an analyst is sick, the manager sees 'Divya Reddy' on 18 cases and knows those need to be reassigned."

**SLA Countdown**
> "This is the remaining time before the investigation deadline is breached. It counts down in real time — you can watch it tick. Green means plenty of time. Amber means less than 24 hours. Red means it has already breached.
>
> This is the most important column in this table. Every red cell in the SLA column is a case where you have already missed your internal deadline. Your MLRO can see this from 10 feet away without reading any details."

**Risk Score**
> "The AI-calculated risk score for this case, expressed as a level: LOW, MEDIUM, HIGH, or CRITICAL. We'll go into detail on how this score is calculated when we show the AI features. For now, know that it lets your manager filter to 'show me only HIGH and CRITICAL risk cases' — the investigations most likely to need STR filing."

---

### Demo: Filters and Search

**Show the Status filter:**

Click the **Status** dropdown on the right side of the table. Select **In Progress**.

> "Instantly, the list filters to only cases that are actively being worked. If I'm a manager doing my morning review, I want to see what my team is currently handling. One click.
>
> You can select multiple statuses at once. I can select 'Open' and 'In Progress' together to see all active investigations. Or I can select only 'Pending Review' to see the cases waiting for my approval."

**Show the My Cases toggle:**

Click the **My Cases** toggle button (person icon near the filters):

> "Switch on 'My Cases' and the list instantly shows only the cases assigned to the logged-in user. An analyst who has 200 cases in the system doesn't want to see all 200 — they want to see their 18. One click."

**Show the Search bar:**

Type **"Patel"** in the search box:

> "As you type, the list filters to cases matching what you typed. This is real-time server-side search — it doesn't just search what's loaded on screen, it searches your entire database.
>
> Why does this matter? If your team has 2,000 cases and you're looking for all investigations related to the Patel family, you can find them in 3 seconds. No scrolling, no sorting, no export to Excel."

---

### Demo: Bulk Operations

Select 3 cases using the checkboxes on the left side of each row. Watch the action bar appear at the top of the table.

> "When you select cases, an action bar appears. As an Admin, I have three options: Change Status, Bulk Reassign, or Delete."

**Demonstrate Bulk Reassign:**

Click **Bulk Reassign** → select a different analyst → confirm.

> "I just reassigned 3 cases to a different analyst in 8 seconds. In the real world: your senior analyst, Divya, has just gone on maternity leave. She has 15 active cases. Her manager needs to redistribute those cases to two other analysts. With OpsOwl — select all 15, bulk reassign 8 to Arjun and 7 to Priya. Done in 2 minutes.
>
> Without OpsOwl: the manager opens Excel, identifies which cases belong to Divya, emails each analyst individually, waits for confirmation, updates the Excel sheet. Half a day's work."

**Demonstrate Bulk Change Status:**

Select cases again → click **Change Status** → select a status → confirm.

> "Similarly, if the compliance head decides that all low-risk cases created before a certain date should be closed, they can select them all and close in one action. What would have taken an analyst a full day of opening cases one by one takes 30 seconds."

---

# ★ PART 3 — INSIDE A CASE
### "The Investigation Workspace — Where the Real Work Happens"

**Click path:** Click any case in the list to open it.

---

### The Case Header — Always Visible

When the case opens, point to the **sticky header** at the top. Explain that this header stays visible no matter how far down the analyst scrolls.

> "The case header shows the most important information about this case at all times. Your analyst never loses context while working deep in the documents tab or the notes tab. Let me walk through each element."

**Case Number and Title**
> "Always visible. If the analyst is on a call and someone asks 'what case are you working on?', the answer is right at the top."

**Status Badge with Transition Buttons**
> "The current status is shown as a coloured badge. Next to it are buttons to move the case forward. Some transitions are instant — one click, no confirmation needed. For example, moving from 'Open' to 'In Progress' is routine — click it, done, timeline updated.
>
> But some transitions are serious — like 'Escalate' or 'Close'. Those are red buttons and they open a dialog asking for a reason. The reason is mandatory and is logged permanently. Why? Because if your auditor asks 'why was this case escalated?', you need an answer. Not 'I think Divya escalated it for some reason' — an actual documented reason with a timestamp."

**Demo:** Click the **In Progress** button. Show the status badge updating. Point to the Timeline tab and show the timeline entry that was automatically created:

> "Every status change creates an automatic timeline entry. The analyst didn't have to write 'moved to In Progress' — the system recorded it automatically. This is how OpsOwl builds the complete case history without asking your analysts to manually document every action."

**Priority Badge**
> "The priority — Critical, High, Medium, Low — displayed prominently. If this needs to change as the investigation unfolds, the analyst or manager can change it here. When priority changes, the SLA countdown recalculates automatically."

**SLA Countdown**
> "A live timer showing how much time is left before the investigation deadline is breached. It counts down in real time. Green = safe. Amber = less than 24 hours, act today. Red = already breached.
>
> Every person who opens this case — the analyst, the manager, the MLRO — sees this timer. Nobody can claim they didn't know the deadline was approaching."

**AI Risk Score Badge**
> "The AI's assessment of how risky this case is, on a scale of 0–100. HIGH 74 means this case scores 74 out of 100 on risk indicators. We'll explain exactly how this score is calculated when we get to the AI features. The point here is that it's visible in the header — the manager doesn't need to open the AI Insights tab to know whether this is a high-risk case."

**Action Buttons on the right of the header:**
- **🤖 AI Copilot** — opens the AI assistant panel for this case
- **⚡ Analyse** — runs the AI risk analysis on all uploaded documents
- **📄 PDF** — generates a complete case summary PDF in one click

> "These three buttons represent the three things a manager most commonly needs to do on a case: get AI help, get a fresh risk score, or export a summary for an auditor. All three are one click from anywhere in the case."

---

### The Two-Column Workspace

Below the header, the case splits into two columns. Explain the layout:

**Left column (65% of the screen) — The Investigation Feed:**
> "The left side is where the investigation story unfolds. At the top is the case description — what triggered this investigation, what the initial facts are. Below that is the live timeline feed — a real-time stream of everything that has happened on this case."

**Right column (35% of the screen) — The Context Panel:**
> "The right side gives you the quick summary without switching tabs. Who owns this case. When it was created. When it's due. The SLA deadline. The 3 most urgent open tasks. The last 2 uploaded documents. The tags and watchers.
>
> A manager reviewing cases can get the essential context from the right column without clicking into any tabs. They only open the tabs when they need to go deeper."

---

### The Live Timeline Feed

Point to the timeline feed in the left column:

> "Watch this section while I do something in another tab."

Open a second browser tab, log in as a different user, navigate to the same case, and add a note.

Come back to the first tab:

> "The note appeared here without refreshing the page. This is not polling — your browser didn't check the server every few seconds. This is a live, persistent connection. The moment any action happens on this case — by any analyst, anywhere in your organisation — it appears in this feed instantly.
>
> Why does this matter? If you have two analysts co-investigating the same case — one in Mumbai, one in Delhi — they can work simultaneously and see each other's actions in real time. Like Google Docs, but for investigations."

---

### Inline Reassignment

In the right column, click the Assignee name:

> "A dropdown appears. I can reassign this case to any analyst in the organisation right here, without opening a form, without navigating away, without emailing anyone.
>
> The new assignee gets a notification immediately. The timeline records who reassigned it and when. One click, full audit trail."

---

# ★ PART 4 — THE INVESTIGATION WORKFLOW
### "Walking Through a Complete AML Investigation"

> "Now I'm going to walk through the complete investigation of the Kiran Patel case I described in the opening story. This will show every step your analyst would take — from case creation to STR filing."

---

## Step 1: Setting Up SLA Policies (Admin does this once)

**Click path:** Administration → **SLA Policies** → **New Policy**

> "Before we create the case, I want to show you something your admin sets up once and it runs forever. SLA Policies.
>
> An SLA policy says: for cases of this type and this priority, the investigation must be completed within this many hours. Your organisation defines what 'on time' means here."

Create these three policies (show each one being created):

**Policy 1:**
- Case Type: AML
- Priority: Critical
- Breach Time: 24 hours

> "Any AML Critical case must be closed within 24 hours. Why 24 hours? Because FIU-IND requires STR filing within 7 days of suspicion. Your internal target should be much tighter — if every analyst knows they have 24 hours, you have a 6-day buffer before you breach the regulatory deadline."

**Policy 2:**
- Case Type: AML
- Priority: High
- Breach Time: 48 hours

**Policy 3:**
- Case Type: Fraud
- Priority: High
- Breach Time: 36 hours

> "Once these policies are set, every case created in the system gets a countdown automatically. Your team doesn't manage SLA in a spreadsheet anymore. The system manages it for them."

**Objection:** *"Our team already knows what the deadlines are."*

> "Do your analysts know the deadline for every case at every point in time? When they have 18 active cases each? When someone goes on leave and cases get reassigned? When a case priority changes because new information emerged? The deadlines in a spreadsheet are static. The deadlines in OpsOwl are live and change dynamically when case priority changes or when cases are reassigned."

---

## Step 2: Setting Up Case Templates (Admin does this once)

**Click path:** Administration → **Templates** → **New Template**

> "A case template pre-fills the investigation form with everything your team needs to start — the description, the priority, and most importantly, the task checklist.
>
> Think of it as your investigation SOP — Standard Operating Procedure — encoded into the software."

Fill in the template:

**Template Name:** AML — Suspicious Cash Transaction

**Default Priority:** High

**Description:**
```
Suspicious cash transaction pattern detected. Customer has made
repeated large cash deposits (above ₹10L threshold) followed by
immediate outward NEFT transfers to recently incorporated entities.
Investigation required under PMLA Section 3 — proceeds of crime.
Analyst should focus on: (1) transaction structuring indicators,
(2) beneficiary entity verification, (3) source of funds documentation.
```

**Pre-created Tasks (add these one by one):**
1. Verify customer KYC — check PAN, Aadhaar, address proof are current
2. Pull 12-month transaction history from CBS
3. Identify all beneficiary accounts and entities receiving transfers
4. Check beneficiary company registration dates on MCA portal
5. Run customer through adverse media screening (Google, RBI defaulter list)
6. Check CIBIL score and existing credit facilities
7. Call customer for explanation if no documentary evidence available
8. Prepare findings summary and risk assessment
9. Draft STR narrative for manager review

Click **Save**.

> "Now every analyst who creates an AML Suspicious Cash Transaction case — from your most senior investigator to someone who joined last week — starts with the same 9-step checklist. Your institutional knowledge is encoded in the system.
>
> When that senior analyst retires, their process doesn't retire with them. It's in the template."

**Objection:** *"Our analysts are experienced. They don't need a checklist."*

> "This is not about whether your analysts are competent. It is about defensibility. When your RBI auditor asks 'how do you ensure consistent investigation quality?', you want to say 'we have a standardised process enforced by our system, with timestamps proving every step was completed' — not 'our senior analysts know what to do'. One is auditable. The other is a prayer."

---

## Step 3: Setting Up Auto-Assignment Rules (Admin does this once)

**Click path:** Administration → **Assignment Rules** → **New Rule**

> "Auto-assignment rules decide who gets a case the moment it's created. You define the rules once, and the system routes cases automatically forever."

Create this rule:
- **Rule Name:** AML Critical → Senior Analyst
- **Condition:** Case Type = AML AND Priority = Critical
- **Assign to:** Divya Reddy (your most experienced AML analyst)
- **Save**

Create a second rule:
- **Rule Name:** Fraud High → Fraud Specialist
- **Condition:** Case Type = Fraud AND Priority = High
- **Assign to:** Arjun Mehta
- **Save**

> "When any AML Critical case is created — whether by an analyst manually, whether it was converted from an alert, whether it came from your CBS via API at 3am — it is assigned to Divya automatically. The correct person gets the case before they've even opened their laptop.
>
> Without this, a case sits unassigned until a manager notices. That might be the next morning. For a 24-hour SLA case, that is already half the time gone."

---

## Step 4: Setting Up Investigation Playbooks (Admin does this once)

**Click path:** Administration → **Playbooks** → **New Playbook**

> "Playbooks are similar to templates, but they work differently. A template is applied when you choose it during case creation — it only runs if your analyst remembers to select it. A playbook runs automatically based on the case type, whether or not the analyst thinks about it.
>
> Think of templates as optional checklists. Playbooks as mandatory checklists."

Create a playbook:
- **Playbook Name:** Standard AML Investigation Process
- **Trigger:** Case Type = AML
- **Tasks:**
  1. Initial triage — confirm alert is valid
  2. Document collection — request and upload all relevant documents
  3. Entity verification — verify all parties and companies
  4. Risk assessment — complete the risk scoring worksheet
  5. Manager review — present findings to manager

> "The moment any AML case is created in this system — by anyone, through any channel — these 5 tasks are created and assigned. Your SOP is enforced at the system level. It is not possible for an analyst to skip it because they forgot."

---

## Step 5: ★ Creating the Kiran Patel Case

**Click path:** Cases → **New Case** button (top right)

Fill in the form step by step:

**Title:**
```
Suspicious Cash Deposits — Kiran Patel (A/C 9876543210)
```

> "Give the case a title that tells anyone who sees it what it's about without opening it. Include the customer name and account number so your team can search for it later."

**Case Type:** AML

**Priority:** High

**Description:** Click the Template dropdown → select **AML — Suspicious Cash Transaction**

Watch the description and tasks auto-populate:

> "Watch — the entire description field filled in automatically, and 9 tasks appeared in the tasks preview. I didn't type a single word of the description. The template did it. The analyst's job now is to investigate, not to fill in forms."

**Due Date:** Set to 2 days from today

**Assigned To:** Divya Reddy (note: auto-assignment rule would have done this too)

Click **Create Case**.

> "Case INV-0052 is created. The SLA countdown has started — 48 hours, counting down. Divya has been notified. The 9 tasks are created and assigned to her. The playbook tasks have also been created. Everything is ready."

Show the newly created case — point out:
- The SLA timer counting down
- The tasks in the right panel
- The auto-generated timeline entry "Case created by admin@opsowl.dev"

---

## Step 6: ★ Uploading Documents

**Click path:** Case → **Documents** tab

> "The first thing Divya does is gather all her evidence. She uploads the documents directly into this case. No email attachments, no shared drives, no document management system to log into separately. Everything stays in the case."

Click **Upload** (or drag files into the upload zone):

Upload 3 files (use any PDF you have, or create simple test files):

1. **kiran_patel_account_statement_jan_mar_2024.pdf** — the bank statement
2. **kiran_patel_kyc_documents.pdf** — Aadhaar, PAN card
3. **beneficiary_company_registrations.pdf** — company incorporation certificates

Watch the status badges change on each document:
- **Queued** — document received, waiting for AI processing
- **Processing** — AI is reading and analysing the document right now
- **Done** — AI has finished, document is fully searchable

> "While those documents are processing, let me explain what the AI is doing in the background.
>
> It is reading every page of every document. For scanned images — like a photographed copy of the Aadhaar card — it is running OCR to extract the text first. Then it is breaking the document into sections, understanding what each section means, and creating a searchable index.
>
> By the time the status says 'Done', Divya can ask the AI 'what is the total amount credited to this account in January 2024?' and get an exact answer — without reading a single page herself."

**Why this matters:**
> "A 50-page bank statement takes your analyst 2–3 hours to read carefully. The AI reads it in 20 seconds and extracts every number, name, account, date, and amount into a searchable format. Divya's 3-hour task just became a 20-second task."

---

## Step 7: Logging Communications

**Click path:** Case → **Comms** tab → **Log Communication**

> "Before Divya draws any conclusions, she calls Kiran Patel to ask about the cash deposits. This conversation must be documented. Not in her personal notes app, not in an email to herself — in the case."

Fill in the communication form:

- **Channel:** Phone
- **Direction:** Outbound (we called the customer)
- **Contact Name:** Kiran Patel
- **Contact Number:** +91 98765 43210
- **Subject:** Inquiry regarding cash deposit activity Jan–Mar 2024
- **Notes:**
  ```
  Called customer at 2:30pm. Customer stated cash deposits are from
  agricultural income from family farm in Nashik and gifts from relatives
  for his daughter's wedding. When asked for documentary evidence (farm
  records, gift letters), customer stated he did not have any documents
  and would need to arrange them. When asked about the three receiving
  companies (Sunrise Trading, GreenLeaf Commodities, BlueSky Imports),
  customer said they are 'business partners' but could not provide
  the nature of business or any contracts. Call duration: 12 minutes.
  Customer was cooperative but evasive on specifics.
  ```
- **Duration:** 12 minutes

Click **Save**.

> "This communication is now permanently on record. Timestamp, channel, direction, what was said. Go to the Timeline tab right now and you'll see it has already appeared there automatically.
>
> Why is this critical? If this case goes to a court hearing or a regulatory show-cause, your lawyer needs to show that you performed due diligence — that you contacted the customer, what they said, and when. This log is that evidence."

**What makes this different from email:**
> "If Divya had sent this call summary to her manager by email, it would be in Divya's Sent folder. When Divya leaves the company, that email is gone. The case in OpsOwl stays forever. The communication log stays forever. The audit trail is complete."

---

## Step 8: Writing Investigation Notes

**Click path:** Case → **Notes** tab → **Add Note**

> "As Divya investigates, she writes her analysis notes here — not in Word, not in a notebook, not in an email. In the case."

Click **Add Note**. Select type: **Internal** (only team members can see this).

Type this note:

```
TRANSACTION PATTERN ANALYSIS — 13 Jun 2024

Based on review of 3-month bank statement (Jan–Mar 2024):

DEPOSITS:
- 11 cash deposits made between 5 Jan and 28 Mar 2024
- Total deposited: ₹58,50,000
- Deposit amounts range: ₹4,80,000 to ₹6,20,000
- Every deposit is below the ₹10,00,000 CTR threshold
- Average gap between deposits: 8 days
- Pattern is consistent with deliberate structuring to avoid CTR reporting

OUTWARD TRANSFERS:
- All deposited funds transferred within 48 hours via NEFT
- Three beneficiary accounts identified:
  1. Sunrise Trading Pvt Ltd — A/C 8765432109, IFSC HDFC0001234
  2. GreenLeaf Commodities LLP — A/C 7654321098, IFSC ICIC0005678
  3. BlueSky Imports — A/C 6543210987, IFSC AXIS0009012

BENEFICIARY VERIFICATION:
- Sunrise Trading: incorporated Dec 2023 (3 months ago), no GST registration,
  no website, director is Ramesh Patel (same surname as customer)
- GreenLeaf Commodities: incorporated Nov 2023 (4 months ago), no visible
  online presence, no phone number listed on MCA portal
- BlueSky Imports: incorporated Jan 2024 (2 months ago), registered address
  is a residential apartment in Andheri, Mumbai

CUSTOMER EXPLANATION:
- Claims agricultural income and family gifts as source
- Could not provide any documentary evidence when asked
- Could not explain the nature of business with receiving entities

ASSESSMENT:
Classic structuring + layering pattern. Deposits structured below CTR
threshold, immediately transferred to three shell companies with
connections to the customer's family. High probability of PMLA violation.

RECOMMENDATION: File STR. Risk level: HIGH.
```

Click **Save**.

> "This note is now permanently attached to this case. Every team member who opens this case can read Divya's complete analysis. Her manager Ravi can see exactly what she found without asking her to summarise it in an email. If Divya goes on leave and Arjun takes over, he reads this note and is fully up to speed in 5 minutes.
>
> Most importantly: this note has a timestamp. It records when Divya reached the conclusion that this case needed an STR. That timestamp is part of the regulatory evidence — it helps prove that you acted within the required 7-day window."

**Note Types — explain briefly:**

> "Notes have three types:
> - **Internal**: Only your investigation team can see it. Used for analysis notes, hypotheses, discussions.
> - **Customer-facing**: Records communications intended for the customer — formal letters, notices.
> - **Escalation**: Specifically marks a note as an escalation to management. Creates a flag so managers know this note needs attention."

---

## Step 9: ★ Completing Tasks

**Click path:** Case → **Tasks** tab

> "The tasks pre-created from the template are here. Divya is now going through them one by one as she completes each step."

Show the task list. Click to complete the first three tasks:

Click the checkbox on **"Verify customer KYC"**:

> "Task marked complete. Timestamp recorded. Divya's name recorded. A 5-second toast appears at the bottom right — this is an 'undo' button if she accidentally clicked the wrong task. After 5 seconds, the completion is permanent."

Click the checkbox on **"Pull 12-month transaction history"** and **"Identify all beneficiary accounts"**:

> "Each completion is logged. A manager reviewing this case can see exactly which steps have been done, by whom, and when. They don't need to ask Divya 'have you checked the beneficiary companies?' — the task list tells them."

Show an **unassigned task** — point to the "Assign to me" button:

> "This task was created but not yet assigned to anyone. Any analyst who picks up this case can click 'Assign to me' to take ownership of it. This is useful when a manager wants to distribute tasks across multiple analysts working on the same case."

Click the **Priority** dropdown on a task to change it:

> "Each task has its own priority and due date. If one task is urgently needed — say, the manager needs the beneficiary verification done by end of day — that task can be set to Critical priority so it appears at the top."

---

## Step 10: ★ Documenting Findings

**Click path:** Case → **Findings** tab → **New Finding**

> "This is one of the most important features in OpsOwl. Findings are different from notes.
>
> A note is informal — 'I think this looks suspicious'. A Finding is a formal, evidence-linked conclusion. 'I have found that this customer structured deposits to avoid CTR reporting, and here is the bank statement proving it.'
>
> Findings are what feed into the STR filing. Every claim in your STR narrative must be backed by a finding. Every finding must be backed by evidence. This creates an unbreakable chain: STR claim → Finding → Document evidence."

**Create Finding 1:**

Click **New Finding**.

- **Title:** Deliberate structuring of cash deposits to avoid CTR reporting
- **Severity:** High
- **Status:** Confirmed
- **Description:**
  ```
  Customer made 11 cash deposits between January and March 2024, totalling
  ₹58,50,000. Every individual deposit was between ₹4,80,000 and ₹6,20,000 —
  consistently below the ₹10,00,000 Currency Transaction Report threshold.
  The pattern of 11 deposits in 83 days, all below threshold, is statistically
  inconsistent with random transaction behaviour and is consistent with
  deliberate structuring under PMLA Section 3.
  ```
- **Link Evidence:** Click **Attach Document** → select the bank statement PDF

Click **Confirm Finding**.

> "The finding is now Confirmed. It has a severity of High. It is linked to the specific document that proves it. The analyst who confirmed it and the timestamp are recorded."

**Create Finding 2:**

- **Title:** 100% of deposited funds transferred to newly incorporated shell companies within 48 hours
- **Severity:** Critical
- **Status:** Confirmed
- **Description:**
  ```
  All ₹58,50,000 deposited was transferred via NEFT to three entities
  within 48 hours of each deposit. All three entities were incorporated
  in the 90 days preceding the investigation period (Nov 2023–Jan 2024).
  None of the entities have a GST registration, a verifiable business
  activity, or a public-facing business presence. One director (Sunrise
  Trading) shares the customer's surname. This transfer pattern is
  consistent with the layering stage of money laundering — moving
  funds rapidly through intermediary entities to obscure the origin.
  ```
- **Severity:** Critical
- **Link Evidence:** Attach the company registration document

Click **Confirm Finding**.

> "Two confirmed findings, one High and one Critical. These two findings will be the backbone of the STR narrative. The AI narrative generator will read these findings, understand what they mean, and write the STR text using these exact facts.
>
> Without findings, your STR narrative is a Word document with claims but no evidence chain. With findings, every claim in the STR has a traceable source."

---

## Step 11: Recording Transactions

**Click path:** Case → **Transactions** tab → **Add Transaction**

> "The Transactions tab creates a structured flow-of-funds record. This is separate from the document — instead of a PDF that the AI read, this is structured data that both the AI and your team can query directly.
>
> The FIU-IND goAML filing requires specific transaction data in a specific format. Logging transactions here means you enter the data once, and OpsOwl formats it correctly for the goAML XML export."

Add three transactions:

**Transaction 1:**
- **Date:** 15 January 2024
- **Type:** Cash Deposit (Credit)
- **Amount:** ₹52,00,000
- **From Account:** Cash (counter deposit)
- **To Account:** 9876543210 (Kiran Patel, FinEdge NBFC)
- **Flagged:** Yes — check the flag checkbox
- **Notes:** Largest single deposit. Counter deposit by customer in person.

**Transaction 2:**
- **Date:** 16 January 2024 (24 hours after deposit)
- **Type:** NEFT (Debit)
- **Amount:** ₹19,50,000
- **From Account:** 9876543210 (Kiran Patel)
- **To Account:** 8765432109 (Sunrise Trading Pvt Ltd, HDFC Bank)
- **Flagged:** Yes
- **Notes:** Transferred 24 hours after cash deposit. Sunrise Trading incorporated Dec 2023 — 1 month old at time of transfer.

**Transaction 3:**
- **Date:** 16 January 2024
- **Type:** NEFT (Debit)
- **Amount:** ₹18,00,000
- **From Account:** 9876543210 (Kiran Patel)
- **To Account:** 7654321098 (GreenLeaf Commodities LLP, ICICI Bank)
- **Flagged:** Yes
- **Notes:** Same-day transfer alongside Sunrise. Total outflow = ₹37.5L within 24 hours of deposit.

> "Three transactions logged. ₹52 lakhs came in, ₹37.5 lakhs went out within 24 hours, split between two shell companies. This flow of funds is now structured data in OpsOwl. The AI narrative generator will use these amounts, dates, and account numbers when writing the STR. The goAML XML export will pull these records directly.
>
> Without this tab, your analyst would enter the same transaction data a second time when filling in the goAML form. That double data entry is eliminated."

---

## Step 12: ★ Linking the Subject (Subject 360)

**Click path:** Sidebar → **Subjects** → **New Subject**

> "Before we run the AI analysis, I want to show you one of the most powerful features in OpsOwl — Subject 360.
>
> A Subject is a person or company that appears in your investigations. The key insight is: the same person appears in multiple cases over time. Kiran Patel might be involved in 3 investigations over 3 years. Without Subject 360, each investigation treats Kiran Patel as a new person. With Subject 360, you see his complete history."

Fill in the subject profile:

- **Type:** Individual
- **Full Name:** Kiran Patel
- **PAN Number:** ABCDE1234F
- **Date of Birth:** 15-Mar-1981
- **Address:** 14, MG Road, Pune 411001
- **Phone:** +91 98765 43210
- **Linked Account:** 9876543210 (FinEdge NBFC)
- **Risk Level:** High
- **Notes:** Suspected structuring + layering. Connected to 3 shell companies through family links.

Click **Save**.

Now go back to the Kiran Patel case → right column → **Subjects** section → **Link Subject** → search "Kiran" → select Kiran Patel → **Link**.

Now click into the Subjects page and open Kiran Patel's profile:

> "This is the Subject 360 view. Right now it shows one case — INV-0052. But imagine it is 2 years from now. Kiran Patel has been involved in 4 investigations. You see: 4 cases listed, ₹2.4 crores total suspicious transactions across all cases, 3 STRs filed, risk score trend showing increasing risk. All of this in one view, without searching through individual cases.
>
> Why does this matter? Money laundering is a long-term behaviour. The same people and accounts appear repeatedly. Without cross-case linking, each investigation looks like an isolated incident. With Subject 360, you see the pattern. And patterns are what FIU-IND and law enforcement actually need."

**Objection:** *"We already have customer profiles in our CBS."*

> "Your CBS profile shows transaction data for that account. The Subject 360 profile shows investigation history — how many times has this person been investigated, what were the outcomes, were STRs filed, are they connected to other investigated entities. These are compliance intelligence questions, not banking questions."

---

# ★ PART 5 — THE AI FEATURES
### "Your Compliance Team's AI Analyst"

> "Now I want to show you the AI layer. This is what makes OpsOwl fundamentally different from any case management system you've seen before.
>
> Everything up to this point — creating cases, uploading documents, writing notes, completing tasks — any case management system can do that. What I'm about to show you, almost none can."

---

## AI Feature 1: ★ Risk Analysis (⚡ Analyse)

**Click path:** Case header → click the **⚡ Analyse** button

Wait 15–30 seconds. Watch the risk score badge in the header update.

> "The AI just read all three documents uploaded to this case. Let me show you what it found."

**Click path:** Case → **AI Insights** tab

Point to the extracted entities:

> "Look at what the AI extracted from your documents automatically:
>
> - Account numbers: every account number mentioned in the bank statement
> - PAN numbers: Kiran Patel's PAN from the KYC document
> - Company names: Sunrise Trading, GreenLeaf Commodities, BlueSky Imports
> - Transaction amounts: every credit and debit with dates
> - Director names: pulled from the company registration certificates
>
> Your analyst would have taken 3 hours to extract this manually. The AI did it in 20 seconds."

Point to the Risk Score (e.g., HIGH 74):

> "The AI calculated a risk score of 74 out of 100 — HIGH. The score is based on: number of confirmed findings, severity of those findings, transaction amounts, structuring indicators found in the documents, and comparison to historical cases in the system.
>
> This score is not a black box. Click on it and you'll see the breakdown — what contributed to the score and how much."

Point to the **Risk Score History** section if there is more than one analysis:

> "Every time analysis runs, the score is recorded. If you add new documents and run analysis again, the score might change. Your MLRO can see whether a case is getting riskier or lower risk as the investigation progresses."

---

## AI Feature 2: ★ AI Copilot — Ask

**Click path:** Case header → **🤖 AI Copilot** → **Ask tab**

> "The Ask tab is like having a junior analyst who has read all your documents and is ready to answer any question about them. The AI uses only your uploaded documents to answer — it does not make things up."

Type in the question box:

**Question 1:** `What are the total cash deposits made by Kiran Patel between January and March 2024?`

Wait for the answer. Show the response.

> "The AI answered with the exact number from the bank statement. It read through 50 pages and found the answer in 3 seconds."

**Question 2:** `What are the names and registration dates of the companies that received NEFT transfers from this account?`

> "The AI pulled this from the company registration documents. It found all three company names and their dates of incorporation. This would have taken your analyst 20 minutes of searching — the AI did it in 4 seconds."

**Question 3:** `Is there anything in these documents that suggests the customer has a legitimate business reason for these transactions?`

> "A genuinely useful compliance question. The AI reads the documents, looks for any evidence of legitimate business purpose — invoices, contracts, business correspondence — and reports what it finds. If the answer is 'no evidence found', that itself is important for the investigation."

**Why this is different from a search engine:**

> "A search engine finds documents that mention a keyword. This AI understands the content and answers questions about it. You're asking it to reason, not just retrieve. That's the difference between Ctrl+F and an analyst."

**Objection:** *"What if the AI gives a wrong answer?"*

> "Every AI response comes with a source reference — which document, which section it based the answer on. Your analyst can click through and verify. More importantly: the AI is answering questions about documents you uploaded. If the document says ₹58.5 lakhs, the AI says ₹58.5 lakhs. It is not guessing.
>
> Think of it as a very fast, very tireless analyst that reads documents accurately. It can still misinterpret an ambiguous table or miss context. That's why your analyst reviews the answer — they don't treat it as final."

---

## AI Feature 3: Summarize Tab

**Click path:** Copilot → **Summarize** tab → click **Generate Summary**

Wait for the summary to appear.

> "One click, and the AI produces a structured summary of everything in all the documents — key entities, key transactions, key dates, key findings. This is the 'executive briefing' version of the documents.
>
> When a manager takes over a case from an analyst who is on leave, they can read this 200-word summary and understand the case in 2 minutes instead of reading 50 pages of statements."

---

## AI Feature 4: Draft Note Tab

**Click path:** Copilot → **Draft Note** tab → click **Draft Note**

Wait for the AI-drafted note.

> "The AI reads the case context — the description, the documents, the existing notes — and drafts an investigation note. Your analyst reviews it, edits it, and approves it.
>
> The average analyst spends 45–60 minutes writing investigation notes. With AI drafting, they spend 10 minutes editing. That's a 75% time saving on note-writing alone."

Show the approve/reject buttons:

> "Nothing the AI writes is saved automatically. Ever. Your analyst must review and explicitly approve every AI output before it affects the case. The AI is an assistant, not an autonomous agent. Your analyst remains fully accountable for every word in the case record."

---

## AI Feature 5: Extract Tasks Tab

**Click path:** Copilot → **Extract Tasks** tab → click **Extract Actions**

> "The AI reads the documents and suggests additional investigation tasks based on what it finds.
>
> For example: if the bank statement mentions a fourth company that the analyst hasn't noticed yet, the AI might suggest 'Investigate transfers to XYZ Enterprises — ₹8L transferred on 12 Feb 2024, entity not yet investigated'.
>
> It's like having a second analyst check your work."

---

## AI Feature 6: ★★ The AI Investigation Agent

**Click path:** Copilot → **⚡ Agent** tab

> "This is the most powerful feature in OpsOwl. Everything I've shown you so far answers one question at a time. The Agent does something different — it runs a complete, autonomous, multi-step investigation and produces a full report.
>
> Let me show you what I mean."

In the textarea, type this goal:

```
Analyse this case in full. Read all available information — the case
notes, the uploaded documents, the transaction records, the findings,
and the timeline. Then produce a complete investigation report that
includes: (1) a summary of what happened, (2) the key risk indicators
found, (3) the flow of funds mapped clearly, (4) an assessment of
whether an STR should be filed, and (5) any gaps in the investigation
that still need to be addressed.
```

Click **⚡ Run Agent**.

> "Watch this feed as the Agent works."

Narrate as events appear:

- When `→ Case Context  fetching…` appears:
  > "The agent is reading the basic case information."

- When `✓ Case Context: AML case INV-0052…` appears:
  > "Got the context. Now moving to notes."

- When `→ Notes  fetching…` appears:
  > "Reading the analyst's investigation notes."

- When `→ Transactions  fetching…` appears:
  > "Reading the transaction records we logged."

- When `→ Findings  fetching…` appears:
  > "Reading the confirmed findings."

- When `→ Similar Cases  fetching…` appears:
  > "This is interesting — the agent is searching your historical case database for similar patterns. Has this type of structuring appeared before? Has Kiran Patel appeared in a previous case? It's cross-referencing your entire case history."

When the Investigation Report appears, read it aloud or ask the CTO to read it:

> "The agent synthesised information from 6 different sources — case notes, documents, transactions, findings, timeline, and similar cases — and produced a complete investigation report. It identified all the risk indicators. It mapped the flow of funds. It recommended an STR filing. It flagged one gap — the beneficiary director relationship hasn't been formally documented.
>
> A senior analyst producing this report manually would spend 3–4 hours. The Agent produced it in under 60 seconds.
>
> And remember: this is a first draft. Your analyst reads it, corrects anything that is wrong or incomplete, and uses it as the basis for the formal narrative. The 3–4 hour task just became a 15-minute review."

Show the **Cancel** button while it runs (or explain it):

> "If the agent is heading in the wrong direction — if you see it reading irrelevant data — you can cancel mid-run and restart with a more specific goal."

Show the **Run again** button after completion:

> "After the report is generated, this button resets everything. The analyst can run the agent again with a different, more targeted goal — for example: 'Focus only on the beneficiary company connections and suggest additional investigation steps.'"

**Key safety point to emphasise:**

> "The Agent is 100% read-only. It cannot create a note, change a status, upload a document, or modify anything. It can only read and report. Your data is completely safe. The Agent is an observer, not an actor."

---

## AI Feature 7: ★ Generate AI Narrative

**Click path:** Case → **Narrative** tab → click **Generate Narrative** (top right)

> "This is where the AI saves your team the most time. Writing the STR narrative is the most time-consuming part of an AML investigation. It requires formal language, specific facts, regulatory terminology, and a coherent structure. Your best analysts can do it in 2–3 hours. Your junior analysts take 4–5 hours and often get it wrong on the first draft.
>
> The AI does it in 30 seconds."

In the Generate Narrative modal:

**Select Type:** SAR Narrative (or STR Suspicion Basis — explain both)

Explain the narrative types briefly:
> "There are four types:
> - **Investigation Summary** — a comprehensive 5–8 paragraph narrative of everything that happened
> - **STR Suspicion Basis** — the specific text required for the FIU-IND STR form's 'Basis of Suspicion' field, using regulatory language
> - **SAR Narrative** — FinCEN-style narrative for institutions with international exposure
> - **Closure Summary** — a short summary for the case closure form explaining why the case was closed"

**Investigator Notes:**
```
Customer claimed agricultural income as source. No documentary evidence
provided. Beneficiary director Ramesh Patel shares surname with customer —
possible family connection not yet formally confirmed. Focus on structuring
pattern and rapid transfer to shell companies.
```

> "These notes give the AI additional context that isn't in the documents or findings. The AI will incorporate these points into the narrative."

Click **Generate Narrative**. Wait 20–30 seconds.

When the narrative appears, read a section aloud:

> "Listen to the language. This reads like a formal compliance document — because the AI was trained on regulatory filings. It mentions the specific PMLA section, it uses the correct terminology, it references the account numbers and amounts from our transaction records, it describes the pattern in formal language.
>
> Now watch what happens when I click Edit."

Click **Edit** on the narrative. Make a small change — rephrase a sentence or add a specific detail.

Click **Save Edit**.

> "The analyst can edit the AI output. They are not locked into what the AI wrote. They review it, improve it, and when they are satisfied — they approve it."

Click **Approve**:

> "Approved. This narrative is now locked. It has a timestamp, the approver's name, the AI model that generated it, the findings it was based on, and the transactions it referenced. Every word in this narrative is traceable to its source. An auditor can look at this narrative and see exactly where every fact came from."

Show the **AI Log** link on the narrative card:

> "The AI log ID — this links to the AI Logs tab where the complete record of this AI generation is stored. The exact prompt that was sent, the exact response that came back, the model version used, the timestamp. Permanent, immutable, auditable."

---

# ★ PART 6 — STR FILING
### "The Regulatory Filing — Done in Minutes, Not Days"

**Click path:** Log in as Manager (ravi.kumar@opsowl.dev) → Open the Kiran Patel case → **STR** tab

> "The STR tab is only visible to Manager, Admin, and Super Admin. Your analysts build the investigation, but the decision to file an STR is a management responsibility.
>
> This is OpsOwl's interpretation of the maker-checker principle — the same person who investigated cannot be the sole person who approves the filing."

Walk through each section of the STR form:

**Report Details:**
- Report Type: STR
- Reporting Period: 01 Jan 2024 — 31 Mar 2024

> "These fields are straightforward."

**Subject Information:**
Click **Auto-fill from Linked Subject** → Kiran Patel's information populates:

> "Because we linked Kiran Patel as a subject earlier, his name, PAN, address, and phone number auto-populate here. Zero retyping. The same information that's in the case is now in the STR form."

**Transaction Information:**
Click **Load from Case Transactions**:

> "The three transactions we logged earlier — the ₹52L deposit, the ₹19.5L transfer to Sunrise Trading, the ₹18L transfer to GreenLeaf — all load into the STR form automatically. Amounts, dates, account numbers, IFSC codes. Everything that the analyst spent time logging in the Transactions tab is now automatically in the STR form.
>
> This is the double data entry problem solved. Your analyst enters transaction data once. It flows into the STR automatically."

**Narrative / Suspicion Basis:**
Click **Load Approved Narrative**:

> "The narrative we just approved in the Narrative tab loads here automatically. The complete, well-written, evidence-grounded narrative that the AI drafted and the analyst approved."

**Filing Entity:**
Your institution details (pre-configured by admin):

> "These are pre-filled from your organisation's settings. Your institution's name, RBI code, compliance officer details — configured once by admin, populated automatically on every STR."

Click **Submit for Approval**:

> "The STR is now in 'Submitted' state. It needs a second manager or admin to review and approve before it can be exported. This is the maker-checker control. The person who submits cannot approve."

---

**Now log in as admin@opsowl.dev → open the same case → STR tab:**

> "As admin, I can see the STR that Ravi submitted. I review it — the narrative is accurate, the transaction amounts are correct, the subject details are complete."

Click **Approve**:

> "Approved. Now I have two export options."

Click **Export XML**:

> "This generates a goAML-compatible XML file — the exact technical format that FIU-IND's goAML portal accepts. Your compliance officer logs into the goAML portal, uploads this file, and the STR is filed. No manual data entry into the portal. No transcription errors. The file is generated directly from the case data."

Click **Export PDF**:

> "This generates a formatted PDF of the complete STR — all fields, the narrative, the transaction table. This is for your internal records. You file this in your records management system as evidence that the STR was submitted."

**Show the total time saved:**

> "Let me recap what just happened. The STR was:
> - Drafted by AI (30 seconds)
> - Reviewed and approved by analyst (10 minutes)
> - Reviewed and approved by manager (5 minutes)
> - Exported as XML and uploaded to goAML (2 minutes)
>
> Total: approximately 20 minutes from having all the evidence to submitting the STR.
>
> Your current process: 2–3 days of writing, reviewing, emailing back and forth, and manually entering data into goAML.
>
> That is not an incremental improvement. That is an order-of-magnitude change."

---

# PART 7 — CASE CLOSURE
### "Formally Closing the Book"

**Click path:** Case → **Closure** tab

> "A case status of 'Closed' is just a label. The Closure tab is the formal documentation of what the investigation found and what was decided.
>
> Think of it as the case's closing argument — the formal statement of outcome."

Fill in the closure form:

- **Outcome:** Suspicious Activity Confirmed
- **Disposition:** STR Filed
- **Enforcement Actions:** Account flagged for enhanced monitoring. Transaction limit set pending compliance review.
- **Closure Summary:**
  ```
  Investigation confirmed a classic AML structuring + layering pattern.
  Customer Kiran Patel made 11 cash deposits totalling ₹58,50,000 between
  January and March 2024, all below the CTR threshold. 100% of funds were
  transferred within 48 hours to three entities incorporated within 90
  days of the investigation period, with no verifiable business activity.
  Customer could not provide documentary evidence for source of funds.
  STR filed with FIU-IND on 13 Jun 2024 (reference: FIU-2024-INV-0052).
  Account placed under enhanced monitoring. Case closed.
  ```
- **Closed By:** Ravi Kumar (Manager)

Click **Submit Closure** → Approve as admin.

> "The case is now formally closed. The closure decision — who decided, what they decided, and why — is permanently on record. This is the document that answers 'what happened to this case?' for the next 10 years."

---

# ★ PART 8 — THE INTELLIGENCE LAYER
### "Management-Level Visibility Across Your Entire Program"

---

## 8a. Workload Dashboard

**Click path:** Sidebar → **Workload**

> "Workload shows you every person in your compliance team and their current load."

Walk through the table:

> "Each row is one analyst or manager. The columns tell you:
>
> - **Active Cases**: How many cases they currently own
> - **Open Tasks**: Total tasks assigned to them across all their cases
> - **Overdue Tasks**: Tasks that have passed their due date — a red flag
> - **Avg Case Age**: How many days on average their cases have been open before closing
> - **Load Bar**: A visual representation of their workload relative to others
>
> At a glance, I can see that Divya has 18 active cases and Arjun has 6. That imbalance is causing Divya's cases to move slower — which is why some of her cases might be approaching SLA breach.
>
> As a manager, I select 5 of Divya's least-urgent cases, click Bulk Reassign, and assign them to Arjun. The workload is now balanced. The whole operation took 60 seconds."

Scroll to the **AI Risk Leaderboard**:

> "This is the second half of the Workload page. The top 20 cases in your system ranked by AI risk score.
>
> If you are the MLRO and you have 500 active cases, you cannot review all 500. You review the ones with the highest risk. This list tells you which 20 cases are most dangerous right now, based on the AI's assessment of the documents and findings.
>
> These are the cases most likely to need STR filing. These are the cases that could become regulatory problems if not handled quickly."

---

## 8b. ★ MLRO Dashboard

**Click path:** Sidebar → **MLRO Dashboard**

> "This page is for one person: the MLRO. The Money Laundering Reporting Officer. It gives them a month-to-date view of how the compliance program is performing."

Walk through each KPI card:

**Cases Opened MTD:**
> "How many new investigations were started this month. If this number is much higher than last month, something changed in your transaction monitoring sensitivity — either more alerts are being generated or your team is being more diligent about escalating."

**Cases Closed MTD:**
> "Investigation throughput. How many cases did your team successfully close this month? If Cases Opened >> Cases Closed, your backlog is growing. That is a staffing or efficiency problem."

**STRs Filed MTD:**
> "The regulatory output metric. How many suspicious transaction reports did your team file with FIU-IND this month? Your MLRO is directly accountable to the board for this number. If it is zero but your team has been opening 50 cases a month, your investigators are not finding suspicious activity — or they are finding it but not filing."

**STRs Pending Review:**
> "This is the backlog of STRs that have been prepared by analysts but not yet approved by a manager. If this number is large, your management layer is a bottleneck. STRs are sitting ready to file but nobody has approved them. This is a process failure and a regulatory risk."

**SLA Breached (Open):**
> "The most important KPI on this page. The number of currently open cases that have already breached their investigation deadline. Every number here is a potential regulatory violation. This should be zero. If it is not zero, the MLRO needs to know the names of those cases and the reasons for the breach."

**Avg Case Age:**
> "How many days, on average, does it take your team to close an investigation? If this number is growing month over month, your team is getting slower — either due to increasing case volume, decreasing staff, or increasingly complex cases."

**Disposition Breakdown chart:**

> "Of all the cases closed this month, what were the outcomes? STR Filed, No Action, Account Closed, Referred to Law Enforcement?
>
> This is the quality metric for your investigations program. If 95% of cases are 'No Action', either your alert thresholds are generating too many false positives, or your analysts are closing cases too quickly without proper investigation.
>
> If 80% of cases end in STR Filing, either your transaction monitoring is extremely well-tuned, or your analysts are over-filing — which can draw negative attention from FIU-IND."

**Investigator Performance table:**

> "Every analyst and manager, with their monthly numbers:
> - Active cases right now
> - Cases closed this month
> - STRs filed this month
> - Open cases where SLA is already breached
>
> If analyst A has filed 8 STRs and analyst B has filed 0, and both have the same caseload, that is a discrepancy worth investigating. Is B missing suspicious activity? Are their cases genuinely less risky?
>
> This level of per-person performance visibility is not possible with spreadsheets. This is live, from actual case data, updated in real time."

---

# ★ PART 9 — ADMINISTRATION
### "Configuring the System to Match Your Organisation"

---

## 9a. ★ User Management

**Click path:** Sidebar → **Users**

> "Every person who has access to OpsOwl must be formally provisioned here. No shared logins. No 'borrow my password'. Every action in the system is tied to a specific, named, accountable user."

Show the users table. Point out columns:
- Full Name, Email, Role, Active/Inactive, Last Login, MFA Status

**Demo: Create a New User**

Click **+ Invite User**:

> "Creating a new user is 3 fields: their email, their full name, and their role. Click invite. They receive an email with a link to set their password. From that moment, their access is active and their actions are logged."

Walk through the roles:

> "**Viewer** — this is the role for your RBI auditor or board observer. They can see everything but cannot create, edit, or delete anything. Give your regulator a Viewer account during an audit. They browse the system themselves. Your team does zero preparation work.
>
> **Analyst** — your investigation staff. They create cases, upload documents, write notes, complete tasks, document findings and transactions. They cannot file STRs or approve AI outputs — that requires manager level.
>
> **Manager** — team leads and compliance managers. Everything the analyst can do, plus: file and approve STRs, reassign cases, view workload, approve AI outputs. The management layer.
>
> **Admin** — your compliance head or IT admin. Everything the manager can do, plus: user management, system configuration, bulk operations including bulk delete, data retention configuration.
>
> **Super Admin** — system administrator. Full access including hard-deleting records and purging user data. Only 1–2 people should have this role."

**Demo: MFA**

Point to the MFA column on a user:

> "Multi-Factor Authentication. For your compliance staff — especially those who can approve STRs and access sensitive investigation data — you can require MFA. When enabled, logging in requires their password plus a 6-digit code from their phone authenticator app.
>
> If a laptop is stolen, the attacker has the password but not the phone. MFA blocks access. Your investigation data is protected even if credentials are compromised."

---

## 9b. Audit Logs

**Click path:** Sidebar → **Audit Logs**

> "I want to spend time on this because this is the feature that determines whether you pass or fail a regulatory audit.
>
> Every single action taken in OpsOwl is recorded here. Not just logins. Not just case status changes. Every action: every note written, every document uploaded, every finding confirmed, every task completed, every STR approved, every user created, every password changed. Everything."

Show the log. Walk through a few entries:

**Entry 1: Case Created**
> "INV-0052 was created by admin@opsowl.dev on 13 Jun 2024 at 2:47pm from IP 192.168.1.50. This entry cannot be deleted or modified. It will be here in 10 years."

**Entry 2: STR Approved**
> "STR on INV-0052 was approved by ravi.kumar@opsowl.dev on 13 Jun 2024 at 4:23pm. The approver, the timestamp, the case reference — all permanent."

**Entry 3: Document Uploaded**
> "kiran_patel_account_statement.pdf was uploaded to INV-0052 by divya.reddy@opsowl.dev on 13 Jun 2024 at 2:55pm. The document was uploaded before the STR was approved — meaning the evidence existed before the filing. That matters."

Click **Export CSV**:

> "Export the complete audit log for any date range as a CSV file. This is what you hand to the RBI auditor. It is the signed, timestamped, complete record of every action taken in your compliance system."

Now explain the most important technical detail:

> "Here is what makes this different from any other system's audit log — including your CBS, your email system, and any other compliance tool you've used.
>
> This audit log is **cryptographically hash-chained**.
>
> I'll explain what that means in plain English. Each entry in the audit log contains a mathematical fingerprint of the previous entry. If you change entry number 500 — even by one character — entry number 501 breaks. Because entry 501's fingerprint includes entry 500, and now 500 is different.
>
> Think of it like a chain of padlocks. If you cut the lock in the middle, the chain falls apart. You can't silently remove a link.
>
> Why does this matter? Your DBA could delete a row from a normal audit log and nobody would ever know. In OpsOwl, any modification or deletion breaks the chain — and the system alerts immediately.
>
> When the RBI asks 'can you prove this audit log has not been tampered with?', the answer is yes — mathematically provable. That is a compliance differentiator that very few systems offer."

---

## 9c. Inbound Events

**Click path:** Sidebar → **Inbound Events**

> "Your core banking system, your transaction monitoring system, your fraud detection engine — all of these can send events to OpsOwl automatically.
>
> This page shows every event that has arrived. Think of it as the 'inbox' for your API."

Show an event row. Click to expand it:

> "Each event shows: when it arrived, which system sent it, the raw data payload, whether it was processed successfully, and if a case was created — the case number.
>
> If your CBS sent an alert at 3am and something went wrong — maybe the case type wasn't recognised — the event is still here. Your team can see it the next morning, understand what failed, and manually escalate it to a case. Nothing is lost."

---

## 9d. Alert Queue

**Click path:** Sidebar → **Alert Queue**

> "Inbound Events are the technical log. Alert Queue is the analyst-facing triage view.
>
> When your transaction monitoring system sends an alert, it doesn't automatically become a case — it lands in the Alert Queue first. Your analyst reviews each alert and makes a decision: is this worth a full investigation, or is it a false positive?
>
> For each alert in the queue, they see the key details: the account, the amount, the alert type, the triggering rule. They click either 'Escalate to Case' or 'Dismiss with Reason'."

Show an alert → demonstrate clicking **Escalate to Case**:

> "The alert just became a full investigation case. One click. The case is created, auto-assignment rules run, the analyst is assigned, the SLA countdown starts. The time from 'CBS fired an alert' to 'formal investigation case open' — under 2 minutes.
>
> Show me your current process and tell me how long that takes."

---

## 9e. Outbound Webhooks

**Click path:** Sidebar → **Webhooks** → **New Webhook**

> "Inbound events are your systems talking to OpsOwl. Outbound webhooks are OpsOwl talking to your other systems.
>
> When a case status changes to 'Escalated', OpsOwl can instantly notify your Slack channel, your risk dashboard, your management reporting system — any system that has a URL to receive data."

Show the webhook configuration:

- URL: Your system's endpoint URL
- Events: Select from the list

> "Available events:
> - **case_created**: When any new investigation is opened
> - **case_status_changed**: When a case moves through the workflow — open, in progress, escalated, closed
> - **str_filed**: When an STR is approved and exported
>
> Real-world use case: when an STR is filed, your MLRO's management reporting system gets a webhook notification and automatically updates the monthly compliance KPI dashboard. Zero manual reporting. Zero delay."

---

## 9f. Scheduled Reports

**Click path:** Sidebar → **Scheduled Reports** → **New Report**

Configure a report:

- **Report Type:** SLA Summary
- **Frequency:** Weekly — every Monday at 08:00am
- **Email Recipients:** mlro@yourbank.com, compliance-head@yourbank.com

> "Every Monday at 8am, the MLRO's inbox receives a CSV with the previous week's complete SLA performance data — cases opened, cases closed, SLA breaches by analyst, average case age, STRs filed.
>
> Nobody created this report. Nobody exported it. Nobody emailed it. The system generated it from live data and sent it automatically.
>
> The MLRO reads the report over their morning chai and already knows the state of the compliance program before their first call of the day."

Available report types:
- **Active Cases** — full list of all open cases with SLA status
- **SLA Summary** — breach statistics by analyst and case type
- **Team Workload** — per-analyst caseload and task completion rates
- **STR Filing Summary** — all STRs filed in the period with case references

---

## 9g. API Keys

**Click path:** Sidebar → **API Keys** → **Create API Key**

> "API keys allow external systems to talk to OpsOwl programmatically — without a human logging in.
>
> Use cases:
> - Your CBS creates an investigation case automatically when a high-risk transaction is detected
> - Your document management system uploads a document directly to an OpsOwl case
> - Your BI tool queries OpsOwl for case status to update your executive dashboard
>
> Each key has a scope — Read, Read-Write, or Admin. Your CBS integration gets a Read-Write key to create cases but cannot delete them. Your BI tool gets a Read-only key. If a key is compromised, revoke it here in 2 seconds. The integration stops working immediately — no downtime, no password reset process."

---

## 9h. Data Retention

**Click path:** Sidebar → **Data Retention**

> "Two conflicting regulations:
>
> The **DPDP Act 2023** (India's new data protection law) says: don't hold personal data longer than necessary.
>
> The **PMLA** (Prevention of Money Laundering Act) says: keep all AML records for 10 years.
>
> OpsOwl resolves this automatically. For STR filings, AML case documents, and audit logs — the minimum retention is hard-coded at 10 years. Even your Super Admin cannot set it lower. The law requires it, and the system enforces it.
>
> For other data types — internal notes on low-risk cases, non-regulated documents, draft communications — you configure the retention period yourself. Your DPO can define the policy here, and OpsOwl enforces it."

---

# ★ PART 10 — SEARCH
### "Find Anything in the Entire System in Under 5 Seconds"

**Click path:** Sidebar → **Search**

> "Every piece of text in OpsOwl is searchable — not just case titles, not just document names, but the content inside documents."

**Demo 1:** Type `9876543210` (account number)

> "I searched for an account number. OpsOwl found it in: 3 cases where it was mentioned in notes, 2 documents where it appeared in bank statements, and the subject profile where it was linked.
>
> Your analyst gets a call: 'Account 9876543210 just appeared in a new suspicious transaction.' They type the account number in Search. In 3 seconds they see: this account has been investigated before, it was part of the Kiran Patel case, an STR was filed. They immediately have context before they even open the new alert."

**Demo 2:** Type `ABCDE1234F` (PAN number)

> "PAN number search. Found in the KYC document uploaded to INV-0052 and in the subject profile. If this PAN appears in any future case, OpsOwl will find it."

**Demo 3:** Type `GreenLeaf Commodities`

> "Company name search. Found in the investigation note Divya wrote, in the transaction records, in the findings. Everything related to this company, across all cases, in one view."

**Why this matters:**

> "Your team is investigating 500 cases a year. After 3 years, you have 1,500 cases, 15,000 documents, and 50,000 notes in the system. Without search, finding any specific piece of information means opening cases one by one.
>
> With full-text search, the collective memory of 3 years of investigations is searchable in under 5 seconds. That institutional knowledge is now a competitive advantage, not a buried archive."

---

# PART 11 — THE CASE TIMELINE
### "The Complete, Uneditable Investigation History"

**Click path:** Open any case → **Timeline** tab

> "The Timeline is the case's complete chronological history. Every action by every user — in time order. Not just the important events — every event."

Scroll through the timeline:

- Case INV-0052 created by admin@opsowl.dev — 13 Jun 2:47pm
- Case assigned to divya.reddy@opsowl.dev — 13 Jun 2:47pm
- Document uploaded: kiran_patel_account_statement.pdf — 13 Jun 2:55pm
- AI analysis initiated — 13 Jun 2:58pm
- Note added by divya.reddy@opsowl.dev — 13 Jun 3:15pm
- Task completed: Verify customer KYC — 13 Jun 3:22pm
- Communication logged: Phone call with Kiran Patel — 13 Jun 3:30pm
- Finding confirmed: Structuring pattern — 13 Jun 3:45pm
- Case status changed to In Progress — 13 Jun 3:46pm
- STR submitted by ravi.kumar@opsowl.dev — 13 Jun 4:20pm
- STR approved by admin@opsowl.dev — 13 Jun 4:23pm
- Case closed — 13 Jun 5:00pm

> "This is the complete story of this investigation. In time order. Every entry shows who did what and when.
>
> Use case 1 — handover: Divya goes on leave, Arjun takes over. He opens the Timeline. In 2 minutes he knows everything that happened, in what order, by whom. No handover meeting needed.
>
> Use case 2 — audit: The RBI auditor asks 'at what point did your team become suspicious of this customer?' The Timeline shows the note at 3:15pm documenting the suspicion — and the STR submission at 4:20pm, which is well within the 7-day window. Case proven.
>
> Use case 3 — dispute: A former employee claims they never approved an STR. The Timeline shows the exact timestamp and IP address of the approval. Dispute resolved."

---

# PART 12 — PDF EXPORT
### "One Button, Complete Case Summary"

**Click path:** Any case → click the **📄 PDF** button in the header

Wait for the PDF to generate. Open it.

> "One button click produces a complete, formatted PDF of the entire case — every section:
> - Case details and metadata
> - Full timeline
> - All investigation notes
> - Findings with evidence links
> - Transaction records
> - Documents list
> - AI insights summary
> - Communications log
> - STR filing reference (if applicable)
>
> This is the document you email to your internal legal team when they need to review an investigation. This is the document you give to external counsel if the case goes to prosecution. This is the document you give to the board if they ask for a specific case review.
>
> Your analyst does not need to compile anything. One button, 10 seconds."

---

# ★ PART 13 — SECURITY ARCHITECTURE
### "Answering the 'Is Our Data Safe?' Question"

> "Before I hand over to questions, I want to address the question that every CTO asks but sometimes hesitates to ask directly: is our data safe with this system?"

---

**Where does the data live?**

> "OpsOwl is designed for on-premise deployment. Your data lives on your servers, in your data centre, behind your firewall. We do not host your data. We do not have access to your data. There is no SaaS component where your investigation data is in someone else's cloud.
>
> We provide: Docker containers, a deployment guide, and configuration support. Your IT team deploys it on your own infrastructure, the same way they deploy any other internal application."

**What about the AI?**

> "The AI runs on your infrastructure too. By default, we use Qwen 2.5 — an open-source AI model that runs entirely on your servers. When your analyst asks a question about a document, the question goes from their browser to your server, and the answer comes back from your server. Nothing leaves your network.
>
> For organisations that want better AI quality and are willing to use a cloud AI service, we support any OpenAI-compatible API. But that is your choice — it is not the default, and it is never mandatory."

**What about authentication?**

> "Every session requires email + password authentication. Passwords are bcrypt-hashed — even we cannot read them. MFA is available for all accounts. Sessions have a maximum lifetime — configurable, typically 8 or 24 hours. If an analyst's session is not used for 30 minutes, it expires.
>
> After 5 failed login attempts, the account is locked and requires admin intervention to unlock. Brute force attacks are blocked."

**What about the audit trail — can it be tampered with?**

> "The audit log is hash-chained, as I explained earlier. Every entry contains a cryptographic hash of the previous entry. You cannot silently delete or modify any entry. If someone tries, the chain breaks and the system detects it immediately.
>
> In practical terms: your own DBA cannot delete a log entry without detection. This is important because in regulatory investigations, the first thing an auditor looks for is evidence of record-tampering. Our system makes that impossible."

**What about data retention and DPDP?**

> "As I showed in the Data Retention section: PMLA-required records are locked at 10-year minimum retention. You cannot set it lower. Non-regulated data can be configured per your DPO's requirements. The Right-to-Erasure under DPDP is supported — with a specific exception for customers named in STR filings, because PMLA prohibits erasing those records regardless of the customer's request."

---

# CLOSING — THE BUSINESS CASE
### "Why OpsOwl Pays for Itself"

---

## Time Savings — Per Investigation

| Activity | Today (Manual) | With OpsOwl | Time Saved |
|----------|---------------|-------------|------------|
| Document review and entity extraction | 2–3 hours | 20–30 seconds (AI) | ~2.5 hours |
| Investigation note writing | 45–60 minutes | 10–15 minutes (AI draft + edit) | ~45 minutes |
| STR narrative writing | 2–3 hours | 20–30 minutes (AI draft + review) | ~2 hours |
| goAML data entry (manual re-entry) | 1 hour | 2 minutes (XML auto-export) | ~58 minutes |
| Morning status updates to management | 30 minutes/day | 0 (live dashboard) | 30 minutes/day |
| Case handover to new analyst | 1 hour | 0 (Timeline self-explains) | 1 hour |
| RBI audit preparation | 2 weeks | 0 (auditor browses live system) | ~10 days |
| **Total per complex AML investigation** | **~7–9 hours** | **~1.5–2 hours** | **~6.5 hours** |

> "If your team processes 80 complex AML investigations per month, that is 520 hours saved per month. At ₹600/hour fully loaded cost for a compliance analyst, that is ₹3.1 lakhs per month in direct productivity savings. OpsOwl typically costs less than half that per month for a team of 20 analysts."

---

## Risk Reduction — The Compliance Insurance

> "These numbers are harder to put a precise figure on, but they are arguably more important than the productivity savings."

**Missed STR deadlines:**
> "One STR filed late is an RBI observation. Three STR delays in a year is a show-cause notice. Five delays is a penalty — ₹1L to ₹10L per violation under PMLA. One serious breach can cost you ₹50L in penalties plus the management distraction of a 3-month regulatory response process.
>
> OpsOwl's SLA tracking means your team has a real-time countdown on every case. The MLRO sees breaches the moment they happen, not on Monday morning. The compliance cost of one avoided breach pays for OpsOwl for a year."

**Audit failures:**
> "A failed RBI IT audit can result in: a public censure (visible to customers and investors), restrictions on new product launches, mandatory third-party reviews at your cost, and in severe cases, fines under Section 12 of PMLA.
>
> OpsOwl's hash-chained audit log, complete document trail, and maker-checker workflows are designed specifically to pass these audits. Your system administrator can demonstrate regulatory compliance with a live demonstration in an hour — not a 2-week documentation exercise."

**Staff turnover:**
> "The average tenure of a compliance analyst in India is 18–24 months. Every time a senior analyst leaves, they take 5 years of institutional knowledge with them — which customers they investigated, what patterns they noticed, what cases are connected to each other.
>
> OpsOwl makes that knowledge explicit and permanent. Every analysis, every note, every finding, every cross-case connection stays in the system when the analyst leaves. Your institutional memory is now in software, not in people's heads."

---

## Audit Readiness — The Regulator Test

> "I want to close with one concrete scenario that will resonate with any compliance head.
>
> The RBI gives you 2 weeks notice of an IT and compliance systems audit. Currently, your team spends those 2 weeks: pulling case data from multiple systems, formatting it into reports, printing case files, creating an evidence binder for each case to be reviewed, preparing answers to likely auditor questions. Your analysts spend 80% of their time during those 2 weeks not on investigations — on audit preparation.
>
> With OpsOwl: you create a Viewer account for the audit team. You give them the URL. They log in on day 1 of the audit and browse the system themselves. They can search any case, any document, any action. The audit log exports in one click. Every case has a complete timeline. Every STR has an unbroken evidence chain from documents to findings to narrative to filing.
>
> Your analysts spend those 2 weeks doing investigations. Not preparing for the audit. The system is permanently audit-ready."

---

## Summary — What You Just Saw

> "Let me recap the complete investigation we walked through today:
>
> 1. **Alert arrived** from CBS → landed in Alert Queue
> 2. **Analyst triaged** the alert → escalated to formal Case
> 3. **Case auto-assigned** to Divya by rule → no manager involvement needed
> 4. **Template applied** → 9 tasks created, description filled in automatically
> 5. **Documents uploaded** → AI extracted all entities in 30 seconds
> 6. **Communication logged** → customer call permanently on record
> 7. **Notes added** → analysis visible to entire team in real time
> 8. **Tasks completed** → every step timestamped and assigned
> 9. **Findings documented** → evidence-linked formal conclusions
> 10. **Transactions recorded** → flow of funds mapped precisely
> 11. **Subject linked** → cross-case profile updated
> 12. **AI Risk Analysis** → risk score HIGH 74, entities extracted
> 13. **AI Copilot** answered specific questions in 3 seconds
> 14. **AI Agent** ran autonomous investigation, produced full report in 60 seconds
> 15. **AI Narrative generated** → STR draft in 30 seconds
> 16. **Narrative reviewed and approved** by analyst → 10 minutes
> 17. **STR submitted** by manager → maker-checker enforced
> 18. **STR approved** by admin → goAML XML exported
> 19. **Case formally closed** with documented disposition
> 20. **MLRO Dashboard** updated live → no data entry required
> 21. **Audit Log** recorded every step → hash-chained, tamper-evident
>
> Complete investigation, completely documented, completely auditable.
>
> From alert to STR filed: **under 3 hours** with OpsOwl.
> Industry average without OpsOwl: **3–5 business days**.
>
> That is the OpsOwl difference. Not an incremental improvement — a fundamental change in how your compliance team operates."

---

## The Three Questions to Close On

Before handing over to Q&A, ask these three closing questions:

> "Based on what you've seen today:
>
> **Question 1:** What would your MLRO say if they could see the Attention Required panel and the MLRO Dashboard live, instead of waiting for the Monday morning report?
>
> **Question 2:** How much time does your team currently spend preparing for RBI audits? What would you do with that time if the system was permanently audit-ready?
>
> **Question 3:** Has your team ever missed an STR deadline? If yes — what would it have meant to have a live SLA countdown on every case?"

---

## Common CTO Technical Questions — Prepared Answers

**Q: What is the tech stack? Our IT team will ask.**
> Backend: Go (a compiled language — high performance, low memory). Database: PostgreSQL. Frontend: Next.js (React). AI: runs on any GPU/CPU server supporting PyTorch. All containerised with Docker. Deployable on any Linux server, on-premise or private cloud.

**Q: What are the minimum server requirements?**
> For a team of up to 25 concurrent users: 4 CPU cores, 8GB RAM, 100GB SSD storage. For 25–100 concurrent users: 8 cores, 16GB RAM, 500GB storage. The AI service requires an additional 4–8 cores or a GPU for faster inference.

**Q: How does it integrate with our CBS?**
> Two integration patterns: (1) Inbound webhook — your CBS calls our REST API endpoint when it generates an alert. A 1-day integration project for your CBS team. (2) API key — your CBS uses our API key to create cases, upload documents, or query case status. Fully documented OpenAPI spec provided.

**Q: What happens if the system goes down?**
> Your investigations are already stored in the database. When the system comes back up, they're exactly where you left them. For uptime guarantees: since it runs on your infrastructure, SLA is determined by your infrastructure. Most clients achieve 99.9% on a 2-server active-passive setup. We provide health check endpoints, monitoring runbook, and Docker health probes.

**Q: Can we migrate our existing cases from Excel or our current system?**
> Yes. We provide a CSV import template. Your team exports their current case data to the template format and our system imports it. Historical cases become searchable, can be reopened, and are included in analytics. Migration typically takes 1–2 days for teams with up to 5,000 historical cases.

**Q: How long does deployment take?**
> Standard deployment: 1 day for a technical resource following our deployment guide. Initial configuration (SLA policies, user accounts, templates, assignment rules): 2–3 hours. First live investigation: same day. Full team onboarding: 1 week of guided use.

**Q: What does licensing look like?**
> On-premise perpetual license. You pay once, you own it. No per-user fees, no transaction fees, no API call fees. Annual maintenance covers updates, security patches, and support. Contact us for pricing based on your team size.

**Q: What if we need a feature you don't have?**
> OpsOwl is built with an open module architecture. Custom features — specific report formats for your regulator, integration with a specific CBS, custom fields for your case types — are available as professional services engagements. The typical custom feature takes 2–4 weeks.

---

*OpsOwl AI — AI-Native Enterprise Investigation Workspace*
*Contact: reddybasha524@gmail.com*
*Deployment: On-Premise · Data never leaves your infrastructure*
*Version: 1.0 · Built for PMLA, RBI, FIU-IND compliance · June 2026*
