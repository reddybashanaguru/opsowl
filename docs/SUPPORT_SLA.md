# OpsOwl AI — Support SLA

**Effective:** June 2026 | **Applies to:** All annual license customers

---

## Support Tiers

### Priority 1 — Production Down

**Definition:** OpsOwl is completely inaccessible, or a critical workflow is broken for all users (cannot log in, cannot create cases, audit log not recording, data loss occurring).

| Metric | Commitment |
|--------|-----------|
| Initial response | **4 business hours** |
| Status update frequency | Every 2 hours until resolved |
| Target resolution | **1 business day** |
| Support channels | Email + phone (dedicated P1 hotline) |
| Who handles it | Senior engineer, escalated immediately |

**What qualifies:**
- Web UI completely inaccessible (5xx on all pages)
- Cannot log in (authentication service down)
- Database corruption or data loss
- API returning errors on all requests
- Audit log chain broken / hash validation failing

**What does NOT qualify as P1:**
- Slow performance
- One user cannot log in (while others can)
- AI analysis feature not working (workaround: manual investigation)

---

### Priority 2 — Feature Broken

**Definition:** A specific feature is broken or significantly degraded for some or all users, but a workaround exists and cases can still be managed.

| Metric | Commitment |
|--------|-----------|
| Initial response | **1 business day** |
| Status update frequency | Every 2 business days |
| Target resolution | **5 business days** |
| Support channels | Email + Help Desk portal |
| Who handles it | Support engineer |

**What qualifies:**
- Document upload failing for some file types
- AI Copilot or AI analysis returning errors
- Email notifications not sending
- LDAP authentication broken (users can still use local passwords)
- Export / report generation failing
- Specific browser (Chrome/Edge/Firefox) showing UI bugs

---

### Priority 3 — Query / How-To / Enhancement

**Definition:** Questions about product usage, configuration help, or feature requests.

| Metric | Commitment |
|--------|-----------|
| Initial response | **3 business days** |
| Target resolution | **10 business days** (or best-effort for enhancements) |
| Support channels | Email + Help Desk portal |
| Who handles it | Support team |

**What qualifies:**
- "How do I configure LDAP?"
- "Can you help me set up scheduled reports?"
- "Is there a way to bulk-import cases?"
- Feature requests and suggestions
- Best-practice questions

---

## Business Hours

| Region | Hours | Time Zone |
|--------|-------|-----------|
| India (standard) | 9:00 AM – 7:00 PM, Monday – Friday | IST (UTC+5:30) |
| India (24×7 add-on) | 24 hours, 7 days | IST |

**Public holidays:** OpsOwl observes major Indian public holidays (Republic Day, Holi, Independence Day, Diwali, Christmas). P1 issues during holidays are handled with a 6-hour response commitment via the emergency line.

---

## How to Raise a Support Request

### Email
Send to: **support@opsowl.dev**

Include in your email:
1. Priority (P1 / P2 / P3) with a one-line description
2. OpsOwl version (`grep OPSOWL_VERSION /opt/opsowl/.env`)
3. Steps to reproduce
4. What you expected vs. what happened
5. Relevant logs (`docker compose logs api --tail=100`)

### Help Desk Portal
Submit at: **support.opsowl.dev** (login with your OpsOwl admin email)

### P1 Emergency Hotline
Phone number provided to your named contact at contract signing. For P1 only.

---

## What Is Included in Annual Support

- Unlimited P1/P2/P3 support requests
- All software updates (monthly releases + security patches)
- Critical security patches within 72 hours of CVE disclosure
- Quarterly customer success review call
- Access to documentation and knowledge base

## What Is Not Included

- Custom development work (billed separately at ₹8,000/hr)
- On-site visits (₹1,00,000 + travel at actuals)
- Training beyond the initial deployment package
- Data recovery caused by customer actions (e.g., accidental deletion, hardware failure outside OpsOwl's scope)

---

## Escalation Path

| Step | Contact | When |
|------|---------|------|
| 1 | Support ticket | Default |
| 2 | Customer Success Manager (CSM) | P2+ unresolved past SLA |
| 3 | VP Engineering | P1 unresolved after 8 hours |
| 4 | CEO | Critical business impact, contractual dispute |

Your named CSM contact is provided at contract signing.

---

## SLA Exceptions

OpsOwl is not liable for SLA breaches caused by:
- Customer-side network issues (ISP outage, firewall misconfiguration)
- Hardware failure on customer-managed servers
- Customer's own modifications to OpsOwl configuration or code
- Force majeure events
- Third-party service outages (LDAP server, email relay, etc.)
- Maintenance windows with 48-hour advance notice

---

## SLA Credits

If OpsOwl breaches a P1 response SLA:

| Breach | Credit |
|--------|--------|
| Response > 4 business hours | 2% of monthly license fee |
| Response > 8 business hours | 5% of monthly license fee |
| Daily credits are capped at the monthly license fee |

Credits are applied to the next renewal invoice. They are not payable as cash.

---

*This SLA document is incorporated by reference into the OpsOwl Software License Agreement.
For questions: legal@opsowl.dev*
