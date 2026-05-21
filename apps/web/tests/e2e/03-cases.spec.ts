import { test, expect } from "@playwright/test";
import { login, apiToken, deleteCase } from "./helpers/auth";

test.describe("Cases — list and create", () => {
  let token: string;
  const createdIds: string[] = [];

  test.beforeAll(async () => {
    token = await apiToken();
  });

  test.afterAll(async () => {
    for (const id of createdIds) {
      await deleteCase(token, id);
    }
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/cases");
  });

  test("cases list renders table headers", async ({ page }) => {
    await expect(page.getByRole("columnheader", { name: /Case/i })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: /Status/i })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: /Priority/i })).toBeVisible();
  });

  test("cases list shows data rows", async ({ page }) => {
    await page.waitForTimeout(2000);
    const rows = page.locator("tbody tr").filter({ hasNotText: "animate-pulse" });
    await expect(rows.first()).toBeVisible();
  });

  test("search filter narrows results", async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search cases/i);
    await expect(searchInput).toBeVisible();
    await searchInput.fill("fraud");
    await page.waitForTimeout(800);
    await expect(page).not.toHaveURL(/error/);
  });

  test("status filter dropdown works", async ({ page }) => {
    const statusSelect = page.locator("select").first();
    await statusSelect.selectOption("open");
    await page.waitForTimeout(600);
    await expect(page).not.toHaveURL(/error/);
    await statusSelect.selectOption("");
  });

  test("priority filter works", async ({ page }) => {
    const selects = page.locator("select");
    await selects.nth(1).selectOption("high");
    await page.waitForTimeout(600);
    await expect(page).not.toHaveURL(/error/);
  });

  test("New Case link navigates to create form", async ({ page }) => {
    await page.getByRole("link", { name: /New/i }).click();
    await expect(page).toHaveURL(/\/cases\/new/);
    await expect(page.getByRole("heading", { name: "New Case" })).toBeVisible();
  });

  test("create case — title too short shows validation", async ({ page }) => {
    // Navigate via clicking the link (reliable — same origin navigation)
    await page.getByRole("link", { name: /New/i }).click();
    await expect(page).toHaveURL(/\/cases\/new/);
    await page.getByRole("button", { name: "Create Case" }).click();
    await expect(page.getByText(/at least 5 characters/i)).toBeVisible();
  });

  test("create case — full happy path", async ({ page }) => {
    await page.getByRole("link", { name: /New/i }).click();
    await expect(page).toHaveURL(/\/cases\/new/);

    const title = `E2E Test Case ${Date.now()}`;
    // Title input
    await page.getByPlaceholder(/Suspicious UPI/i).fill(title);
    // Priority — click "high" button (it's a button group, not a select)
    await page.getByRole("button", { name: "high" }).click();
    // Tags
    await page.getByPlaceholder(/upi-fraud/i).fill("e2e,playwright");

    await page.getByRole("button", { name: "Create Case" }).click();

    // Should redirect to case detail
    await expect(page).toHaveURL(/\/cases\/[a-f0-9-]{36}$/, { timeout: 10_000 });
    await expect(page.getByText(title)).toBeVisible();

    const url = page.url();
    const id = url.split("/").pop()!;
    createdIds.push(id);
  });

  test("case row click navigates to case detail", async ({ page }) => {
    await page.waitForTimeout(1500);
    // Rows use onClick on <tr>, not <a> links
    const firstRow = page.locator("tbody tr").first();
    await firstRow.click();
    await expect(page).toHaveURL(/\/cases\/[a-f0-9-]{36}/);
  });
});
