import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * QA suite for Feature 3: FATF Typology Library + Case Intelligence
 *
 * Tests cover:
 *  — Library page: loads, search, category filter, card expand
 *  — Insights page: typology section visible, tag modal open/close
 *  — API: tag typology to a case, list case typologies, untag
 */

const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASS  = "OpsOwl@2024";

test.describe("FATF Typology Library", () => {
  let token: string;
  let testCase: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E Typology Test Case");
    testCase = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, testCase);
  });

  // ── Library page ───────────────────────────────────────────────────────────

  test("FATF Typologies nav link is visible in admin sidebar", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    await expect(page.getByRole("link", { name: "FATF Typologies" })).toBeVisible();
  });

  test("typology library page loads and shows typologies", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    await expect(page.getByText("FATF Typology Library")).toBeVisible();
    // Wait for at least one typology name to appear (data loaded from API)
    await expect(page.getByText("Smurfing / Structuring")).toBeVisible({ timeout: 10_000 });
  });

  test("library shows total typology count", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    // The header shows "25 curated ML/TF typologies"
    await expect(page.getByText(/curated ML\/TF typologies/)).toBeVisible({ timeout: 8_000 });
  });

  test("category filter pills are visible", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    await expect(page.getByText("Smurfing / Structuring")).toBeVisible({ timeout: 10_000 });
    // Category pill buttons include count in parentheses — e.g. "Layering (8)"
    await expect(page.getByRole("button", { name: /^Layering \(/ })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Placement \(/ })).toBeVisible();
  });

  test("search filters typologies by name", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    // Wait for data to load
    await expect(page.getByText("Commingling of Funds")).toBeVisible({ timeout: 10_000 });
    await page.getByPlaceholder("Search typologies by name, description, or code…").fill("Smurfing");
    await expect(page.getByText("Smurfing / Structuring")).toBeVisible({ timeout: 3_000 });
    // Other typologies should no longer show
    await expect(page.getByText("Commingling of Funds")).not.toBeVisible({ timeout: 3_000 });
  });

  test("category filter narrows results", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    await expect(page.getByText("Trade-Based Money Laundering")).toBeVisible({ timeout: 10_000 });
    // Use the select dropdown to filter by TF category
    await page.getByRole("combobox").selectOption("TF");
    // TBML is LAYERING — should not show when filtering TF
    await expect(page.getByText("Trade-Based Money Laundering")).not.toBeVisible({ timeout: 3_000 });
    // "Self-Funded / Retail Terrorism" only appears as a card name in TF category
    await expect(page.getByText("Self-Funded / Retail Terrorism")).toBeVisible({ timeout: 3_000 });
  });

  test("expanding a typology card shows Red Flags", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    await page.getByPlaceholder("Search typologies by name, description, or code…").fill("Hawala");
    const card = page.getByText("Hawala / Informal Value Transfer").locator("..").locator("..");
    await page.getByText("Hawala / Informal Value Transfer").click();
    await expect(page.getByText("Red Flags")).toBeVisible({ timeout: 3_000 });
  });

  test("expanding a typology card shows Risk Keywords", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    await expect(page.getByText("Hawala / Informal Value Transfer")).toBeVisible({ timeout: 10_000 });
    await page.getByPlaceholder("Search typologies by name, description, or code…").fill("Hawala");
    await expect(page.getByText("Hawala / Informal Value Transfer")).toBeVisible({ timeout: 5_000 });
    await page.getByText("Hawala / Informal Value Transfer").click();
    await expect(page.getByText("Risk Keywords")).toBeVisible({ timeout: 3_000 });
    // "hawala" appears in both the card name area and as a keyword chip — use first()
    await expect(page.getByText("hawala").first()).toBeVisible();
  });

  test("expanding a typology shows FATF Reference", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, "/settings/typologies");
    await page.getByPlaceholder("Search typologies by name, description, or code…").fill("Crypto");
    await page.getByText("Virtual Asset / Cryptocurrency Abuse").click();
    await expect(page.getByText("FATF Reference")).toBeVisible({ timeout: 3_000 });
  });

  // ── API tests ──────────────────────────────────────────────────────────────

  test("GET /api/v1/typologies returns 25 typologies", async () => {
    const res = await fetch(`${API_URL}/typologies`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: unknown[] };
    const data = body.data;
    expect(Array.isArray(data)).toBe(true);
    expect(data.length).toBeGreaterThanOrEqual(25);
  });

  test("GET /api/v1/typologies?category=LAYERING returns only LAYERING typologies", async () => {
    const res = await fetch(`${API_URL}/typologies?category=LAYERING`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: Array<{ category: string }> };
    const data = body.data;
    expect(data.every(t => t.category === "LAYERING")).toBe(true);
    expect(data.length).toBeGreaterThanOrEqual(1);
  });

  test("GET /api/v1/typologies?q=smurfing returns matching typology", async () => {
    const res = await fetch(`${API_URL}/typologies?q=smurfing`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: Array<{ code: string }> };
    expect(body.data.some(t => t.code === "SMURFING")).toBe(true);
  });

  test("GET /api/v1/cases/:id/typologies returns empty array initially", async () => {
    const res = await fetch(`${API_URL}/cases/${testCase}/typologies`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: unknown[] };
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.data.length).toBe(0);
  });

  test("POST /api/v1/cases/:id/typologies tags a typology to the case", async () => {
    const libRes = await fetch(`${API_URL}/typologies?q=trade-based`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const libBody = await libRes.json() as { data: Array<{ id: string; code: string }> };
    const tbml = libBody.data.find(t => t.code === "TBML");
    expect(tbml).toBeDefined();

    const tagRes = await fetch(`${API_URL}/cases/${testCase}/typologies`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ typology_id: tbml!.id, notes: "E2E test tag", is_ai_suggested: false }),
    });
    expect(tagRes.status).toBe(201);
  });

  test("GET /api/v1/cases/:id/typologies lists tagged typology", async () => {
    const libRes = await fetch(`${API_URL}/typologies?q=smurfing`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const libBody = await libRes.json() as { data: Array<{ id: string; code: string }> };
    const smurfing = libBody.data.find(t => t.code === "SMURFING");
    if (smurfing) {
      await fetch(`${API_URL}/cases/${testCase}/typologies`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ typology_id: smurfing.id, notes: "", is_ai_suggested: true }),
      });
    }

    const res = await fetch(`${API_URL}/cases/${testCase}/typologies`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: Array<{ code: string }> };
    expect(body.data.length).toBeGreaterThanOrEqual(1);
  });

  test("DELETE /api/v1/cases/:id/typologies/:typologyId untags a typology", async () => {
    const listRes = await fetch(`${API_URL}/cases/${testCase}/typologies`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const listBody = await listRes.json() as { data: Array<{ id: string }> };
    const tagged = listBody.data;
    if (tagged.length === 0) {
      return;
    }
    const typologyId = tagged[0].id;

    const delRes = await fetch(`${API_URL}/cases/${testCase}/typologies/${typologyId}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(delRes.status).toBe(204);
  });

  // ── Insights page UI ───────────────────────────────────────────────────────

  test("insights page shows Matched FATF Typologies section", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${testCase}/insights`);
    await expect(page.getByText("Matched FATF Typologies")).toBeVisible({ timeout: 8_000 });
  });

  test("Tag Typology button opens the modal", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${testCase}/insights`);
    await page.getByText("Tag Typology").click();
    await expect(page.getByText("Tag FATF Typology")).toBeVisible({ timeout: 5_000 });
  });

  test("modal search filters typologies", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${testCase}/insights`);
    await page.getByText("Tag Typology").click();
    await page.getByPlaceholder("Search typologies…").fill("Casino");
    await expect(page.getByText("Casino / Gaming Layering")).toBeVisible({ timeout: 3_000 });
  });

  test("modal category filter works", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${testCase}/insights`);
    await page.getByText("Tag Typology").click();
    await page.locator("select").selectOption("TF");
    await expect(page.getByText("Terrorist Financing")).toBeVisible({ timeout: 3_000 });
  });

  test("clicking modal overlay closes it", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${testCase}/insights`);
    await page.getByText("Tag Typology").click();
    await expect(page.getByText("Tag FATF Typology")).toBeVisible({ timeout: 5_000 });
    // Click overlay (outside the modal box)
    await page.locator(".fixed.inset-0").click({ position: { x: 10, y: 10 } });
    await expect(page.getByText("Tag FATF Typology")).not.toBeVisible({ timeout: 3_000 });
  });

  test("closing modal with × button works", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${testCase}/insights`);
    await page.getByText("Tag Typology").click();
    await expect(page.getByText("Tag FATF Typology")).toBeVisible({ timeout: 5_000 });
    await page.getByRole("button", { name: "×" }).click();
    await expect(page.getByText("Tag FATF Typology")).not.toBeVisible({ timeout: 3_000 });
  });
});
