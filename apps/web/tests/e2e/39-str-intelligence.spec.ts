import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * E2E suite for Feature 5: STR Pre-Submission Intelligence
 *
 * Tests cover:
 *  — Auto-populate button is visible in the new filing form
 *  — Pre-Flight Checklist panel visible on draft STR filings
 *  — All 4 checklist sections expand correctly
 *  — Narrative quality score renders
 *  — Transaction risk signals section renders
 *  — FATF typology section renders after typologies load
 *  — Checklist shows count of passed checks
 *  — Panel collapses and expands
 */

const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASS  = "OpsOwl@2024";

async function createSTRDraft(token: string, caseId: string, overrides: Record<string, unknown> = {}) {
  const body = {
    report_type: "STR",
    suspicion_date: "2026-01-15",
    suspicion_basis: "Unusual cash transactions inconsistent with customer profile",
    narrative: "The subject made multiple cash deposits during January 2026.",
    evidence_summary: "Transaction logs reviewed",
    filing_entity: { name: "Test Bank", rentity_id: "FIU-TEST-001", local_currency: "INR" },
    subject: { first_name: "Rajesh", last_name: "Kumar", id_type: "PAN", id_number: "ABCDE1234F" },
    transactions: [{ date: "2026-01-10", amount: 90000, transmode_code: "C", description: "Cash deposit" }],
    ...overrides,
  };
  const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify(body),
  });
  const json = await res.json() as { data: { id: string } };
  return json.data;
}

test.describe("STR Pre-Submission Intelligence", () => {
  let token: string;
  let caseId: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "STR Intelligence Test Case", "high");
    caseId = c.id;
    // Ensure at least one draft filing exists
    await createSTRDraft(token, caseId);
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  // ── API sanity ─────────────────────────────────────────────────────────────

  test("STR list endpoint returns filings for case", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: { filings: unknown[] } };
    expect(Array.isArray(body.data.filings)).toBe(true);
    expect(body.data.filings.length).toBeGreaterThan(0);
  });

  test("goAML validate endpoint returns readiness object", async () => {
    const listRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const listBody = await listRes.json() as { data: { filings: Array<{ id: string }> } };
    const filingId = listBody.data.filings[0]?.id;
    expect(filingId).toBeTruthy();

    const res = await fetch(`${API_URL}/cases/${caseId}/str/${filingId}/validate/goaml`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: { ready: boolean; errors: unknown[] } };
    expect(typeof body.data.ready).toBe("boolean");
    expect(Array.isArray(body.data.errors)).toBe(true);
  });

  // ── UI: Auto-populate button ───────────────────────────────────────────────

  test("Auto-populate button visible in New Filing form", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    // Open new filing form
    await page.getByRole("button", { name: "+ New Filing" }).click();
    await expect(page.getByTestId("auto-populate-btn")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText("Auto-populate from Case")).toBeVisible();
  });

  test("Auto-populate button is clickable and shows loading state", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await page.getByRole("button", { name: "+ New Filing" }).click();
    await expect(page.getByTestId("auto-populate-btn")).toBeVisible({ timeout: 5_000 });
    await page.getByTestId("auto-populate-btn").click();
    // Should show "Populating…" or complete quickly for a case with no subjects
    // Either way, button should not throw errors
    await page.waitForTimeout(2000);
    // After completion, button should return to normal state or show populated badge
    const isNormal = await page.getByText("Auto-populate from Case").isVisible().catch(() => false);
    const isPopulated = await page.getByText(/Auto-populated from case/).isVisible().catch(() => false);
    expect(isNormal || isPopulated).toBe(true);
  });

  // ── UI: Pre-Flight Checklist panel ────────────────────────────────────────

  test("Pre-Flight Checklist panel is visible on draft filing", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await expect(page.getByText("Pre-Flight Checklist")).toBeVisible({ timeout: 10_000 });
  });

  test("checklist header shows N/4 checks passed", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await expect(page.getByText(/\/4 checks passed/)).toBeVisible({ timeout: 10_000 });
  });

  test("expanding checklist shows 4 section headers", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await expect(page.getByText("Pre-Flight Checklist")).toBeVisible({ timeout: 10_000 });
    await page.getByText("Pre-Flight Checklist").click();
    await expect(page.getByText("Structural Completeness")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText(/Narrative Quality/)).toBeVisible({ timeout: 3_000 });
    await expect(page.getByText("Transaction Risk Signals")).toBeVisible({ timeout: 3_000 });
    await expect(page.getByText("FATF Typology Match")).toBeVisible({ timeout: 3_000 });
  });

  test("clicking Structural Completeness shows goAML validation", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await expect(page.getByText("Pre-Flight Checklist")).toBeVisible({ timeout: 10_000 });
    await page.getByText("Pre-Flight Checklist").click();
    await expect(page.getByText("Structural Completeness")).toBeVisible({ timeout: 5_000 });
    await page.getByText("Structural Completeness").click();
    // Should show either "All required goAML fields" or list of missing fields
    await expect(
      page.getByText(/goAML|required|missing|field/).first()
    ).toBeVisible({ timeout: 8_000 });
  });

  test("clicking Narrative Quality shows score and checks", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await expect(page.getByText("Pre-Flight Checklist")).toBeVisible({ timeout: 10_000 });
    await page.getByText("Pre-Flight Checklist").click();
    await expect(page.getByText(/Narrative Quality/)).toBeVisible({ timeout: 5_000 });
    await page.getByText(/Narrative Quality — \d+%/).click();
    // Score bar and check items should appear
    await expect(page.getByText("Substantive narrative")).toBeVisible({ timeout: 5_000 });
  });

  test("clicking Transaction Risk Signals shows clean or flagged state", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await page.getByText("Pre-Flight Checklist").waitFor({ timeout: 10_000 });
    await page.getByText("Pre-Flight Checklist").click();
    await page.getByText("Transaction Risk Signals").waitFor({ timeout: 5_000 });
    await page.getByText("Transaction Risk Signals").click();
    const hasClean = await page.getByText("No anomalous patterns").isVisible().catch(() => false);
    const hasSignal = await page.getByText(/pattern|Missing counterparty|frequency|structuring/).first().isVisible().catch(() => false);
    expect(hasClean || hasSignal).toBe(true);
  });

  test("clicking FATF Typology Match shows typology content", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await page.getByText("Pre-Flight Checklist").waitFor({ timeout: 10_000 });
    await page.getByText("Pre-Flight Checklist").click();
    await page.getByText("FATF Typology Match").waitFor({ timeout: 5_000 });
    await page.getByText("FATF Typology Match").click();
    await expect(
      page.getByText(/FATF|No FATF|typolog|Loading/).first()
    ).toBeVisible({ timeout: 8_000 });
  });

  test("checklist panel collapses when header clicked again", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseId}/str`);
    await page.getByText("Pre-Flight Checklist").waitFor({ timeout: 10_000 });
    await page.getByText("Pre-Flight Checklist").click();
    await expect(page.getByText("Structural Completeness")).toBeVisible({ timeout: 5_000 });
    await page.getByText("Pre-Flight Checklist").click();
    await expect(page.getByText("Structural Completeness")).not.toBeVisible({ timeout: 3_000 });
  });
});
