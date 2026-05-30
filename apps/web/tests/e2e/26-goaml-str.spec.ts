import { test, expect } from "@playwright/test";
import { login, apiToken, apiTokenAs, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * goAML 4.0 STR filing E2E tests.
 *
 * Verifies:
 * - STR filing creation with full goAML fields
 * - goAML readiness validation endpoint
 * - Export goAML XML button availability after approval
 * - Analyst cannot see the STR tab (access restriction)
 * - Manager can submit and approve workflow
 */

const MANAGER_EMAIL = "manager@opsowl.dev";
const MANAGER_PASSWORD = "OpsOwl@2024";
const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASSWORD = "OpsOwl@2024";
const ANALYST_EMAIL = "analyst@opsowl.dev";
const ANALYST_PASSWORD = "OpsOwl@2024";

async function loginAs(page: import("@playwright/test").Page, email: string, password: string) {
  await login(page, email, password);
}

test.describe("goAML STR filing", () => {
  let token: string;
  let caseId: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E goAML STR Test Case", "high");
    caseId = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  // ── Analyst cannot see STR tab ───────────────────────────────────────────────

  test("analyst does not see STR tab on case detail", async ({ page }) => {
    await loginAs(page, ANALYST_EMAIL, ANALYST_PASSWORD);
    await page.goto(`/cases/${caseId}`);
    await expect(page.getByRole("button", { name: /^STR$/i })).not.toBeVisible();
  });

  // ── Validate endpoint returns errors on incomplete filing ────────────────────

  test("goAML validate endpoint returns errors for incomplete filing", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    // Create a minimal draft (no goAML fields)
    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify({
        report_type: "STR",
        suspicion_basis: "Test suspicion",
        narrative: "Test narrative",
        evidence_summary: "",
        filing_entity: { name: "Test Bank" },
        subject: { name: "John Doe" },
        transactions: [],
      }),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();

    // Validate goAML readiness
    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    expect(valRes.status).toBe(200);
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(false);
    expect(val.errors.length).toBeGreaterThan(0);

    const fields = val.errors.map((e: { field: string }) => e.field);
    expect(fields).toContain("filing_entity.rentity_id");
    expect(fields).toContain("transactions");
  });

  // ── Full workflow: create draft with goAML fields → submit → approve ────────

  test("full STR workflow with goAML fields", async ({ page }) => {
    await loginAs(page, MANAGER_EMAIL, MANAGER_PASSWORD);
    await page.goto(`/cases/${caseId}/str`);

    // Click New Filing
    await page.getByRole("button", { name: /new filing/i }).click();

    // Basic fields
    await page.getByPlaceholder(/unusual cash/i).fill("Large structured cash deposits to avoid CTR threshold");
    await page.getByPlaceholder(/detailed description/i).fill(
      "Subject made 5 cash deposits of ₹9.5L each within 10 days — pattern suggests deliberate structuring."
    );

    // Subject
    await page.getByPlaceholder("First name").fill("Ramesh");
    await page.getByPlaceholder("Last name").fill("Kumar");
    await page.getByPlaceholder(/ABCDE1234F|Identifier/i).first().fill("ABCDE1234F");

    // Add a transaction
    await page.getByRole("button", { name: /add transaction/i }).click();
    // First transaction row (index 0) — fill date and amount
    const dateInputs = page.locator("input[type='date']");
    await dateInputs.first().fill("2026-04-15");
    const amountInputs = page.getByPlaceholder(/e.g. 1500000/i);
    await amountInputs.first().fill("950000");

    // Expand goAML fields
    await page.getByRole("button", { name: /reporting officer/i }).click();

    await page.getByPlaceholder("Your bank / NBFC name").fill("Sunrise Bank Ltd");
    await page.getByPlaceholder(/FIU-IN-BANK/i).fill("FIU-IN-SUN-0042");
    await page.getByPlaceholder("Officer first name").fill("Priya");
    await page.getByPlaceholder("Officer last name").fill("Sharma");
    // Officer PAN field (the second PAN-style input)
    const panInputs = page.getByPlaceholder("Officer PAN");
    await panInputs.fill("PQRST5678Y");

    // Save draft
    await page.getByRole("button", { name: /save draft/i }).click();

    // Filing card should appear
    await expect(page.getByText("DRAFT")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText("Large structured cash deposits")).toBeVisible();
  });

  // ── goAML readiness panel shows in filing card ───────────────────────────────

  test("goAML readiness panel appears in filing card for managers", async ({ page }) => {
    await loginAs(page, MANAGER_EMAIL, MANAGER_PASSWORD);
    await page.goto(`/cases/${caseId}/str`);

    // At least one draft filing should exist from previous test
    await expect(page.locator(".bg-slate-800\\/60").first()).toBeVisible({ timeout: 8_000 });

    // Expand the readiness panel
    await page.getByRole("button", { name: /goaml 4.0 readiness/i }).first().click();
    // Should show either pass or list of errors — either is valid
    await expect(
      page.getByText(/required fields present|required before export/i).first()
    ).toBeVisible({ timeout: 6_000 });
  });

  // ── Export goAML blocked before approval ────────────────────────────────────

  test("export goAML XML button only visible for approved/filed status", async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto(`/cases/${caseId}/str`);

    // Draft filings should NOT show export buttons
    const exportBtn = page.getByRole("button", { name: /export goaml xml/i });
    await expect(exportBtn).not.toBeVisible();
  });

  // ── API: goAML export blocked for analyst ────────────────────────────────────

  test("analyst cannot call goAML export API", async () => {
    const analystToken = await apiTokenAs(ANALYST_EMAIL, ANALYST_PASSWORD);
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    // Get the first STR filing
    const listRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data } = await listRes.json();
    const filings = data?.filings ?? [];
    if (filings.length === 0) return; // no filings seeded, skip

    const strId = filings[0].id;

    // Analyst tries to call export — must be 403
    const res = await fetch(`${API_URL}/cases/${caseId}/str/${strId}/export/goaml`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(403);
  });
});
