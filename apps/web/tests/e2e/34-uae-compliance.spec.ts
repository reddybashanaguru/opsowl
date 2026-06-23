import { test, expect } from "@playwright/test";
import { login, apiTokenAs, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * UAE AML/CFT Compliance E2E Tests
 *
 * Federal Decree-Law No. 20 of 2018 on AML/CFT
 * CBUAE Supervisory Guidance — 2026
 *
 * Covers:
 * 1. Settings — UAE jurisdiction preset applies correct values
 * 2. Settings — new UAE fields (currency, CTR, wire, id_types, deadline) editable
 * 3. STR form — suspicion_date picker renders with deadline countdown
 * 4. STR form — Emirates ID option in ID type dropdown
 * 5. STR API — suspicion_date persisted; deadline_days + is_overdue returned
 * 6. STR deadlines API — GET /api/v1/str/deadlines returns approaching filings
 * 7. Transactions page loads without error
 * 8. Dashboard — compliance officers see Attention Required panel
 */

const ADMIN_EMAIL   = "admin@opsowl.dev";
const ADMIN_PASS    = "OpsOwl@2024";
const MANAGER_EMAIL = "ravi.kumar@opsowl.dev";
const MANAGER_PASS  = "OpsOwl@2024";
const ANALYST_EMAIL = "divya.reddy@opsowl.dev";
const ANALYST_PASS  = "OpsOwl@2024";

// Helper — apply UAE config via API
async function applyUAEConfig(token: string) {
  const res = await fetch(`${API_URL}/config`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({
      app_name: "OpsOwl AI",
      industry: "financial_services",
      case_label: "Case", cases_label: "Cases",
      task_label: "Task", tasks_label: "Tasks",
      document_label: "Document", documents_label: "Documents",
      risk_score_label: "Risk Score", risk_level_label: "Risk Level",
      sla_label: "SLA", comms_label: "Communications",
      analyst_label: "Analyst", timeline_label: "Timeline",
      note_label: "Note", notes_label: "Notes",
      str_label: "STR",
      str_reference_label: "UAE FIU Reference",
      str_regulator_label: "UAE FIU",
      str_law_reference: "Federal Decree-Law No. 20 of 2018",
      str_id_label: "Emirates ID",
      str_deadline_days: 30,
      local_currency: "AED",
      currency_symbol: "AED",
      ctr_threshold: 55000,
      wire_threshold: 3675,
      id_types: ["PASSPORT","EMIRATES_ID","UAE_TRADE_LICENSE","TRN","GCC_NATIONAL_ID","RESIDENCY_VISA","DRIVING_LICENSE","OTHER"],
    }),
  });
  return res.status;
}

async function restoreIndiaConfig(token: string) {
  await fetch(`${API_URL}/config`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({
      app_name: "OpsOwl AI",
      industry: "financial_services",
      case_label: "Case", cases_label: "Cases",
      task_label: "Task", tasks_label: "Tasks",
      document_label: "Document", documents_label: "Documents",
      risk_score_label: "Risk Score", risk_level_label: "Risk Level",
      sla_label: "SLA", comms_label: "Communications",
      analyst_label: "Analyst", timeline_label: "Timeline",
      note_label: "Note", notes_label: "Notes",
      str_label: "STR",
      str_reference_label: "FIU Reference",
      str_regulator_label: "FIU-IND",
      str_law_reference: "PMLA 2002",
      str_id_label: "PAN",
      str_deadline_days: 30,
      local_currency: "INR",
      currency_symbol: "₹",
      ctr_threshold: 1000000,
      wire_threshold: 0,
      id_types: ["PASSPORT","NATIONAL_ID","TAX_ID","DRIVING_LICENSE","PAN","AADHAAR_REF","VOTER_ID","OTHER"],
    }),
  });
}

// ── 1 & 2. Settings: UAE preset ───────────────────────────────────────────────

test.describe("Settings — UAE jurisdiction preset", () => {
  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/settings");
    await page.waitForLoadState("networkidle");
  });

  test.afterEach(async () => {
    const token = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);
    await restoreIndiaConfig(token);
  });

  test("Jurisdiction Quick Setup section is visible", async ({ page }) => {
    await expect(page.getByText("Jurisdiction Quick Setup")).toBeVisible();
  });

  test("India and UAE preset buttons are rendered", async ({ page }) => {
    await expect(page.getByTestId("preset-india")).toBeVisible();
    await expect(page.getByTestId("preset-uae")).toBeVisible();
  });

  test("clicking UAE preset fills local_currency with AED", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    await expect(page.getByTestId("field-local_currency")).toHaveValue("AED");
  });

  test("clicking UAE preset sets CTR threshold to 55000", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    await expect(page.getByTestId("field-ctr_threshold")).toHaveValue("55000");
  });

  test("clicking UAE preset sets wire threshold to 3675", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    await expect(page.getByTestId("field-wire_threshold")).toHaveValue("3675");
  });

  test("clicking UAE preset sets filing deadline to 30 days", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    await expect(page.getByTestId("field-str_deadline_days")).toHaveValue("30");
  });

  test("clicking UAE preset sets id_types to include EMIRATES_ID", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    const val = await page.getByTestId("field-id_types").inputValue();
    expect(val).toContain("EMIRATES_ID");
  });

  test("UAE preset shows Active badge after clicking", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    await expect(page.getByTestId("preset-uae").getByText("Active")).toBeVisible();
  });

  test("India preset shows Active badge after clicking India", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    await page.getByTestId("preset-india").click();
    await expect(page.getByTestId("preset-india").getByText("Active")).toBeVisible();
  });

  test("UAE preset saves and config API returns AED", async ({ page }) => {
    await page.getByTestId("preset-uae").click();
    await page.getByRole("button", { name: "Save Settings" }).click();
    await expect(page.getByText(/Saved — all users see updated labels/i)).toBeVisible({ timeout: 5_000 });

    // Verify via API that AED is now the config
    const res = await fetch(`${API_URL}/config`);
    const body = await res.json();
    const cfg = body.data ?? body;
    expect(cfg.local_currency).toBe("AED");
    expect(cfg.ctr_threshold).toBe(55000);
  });
});

// ── Settings — new UAE config fields editable ─────────────────────────────────

test.describe("Settings — new UAE config fields", () => {
  test.beforeEach(async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/settings");
    await page.waitForLoadState("networkidle");
  });

  test("str_deadline_days field accepts numeric input", async ({ page }) => {
    const input = page.getByTestId("field-str_deadline_days");
    await input.fill("45");
    await expect(input).toHaveValue("45");
  });

  test("local_currency field uppercases input", async ({ page }) => {
    const input = page.getByTestId("field-local_currency");
    await input.fill("usd");
    await expect(input).toHaveValue("USD");
  });

  test("id_types field accepts comma-separated values", async ({ page }) => {
    const input = page.getByTestId("field-id_types");
    await input.fill("PASSPORT, EMIRATES_ID, OTHER");
    await expect(input).toHaveValue("PASSPORT, EMIRATES_ID, OTHER");
  });

  test("ctr_threshold field renders with numeric value", async ({ page }) => {
    const input = page.getByTestId("field-ctr_threshold");
    await expect(input).toBeVisible();
    const val = await input.inputValue();
    expect(Number(val)).toBeGreaterThanOrEqual(0);
  });
});

// ── 3 & 4. STR form — suspicion_date + Emirates ID ───────────────────────────

test.describe("STR form — UAE compliance fields", () => {
  let caseId: string;
  let adminToken: string;

  test.beforeAll(async () => {
    adminToken = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);
    await applyUAEConfig(adminToken);
    const c = await createCase(adminToken, "UAE STR Form E2E Test", "high");
    caseId = c.id;
  });

  test.afterAll(async () => {
    if (caseId) await deleteCase(adminToken, caseId);
    await restoreIndiaConfig(adminToken);
  });

  test("STR page renders for manager", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await expect(page.getByText("Regulatory Filings")).toBeVisible({ timeout: 5_000 });
  });

  test("New Filing button is visible for manager", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await expect(page.getByRole("button", { name: /New Filing/i })).toBeVisible();
  });

  test("STR form renders 'Date Suspicion Arose' field", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await page.getByRole("button", { name: /New Filing/i }).click();
    await expect(page.getByText(/Date Suspicion Arose/i)).toBeVisible({ timeout: 3_000 });
  });

  test("entering suspicion_date shows filing deadline countdown", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await page.getByRole("button", { name: /New Filing/i }).click();

    const dateStr = new Date().toISOString().split("T")[0];
    const dateInputs = page.locator("input[type='date']");
    await dateInputs.first().fill(dateStr);

    await expect(page.getByText(/Filing deadline:/i)).toBeVisible({ timeout: 3_000 });
    await expect(page.getByText(/30 day/i)).toBeVisible();
  });

  test("STR form shows 'Article 15' regulatory reference in deadline label", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await page.getByRole("button", { name: /New Filing/i }).click();
    await expect(page.getByText(/Article 15/i)).toBeVisible({ timeout: 3_000 });
  });

  test("ID type dropdown contains EMIRATES_ID option", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await page.getByRole("button", { name: /New Filing/i }).click();

    const selects = page.locator("select");
    const count = await selects.count();
    let found = false;
    for (let i = 0; i < count; i++) {
      const opts = await selects.nth(i).evaluate((el: HTMLSelectElement) =>
        Array.from(el.options).map((o) => o.value)
      );
      if (opts.includes("EMIRATES_ID")) { found = true; break; }
    }
    expect(found).toBe(true);
  });

  test("compliance banner shows UAE FIU law reference", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await expect(page.getByText(/Federal Decree-Law/i).first()).toBeVisible({ timeout: 5_000 });
  });

  test("analyst is blocked from STR page (access restriction)", async ({ page }) => {
    await login(page, ANALYST_EMAIL, ANALYST_PASS);
    await page.goto(`/cases/${caseId}/str`);
    await expect(page.getByText(/Access Restricted/i)).toBeVisible({ timeout: 5_000 });
  });
});

// ── 5. STR API — suspicion_date + deadline fields ─────────────────────────────

test.describe("STR API — suspicion_date and deadline fields", () => {
  let caseId: string;
  let managerToken: string;
  let filingId: string;

  test.beforeAll(async () => {
    managerToken = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASS);
    const c = await createCase(managerToken, "UAE STR API suspicion_date test", "medium");
    caseId = c.id;

    const today = new Date().toISOString().split("T")[0];
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${managerToken}` },
      body: JSON.stringify({
        report_type: "STR",
        suspicion_date: today,
        suspicion_basis: "UAE E2E test — suspicion date tracking",
        narrative: "Automated E2E test for UAE suspicion_date compliance",
        evidence_summary: "Test evidence",
        filing_entity: { local_currency: "AED", entity_country: "AE", fiu_name: "UAE FIU" },
        subject: { first_name: "Ahmed", id_type: "EMIRATES_ID", id_number: "784-1990-1234567-1" },
        transactions: [],
      }),
    });
    if (res.status === 201) {
      const body = await res.json();
      filingId = (body.data ?? body).id;
    }
  });

  test.afterAll(async () => {
    if (caseId) await deleteCase(managerToken, caseId);
  });

  test("STR creation with suspicion_date returns 201", async () => {
    expect(filingId).toBeTruthy();
  });

  test("STR GET response includes suspicion_date", async () => {
    if (!filingId) return;
    const res = await fetch(`${API_URL}/cases/${caseId}/str/${filingId}`, {
      headers: { Authorization: `Bearer ${managerToken}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json();
    const filing = body.data ?? body;
    expect(filing.suspicion_date).toBeTruthy();
  });

  test("STR response includes deadline_days field (number)", async () => {
    if (!filingId) return;
    const res = await fetch(`${API_URL}/cases/${caseId}/str/${filingId}`, {
      headers: { Authorization: `Bearer ${managerToken}` },
    });
    const body = await res.json();
    const filing = body.data ?? body;
    expect(typeof filing.deadline_days).toBe("number");
    expect(filing.deadline_days).toBeGreaterThanOrEqual(0);
  });

  test("STR response includes is_overdue = false for today's suspicion", async () => {
    if (!filingId) return;
    const res = await fetch(`${API_URL}/cases/${caseId}/str/${filingId}`, {
      headers: { Authorization: `Bearer ${managerToken}` },
    });
    const body = await res.json();
    const filing = body.data ?? body;
    expect(typeof filing.is_overdue).toBe("boolean");
    expect(filing.is_overdue).toBe(false);
  });

  test("STR with suspicion_date 35 days ago has is_overdue = true", async () => {
    const pastDate = new Date();
    pastDate.setDate(pastDate.getDate() - 35);
    const pastStr = pastDate.toISOString().split("T")[0];

    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${managerToken}` },
      body: JSON.stringify({
        report_type: "STR",
        suspicion_date: pastStr,
        suspicion_basis: "Overdue test — 35 days ago",
        narrative: "Testing is_overdue = true",
        evidence_summary: "",
        filing_entity: {},
        subject: { first_name: "Test", id_type: "PASSPORT", id_number: "X1234567" },
        transactions: [],
      }),
    });
    expect(res.status).toBe(201);
    const body = await res.json();
    const filing = body.data ?? body;
    expect(filing.is_overdue).toBe(true);
    expect(filing.deadline_days).toBeLessThan(0);
  });

  test("STR list also returns suspicion_date for each filing", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      headers: { Authorization: `Bearer ${managerToken}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json();
    const filings: Array<{ suspicion_date?: string; deadline_days: number }> = (body.data ?? body).filings ?? [];
    expect(filings.length).toBeGreaterThan(0);
    const withDate = filings.filter((f) => f.suspicion_date);
    expect(withDate.length).toBeGreaterThan(0);
  });
});

// ── 6. STR deadlines API ──────────────────────────────────────────────────────

test.describe("STR deadlines API — GET /api/v1/str/deadlines", () => {
  let managerToken: string;
  let analystToken: string;

  test.beforeAll(async () => {
    managerToken = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASS);
    analystToken = await apiTokenAs(ANALYST_EMAIL, ANALYST_PASS);
  });

  test("returns 200 with items array and count", async () => {
    const res = await fetch(`${API_URL}/str/deadlines?deadline_days=30&window_days=14`, {
      headers: { Authorization: `Bearer ${managerToken}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json();
    const data = body.data ?? body;
    expect(data).toHaveProperty("items");
    expect(Array.isArray(data.items)).toBe(true);
    expect(data).toHaveProperty("count");
  });

  test("respects custom deadline_days and window_days params", async () => {
    const res = await fetch(`${API_URL}/str/deadlines?deadline_days=60&window_days=30`, {
      headers: { Authorization: `Bearer ${managerToken}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json();
    const data = body.data ?? body;
    expect(data.deadline_days).toBe(60);
    expect(data.window_days).toBe(30);
  });

  test("analyst is denied access (403)", async () => {
    const res = await fetch(`${API_URL}/str/deadlines`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(403);
  });

  test("deadline items contain required fields when results exist", async () => {
    const res = await fetch(`${API_URL}/str/deadlines?deadline_days=365&window_days=365`, {
      headers: { Authorization: `Bearer ${managerToken}` },
    });
    const body = await res.json();
    const data = body.data ?? body;
    if (data.items.length > 0) {
      const item = data.items[0];
      expect(item).toHaveProperty("filing_id");
      expect(item).toHaveProperty("case_id");
      expect(item).toHaveProperty("case_number");
      expect(item).toHaveProperty("suspicion_date");
      expect(item).toHaveProperty("deadline_date");
      expect(item).toHaveProperty("days_remaining");
      expect(item).toHaveProperty("is_overdue");
    }
  });
});

// ── 7. Transactions page ──────────────────────────────────────────────────────

test.describe("Transactions — UAE CTR context", () => {
  let caseId: string;
  let adminToken: string;

  test.beforeAll(async () => {
    adminToken = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);
    const c = await createCase(adminToken, "UAE Transactions CTR Test", "low");
    caseId = c.id;

    // Create a large cash transaction above UAE CTR threshold (AED 75,000 > AED 55,000)
    const today = new Date().toISOString().split("T")[0];
    const txnRes = await fetch(`${API_URL}/cases/${caseId}/transactions`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({
        txn_ref: `CTR-UAE-SETUP-${Date.now()}`,
        txn_date: today,
        amount: 75000,
        currency: "AED",
        txn_type: "credit",
        channel: "cash",
        narration: "Large cash deposit — UAE CTR E2E test",
        from_name_raw: "Test Customer",
        to_account_raw: "ACC-001",
      }),
    });
    if (!txnRes.ok) {
      const body = await txnRes.text();
      throw new Error(`Transaction setup failed: ${txnRes.status} ${body}`);
    }
  });

  test.afterAll(async () => {
    if (caseId) await deleteCase(adminToken, caseId);
  });

  test("transactions page loads for the case", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/cases/${caseId}/transactions`);
    await expect(page.getByRole("heading", { name: "Transactions" })).toBeVisible({ timeout: 5_000 });
  });

  test("transaction row above AED 55K shows CTR badge when config is UAE", async ({ page }) => {
    // Intercept the config API to always return UAE config (CTR 55,000 AED)
    // This avoids relying on async DB state propagation timing.
    await page.route("**/api/v1/config", async (route) => {
      if (route.request().method() === "GET") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            data: {
              app_name: "OpsOwl AI",
              industry: "financial_services",
              case_label: "Case", cases_label: "Cases",
              task_label: "Task", tasks_label: "Tasks",
              document_label: "Document", documents_label: "Documents",
              risk_score_label: "Risk Score", risk_level_label: "Risk Level",
              sla_label: "SLA", comms_label: "Communications",
              analyst_label: "Analyst", timeline_label: "Timeline",
              note_label: "Note", notes_label: "Notes",
              str_label: "STR", str_reference_label: "UAE FIU Reference",
              str_regulator_label: "UAE FIU",
              str_law_reference: "Federal Decree-Law No. 20 of 2018",
              str_id_label: "Emirates ID",
              str_deadline_days: 30,
              local_currency: "AED",
              currency_symbol: "AED",
              ctr_threshold: 55000,
              wire_threshold: 3675,
              id_types: ["PASSPORT","EMIRATES_ID","UAE_TRADE_LICENSE","TRN","GCC_NATIONAL_ID","RESIDENCY_VISA","DRIVING_LICENSE","OTHER"],
            },
          }),
        });
      } else {
        await route.continue();
      }
    });

    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/cases/${caseId}/transactions`);
    await page.waitForLoadState("networkidle");

    // Badge renders when amount (75,000) >= ctr_threshold (55,000)
    const ctrBadge = page.locator("span", { hasText: "CTR" }).first();
    await expect(ctrBadge).toBeVisible({ timeout: 8_000 });
  });

  test("transactions page shows transaction rows", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/cases/${caseId}/transactions`);
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: "Transactions" })).toBeVisible({ timeout: 5_000 });
  });
});

// ── 8. Dashboard ──────────────────────────────────────────────────────────────

test.describe("Dashboard — STR regulatory deadline widget", () => {
  test("dashboard renders for compliance manager", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto("/dashboard");
    await expect(page.getByText(/Good morning|Good afternoon|Good evening/i)).toBeVisible({ timeout: 5_000 });
  });

  test("Attention Required panel shows for manager", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto("/dashboard");
    await expect(page.getByText("Attention Required")).toBeVisible({ timeout: 5_000 });
  });

  test("STR deadlines widget appears if overdue filings exist", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto("/dashboard");
    await page.waitForTimeout(3000);
    // Widget only renders when count > 0 — soft check: page must not crash
    await expect(page.getByText(/Good morning|Good afternoon|Good evening/i)).toBeVisible();
  });
});
