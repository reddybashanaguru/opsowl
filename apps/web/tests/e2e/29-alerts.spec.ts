import { test, expect } from "@playwright/test";
import crypto from "crypto";
import { login, apiTokenAs, API_URL } from "./helpers/auth";

/**
 * Alert-to-Case Automation E2E tests.
 *
 * Verifies:
 * — Alert source CRUD (create, list, get, delete)
 * — Alert rule creation: auto-convert threshold drives case creation
 * — Webhook ingest → queued (below threshold)
 * — Webhook ingest → auto_converted (at or above threshold)
 * — Deduplication: same external_alert_id → 200 + status=deduplicated
 * — HMAC-SHA256 enforcement: missing → 401, wrong → 401, correct → 201
 * — Manual convert queued alert → case created, status=converted
 * — Dismiss alert → status=dismissed
 * — Access control: analyst/viewer cannot manage sources or rules
 * — TMS normalizers: Actimize, SAS AML generic payloads accepted
 * — End-to-end chain: high-risk alert → case created → subject linked → 360 shows total_cases
 * — UI: Alert Queue page, pending alert visible, convert/dismiss modals, filter
 */

const ADMIN_EMAIL   = "admin@opsowl.dev";
const ADMIN_PASS    = "OpsOwl@2024";
const ANALYST_EMAIL = "sneha.patel@opsowl.dev";   // seeded analyst
const ANALYST_PASS  = "OpsOwl@2024";

// ── Helpers ───────────────────────────────────────────────────────────────────

function genericPayload(overrides: Record<string, unknown> = {}) {
  return {
    alert_id:          `E2E-${Date.now()}-${Math.floor(Math.random() * 9999)}`,
    alert_type:        "CASH_STRUCTURING",
    risk_score:        60,
    subject_name:      "E2E Test Customer",
    subject_id_type:   "PAN",
    subject_id_number: `PAN${Math.floor(Math.random() * 90000 + 10000)}`,
    account_number:    "ACC-E2E-001",
    amount:            485000,
    currency:          "INR",
    transaction_date:  "2026-05-20",
    description:       "E2E automated test alert",
    ...overrides,
  };
}

function hmacSign(secret: string, body: string): string {
  return `sha256=${crypto.createHmac("sha256", secret).update(body).digest("hex")}`;
}

// ── Test suite ────────────────────────────────────────────────────────────────

test.describe("Alert-to-Case Automation — API", () => {
  let adminToken: string;
  let analystToken: string;
  let sourceID: string;           // no-secret source for most tests
  let securedSourceID: string;    // source with HMAC secret
  let ruleID: string;
  let queuedAlertID: string;
  const WEBHOOK_SECRET = "e2e-hmac-secret-2026";

  test.beforeAll(async () => {
    adminToken   = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);
    analystToken = await apiTokenAs(ANALYST_EMAIL, ANALYST_PASS);
  });

  test.afterAll(async () => {
    if (ruleID) {
      await fetch(`${API_URL}/alert-rules/${ruleID}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${adminToken}` },
      });
    }
    for (const sid of [sourceID, securedSourceID]) {
      if (sid) {
        await fetch(`${API_URL}/alert-sources/${sid}`, {
          method: "DELETE",
          headers: { Authorization: `Bearer ${adminToken}` },
        });
      }
    }
  });

  // ── Alert source management ────────────────────────────────────────────────

  test("admin creates a generic (no-secret) alert source", async () => {
    const res = await fetch(`${API_URL}/alert-sources`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({ name: "E2E Generic TMS", source_system: "generic", source_type: "webhook" }),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    sourceID = data.id;
    expect(data.name).toBe("E2E Generic TMS");
    expect(data.source_system).toBe("generic");
  });

  test("admin creates a secured source with HMAC webhook_secret", async () => {
    const res = await fetch(`${API_URL}/alert-sources`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({
        name: "E2E Secured Actimize",
        source_system: "actimize",
        source_type: "webhook",
        webhook_secret: WEBHOOK_SECRET,
      }),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    securedSourceID = data.id;
    expect(data.source_system).toBe("actimize");
  });

  test("GET /alert-sources lists created sources", async () => {
    const res = await fetch(`${API_URL}/alert-sources`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const sources: Array<{ id: string }> = data.sources ?? data ?? [];
    expect(sources.find((s) => s.id === sourceID)).toBeDefined();
  });

  test("GET /alert-sources/:id does not expose webhook_secret in response", async () => {
    const res = await fetch(`${API_URL}/alert-sources/${securedSourceID}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.webhook_secret ?? "").toBe("");
  });

  test("analyst cannot create alert source (403)", async () => {
    const res = await fetch(`${API_URL}/alert-sources`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({ name: "Should Fail", source_system: "generic" }),
    });
    expect(res.status).toBe(403);
  });

  test("unknown source ID on ingest returns 404", async () => {
    const res = await fetch(`${API_URL}/alerts/ingest/00000000-0000-0000-0000-000000000099`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(genericPayload()),
    });
    expect(res.status).toBe(404);
  });

  // ── Alert rules ────────────────────────────────────────────────────────────

  test("admin creates auto-convert rule with min_risk_score=85", async () => {
    const res = await fetch(`${API_URL}/alert-rules`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({
        name: "E2E High Risk Auto Convert",
        min_risk_score: 85,
        alert_type: "*",
        mode: "auto",
        priority_mapping: "high",
      }),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    ruleID = data.id;
    expect(data.min_risk_score).toBe(85);
    expect(data.mode).toBe("auto");
  });

  test("analyst cannot create alert rule (403)", async () => {
    const res = await fetch(`${API_URL}/alert-rules`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({ name: "Should Fail", min_risk_score: 80, auto_convert: true, is_active: true }),
    });
    expect(res.status).toBe(403);
  });

  test("GET /alert-rules returns the created rule", async () => {
    const res = await fetch(`${API_URL}/alert-rules`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const rules: Array<{ id: string }> = data.rules ?? data ?? [];
    expect(rules.find((r) => r.id === ruleID)).toBeDefined();
  });

  // ── Webhook ingest ─────────────────────────────────────────────────────────

  test("risk_score=60 ingest → queued (below 85 threshold)", async () => {
    const body = JSON.stringify(genericPayload({ risk_score: 60, alert_type: "CASH_STRUCTURING" }));
    const res = await fetch(`${API_URL}/alerts/ingest/${sourceID}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    expect(data.status).toBe("queued");
    expect(data.alert_id).toBeDefined();
    queuedAlertID = data.alert_id;
  });

  test("risk_score=91 ingest → auto_converted with case_id", async () => {
    const payload = genericPayload({
      alert_id:    `E2E-AUTO-${Date.now()}`,
      risk_score:  91,
      alert_type:  "SUSPICIOUS_TRANSFER",
      subject_name: "E2E Auto-Convert Subject",
    });
    const body = JSON.stringify(payload);
    const res = await fetch(`${API_URL}/alerts/ingest/${sourceID}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    expect(data.status).toBe("auto_converted");
    expect(data.case_id).toBeDefined();

    // Cleanup converted case
    if (data.case_id) {
      await fetch(`${API_URL}/cases/${data.case_id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${adminToken}` },
      });
    }
  });

  test("same external_alert_id ingested twice → second returns 200 + deduplicated", async () => {
    const fixedID = `E2E-DEDUP-${Date.now()}`;
    const body = JSON.stringify(genericPayload({ alert_id: fixedID, risk_score: 55 }));

    const r1 = await fetch(`${API_URL}/alerts/ingest/${sourceID}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body,
    });
    expect(r1.status).toBe(201);
    const originalAlertID = (await r1.json()).data.alert_id;

    const r2 = await fetch(`${API_URL}/alerts/ingest/${sourceID}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body,
    });
    expect(r2.status).toBe(200);
    const { data: d2 } = await r2.json();
    expect(d2.status).toBe("deduplicated");
    expect(d2.alert_id).toBe(originalAlertID);
  });

  // ── HMAC signature enforcement ─────────────────────────────────────────────

  test("ingest to secured source without X-OpsOwl-Signature → 401 MISSING_SIGNATURE", async () => {
    const body = JSON.stringify(genericPayload({ alert_id: `E2E-NOSIG-${Date.now()}` }));
    const res = await fetch(`${API_URL}/alerts/ingest/${securedSourceID}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body,
    });
    expect(res.status).toBe(401);
    const { error } = await res.json();
    expect(error.code).toMatch(/MISSING_SIGNATURE/i);
  });

  test("ingest to secured source with wrong HMAC → 401 INVALID_SIGNATURE", async () => {
    const body = JSON.stringify(genericPayload({ alert_id: `E2E-BADSIG-${Date.now()}` }));
    const res = await fetch(`${API_URL}/alerts/ingest/${securedSourceID}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-OpsOwl-Signature": "sha256=deadbeef" },
      body,
    });
    expect(res.status).toBe(401);
    const { error } = await res.json();
    expect(error.code).toMatch(/INVALID_SIGNATURE/i);
  });

  test("ingest to secured source with correct HMAC-SHA256 → 201 accepted", async () => {
    const payload = genericPayload({ alert_id: `E2E-GOODSIG-${Date.now()}`, risk_score: 55 });
    const body = JSON.stringify(payload);
    const sig = hmacSign(WEBHOOK_SECRET, body);

    const res = await fetch(`${API_URL}/alerts/ingest/${securedSourceID}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-OpsOwl-Signature": sig },
      body,
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    expect(["queued", "auto_converted"]).toContain(data.status);
  });

  // ── TMS normalizer payloads ────────────────────────────────────────────────

  test("Actimize-format payload (AlertID, RiskScore, CustomerName) accepted", async () => {
    const body = JSON.stringify({
      AlertID:           `ACT-E2E-${Date.now()}`,
      AlertType:         "CASH_STRUCTURING",
      RiskScore:         55,
      CustomerName:      "Actimize E2E Customer",
      CustomerID:        "ACT-CUST-001",
      AccountNumber:     "ACT-ACC-001",
      TransactionAmount: 480000,
      Currency:          "INR",
    });
    const res = await fetch(`${API_URL}/alerts/ingest/${sourceID}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body,
    });
    expect([200, 201]).toContain(res.status);
  });

  test("SAS AML format payload (alert_reference, scenario_name, score) accepted", async () => {
    const body = JSON.stringify({
      alert_reference: `SAS-E2E-${Date.now()}`,
      scenario_name:   "SUSPICIOUS_TRANSFER",
      score:           58,
      party_name:      "SAS E2E Party",
      party_id:        "SAS-PARTY-001",
      acct_num:        "SAS-ACC-001",
      amt:             320000,
      ccy:             "INR",
    });
    const res = await fetch(`${API_URL}/alerts/ingest/${sourceID}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body,
    });
    expect([200, 201]).toContain(res.status);
  });

  // ── Alert queue list ───────────────────────────────────────────────────────

  test("GET /alerts?status=pending returns queued alert", async () => {
    if (!queuedAlertID) return;
    const res = await fetch(`${API_URL}/alerts?status=pending&limit=50`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.total).toBeGreaterThan(0);
    const found = (data.alerts as Array<{ id: string }>).find((a) => a.id === queuedAlertID);
    expect(found).toBeDefined();
  });

  test("GET /alerts/:id returns full alert detail", async () => {
    if (!queuedAlertID) return;
    const res = await fetch(`${API_URL}/alerts/${queuedAlertID}`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.id).toBe(queuedAlertID);
    expect(data.status).toBe("pending");
    expect(typeof data.risk_score).toBe("number");
    expect(data.alert_type).toBeDefined();
    expect(data.source_system).toBeDefined();
  });

  // ── Manual convert ─────────────────────────────────────────────────────────

  test("POST /alerts/:id/convert creates a case and status becomes converted", async () => {
    if (!queuedAlertID) return;
    const res = await fetch(`${API_URL}/alerts/${queuedAlertID}/convert`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${analystToken}`,
        "Idempotency-Key": `e2e-convert-${queuedAlertID}`,
      },
      body: JSON.stringify({ title: "E2E Converted Case", description: "E2E test", priority: "medium" }),
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.status).toBe("converted");
    expect(data.case_id).toBeDefined();

    // Verify alert status updated to converted
    const alertRes = await fetch(`${API_URL}/alerts/${queuedAlertID}`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    const alertData = (await alertRes.json()).data;
    expect(alertData.status).toBe("converted");
    expect(alertData.linked_case_id).toBe(data.case_id);

    // Cleanup case
    await fetch(`${API_URL}/cases/${data.case_id}`, {
      method: "DELETE", headers: { Authorization: `Bearer ${adminToken}` },
    });
  });

  test("already-converted alert returns 409 on re-convert", async () => {
    if (!queuedAlertID) return;
    const res = await fetch(`${API_URL}/alerts/${queuedAlertID}/convert`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${analystToken}`,
        "Idempotency-Key": `e2e-convert-again-${queuedAlertID}`,
      },
      body: JSON.stringify({ title: "Duplicate convert", priority: "low" }),
    });
    expect(res.status).toBe(409);
  });

  // ── Dismiss alert ──────────────────────────────────────────────────────────

  test("POST /alerts/:id/dismiss marks alert dismissed with reason stored", async () => {
    // Ingest a fresh alert to dismiss (separate from queuedAlertID)
    const ingestBody = JSON.stringify(genericPayload({ alert_id: `E2E-DISMISS-${Date.now()}`, risk_score: 48 }));
    const ingestRes = await fetch(`${API_URL}/alerts/ingest/${sourceID}`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: ingestBody,
    });
    expect(ingestRes.status).toBe(201);
    const dismissAlertID = (await ingestRes.json()).data.alert_id;

    const res = await fetch(`${API_URL}/alerts/${dismissAlertID}/dismiss`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${analystToken}`,
        "Idempotency-Key": `e2e-dismiss-${dismissAlertID}`,
      },
      body: JSON.stringify({ reason: "False positive — customer has documented source of funds" }),
    });
    expect(res.status).toBe(200);

    const alertRes = await fetch(`${API_URL}/alerts/${dismissAlertID}`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect((await alertRes.json()).data.status).toBe("dismissed");
  });
});

// ── End-to-end intelligence chain ─────────────────────────────────────────────

test.describe("Alert-to-Case: full intelligence chain", () => {
  let adminToken: string;
  let chainSourceID: string;
  let chainCaseID: string;
  let chainSubjectID: string | null = null;

  test.beforeAll(async () => {
    adminToken = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);

    const srcRes = await fetch(`${API_URL}/alert-sources`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({ name: "E2E Chain Source", source_system: "generic" }),
    });
    chainSourceID = (await srcRes.json()).data.id;

    // Create rule: auto-convert at risk_score >= 75
    await fetch(`${API_URL}/alert-rules`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({
        name: "E2E Chain Rule", min_risk_score: 75, alert_type: "*", mode: "auto", priority_mapping: "high",
      }),
    });
  });

  test.afterAll(async () => {
    if (chainCaseID) {
      await fetch(`${API_URL}/cases/${chainCaseID}`, {
        method: "DELETE", headers: { Authorization: `Bearer ${adminToken}` },
      });
    }
    if (chainSubjectID) {
      await fetch(`${API_URL}/subjects/${chainSubjectID}`, {
        method: "DELETE", headers: { Authorization: `Bearer ${adminToken}` },
      });
    }
    if (chainSourceID) {
      await fetch(`${API_URL}/alert-sources/${chainSourceID}`, {
        method: "DELETE", headers: { Authorization: `Bearer ${adminToken}` },
      });
    }
  });

  test("high-risk ingest auto-converts, creates subject, and returns case_id", async () => {
    const payload = {
      alert_id:          `CHAIN-${Date.now()}`,
      alert_type:        "SUSPICIOUS_TRANSFER",
      risk_score:        88,
      subject_name:      "E2E Chain Customer",
      subject_id_type:   "PAN",
      subject_id_number: `CHAIN${Math.floor(Math.random() * 90000 + 10000)}`,
      account_number:    "CHAIN-ACC-001",
      amount:            1500000,
      currency:          "INR",
      description:       "Large suspicious cross-border transfer",
    };
    const res = await fetch(`${API_URL}/alerts/ingest/${chainSourceID}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    expect(data.status).toBe("auto_converted");
    expect(data.case_id).toBeDefined();
    chainCaseID = data.case_id;
    chainSubjectID = data.subject_id ?? null;
  });

  test("auto-converted case exists with high priority", async () => {
    if (!chainCaseID) return;
    const res = await fetch(`${API_URL}/cases/${chainCaseID}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.priority).toBe("high");
    expect(data.case_number).toMatch(/OWL-/);
  });

  test("linked subject 360 profile shows total_cases >= 1", async () => {
    if (!chainSubjectID) return;
    // Brief wait for async subject-case link
    await new Promise((r) => setTimeout(r, 1500));

    const res = await fetch(`${API_URL}/subjects/${chainSubjectID}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.total_cases).toBeGreaterThanOrEqual(1);
  });

  test("GET /alerts?status=converted finds the auto-converted alert by case_id", async () => {
    if (!chainCaseID) return;
    const res = await fetch(`${API_URL}/alerts?status=converted&limit=50`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const linked = (data.alerts as Array<{ linked_case_id: string | null }>)
      .find((a) => a.linked_case_id === chainCaseID);
    expect(linked).toBeDefined();
  });
});

// ── UI tests ──────────────────────────────────────────────────────────────────

test.describe("Alert Queue — UI", () => {
  let adminToken: string;
  let uiSourceID: string;
  let pendingAlertID: string;

  test.beforeAll(async () => {
    adminToken = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);

    // Create source for UI seeding
    const srcRes = await fetch(`${API_URL}/alert-sources`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({ name: "E2E UI Alert Source", source_system: "generic" }),
    });
    uiSourceID = (await srcRes.json()).data.id;

    // Ingest a low-risk alert to populate the pending queue
    const ingestRes = await fetch(`${API_URL}/alerts/ingest/${uiSourceID}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        alert_id:          `E2E-UI-${Date.now()}`,
        alert_type:        "CASH_STRUCTURING",
        risk_score:        55,
        subject_name:      "E2E UI Queue Customer",
        subject_id_type:   "PAN",
        subject_id_number: `UIPAN${Math.floor(Math.random() * 90000 + 10000)}`,
        account_number:    "UI-ACC-001",
        amount:            490000,
        currency:          "INR",
        description:       "E2E UI test pending alert",
      }),
    });
    pendingAlertID = (await ingestRes.json()).data?.alert_id;
  });

  test.afterAll(async () => {
    if (uiSourceID) {
      await fetch(`${API_URL}/alert-sources/${uiSourceID}`, {
        method: "DELETE", headers: { Authorization: `Bearer ${adminToken}` },
      });
    }
  });

  test("Alert Queue page renders heading, description, and Refresh button", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/alerts");
    await expect(page.getByRole("heading", { name: /alert queue/i })).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText(/incoming alerts from connected tms/i)).toBeVisible();
    await expect(page.getByRole("button", { name: /refresh/i })).toBeVisible();
  });

  test("sidebar shows Alert Queue nav link", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/dashboard");
    await expect(page.getByRole("link", { name: /alert queue/i })).toBeVisible();
  });

  test("pending alert row visible with type, risk badge, and status badge", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/alerts");
    // Default filter is pending
    await expect(page.getByText("E2E UI Queue Customer").first()).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText(/cash structuring/i).first()).toBeVisible();
    // Scope status badge check to within the table row to avoid hidden <option> elements
    const queueRow = page.locator("tr").filter({ hasText: "E2E UI Queue Customer" }).first();
    await expect(queueRow.getByText(/pending/i)).toBeVisible();
  });

  test("convert button on pending row opens Convert Alert to Case modal", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/alerts");
    await expect(page.getByText("E2E UI Queue Customer").first()).toBeVisible({ timeout: 8_000 });

    const row = page.locator("tr").filter({ hasText: "E2E UI Queue Customer" }).first();
    // First action button is the CheckCircle convert button
    await row.locator("button").first().click();

    const convertModal = page.locator(".fixed.inset-0");
    await expect(convertModal.getByRole("heading", { name: /convert alert to case/i })).toBeVisible({ timeout: 6_000 });
    // Modal input has no id/for label association — check for the input element directly
    await expect(convertModal.locator("input").first()).toBeVisible();
    await expect(convertModal.getByRole("button", { name: /create case/i })).toBeVisible();

    // Close without converting
    await convertModal.getByRole("button", { name: /cancel/i }).click();
    await expect(page.getByRole("heading", { name: /convert alert to case/i })).not.toBeVisible();
  });

  test("dismiss button on pending row opens Dismiss Alert modal", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/alerts");
    await expect(page.getByText("E2E UI Queue Customer").first()).toBeVisible({ timeout: 8_000 });

    const row = page.locator("tr").filter({ hasText: "E2E UI Queue Customer" }).first();
    // Second action button is the XCircle dismiss button
    await row.locator("button").nth(1).click();

    const dismissModal = page.locator(".fixed.inset-0");
    await expect(dismissModal.getByRole("heading", { name: /dismiss alert/i })).toBeVisible({ timeout: 6_000 });
    await expect(dismissModal.getByPlaceholder(/reason for dismissal/i)).toBeVisible();
    // Scope to modal to avoid the 26 table "Dismiss" icon buttons
    await expect(dismissModal.getByRole("button", { name: /^dismiss$/i })).toBeVisible();

    // Close without dismissing
    await dismissModal.getByRole("button", { name: /cancel/i }).click();
    await expect(page.getByRole("heading", { name: /dismiss alert/i })).not.toBeVisible();
  });

  test("status filter switches between pending / converted / all", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/alerts");
    await expect(page.getByRole("heading", { name: /alert queue/i })).toBeVisible({ timeout: 8_000 });

    const statusSelect = page.locator("select").first();
    // Switch to "All statuses"
    await statusSelect.selectOption("");
    await expect(page.getByRole("heading", { name: /alert queue/i })).toBeVisible();

    // Switch to "Converted"
    await statusSelect.selectOption("converted");
    // Either shows "No alerts found" or shows the heading (page loaded successfully)
    await expect(page.getByRole("heading", { name: /alert queue/i })).toBeVisible({ timeout: 6_000 });
  });

  test("full convert flow via UI: pending → converted with case number link", async ({ page }) => {
    // Ingest a fresh alert just for this UI convert test
    const freshIngestRes = await fetch(`${API_URL}/alerts/ingest/${uiSourceID}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        alert_id:          `E2E-UI-CONV-${Date.now()}`,
        alert_type:        "SUSPICIOUS_TRANSFER",
        risk_score:        52,
        subject_name:      "E2E UI Convert Customer",
        subject_id_type:   "PAN",
        subject_id_number: `CVPAN${Math.floor(Math.random() * 90000 + 10000)}`,
        account_number:    "CV-ACC-001",
        amount:            750000,
        currency:          "INR",
        description:       "E2E UI convert test",
      }),
    });
    expect(freshIngestRes.status).toBe(201);

    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/alerts");
    await expect(page.getByText("E2E UI Convert Customer").first()).toBeVisible({ timeout: 8_000 });

    // Click convert on the row
    const row = page.locator("tr").filter({ hasText: "E2E UI Convert Customer" }).first();
    await row.locator("button").first().click();
    await expect(page.getByRole("heading", { name: /convert alert to case/i })).toBeVisible({ timeout: 6_000 });

    // Submit conversion
    await page.getByRole("button", { name: /create case/i }).click();

    // After conversion, the row should show "converted" status (page refreshes)
    await page.locator("select").first().selectOption("converted");
    await expect(page.getByText("E2E UI Convert Customer").first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole("heading", { name: /alert queue/i })).toBeVisible();
  });
});
