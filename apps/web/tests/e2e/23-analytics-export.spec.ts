import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("Analytics CSV Export", () => {
  test.beforeEach(async ({ page }) => {
    await login(page, undefined, undefined, "/workload");
    await page.waitForTimeout(1500);
  });

  // ── Workload export ────────────────────────────────────────────────────────

  test("Export Workload button is visible", async ({ page }) => {
    await expect(page.getByRole("button", { name: /Export Workload/i })).toBeVisible();
  });

  test("Export Risk button is visible", async ({ page }) => {
    await expect(page.getByRole("button", { name: /Export Risk/i })).toBeVisible();
  });

  test("Export Workload button shows Exporting… while downloading", async ({ page }) => {
    // Intercept the export request to hold it open briefly
    await page.route("**/analytics/workload/export", async (route) => {
      await new Promise((r) => setTimeout(r, 800));
      await route.continue();
    });

    const btn = page.getByRole("button", { name: /Export Workload/i });
    await btn.click();
    await expect(page.getByText("Exporting…").first()).toBeVisible({ timeout: 3_000 });
  });

  test("Export Workload button is disabled while exporting", async ({ page }) => {
    await page.route("**/analytics/workload/export", async (route) => {
      await new Promise((r) => setTimeout(r, 800));
      await route.continue();
    });

    const btn = page.getByRole("button", { name: /Export Workload/i });
    await btn.click();
    await expect(btn).toBeDisabled({ timeout: 2_000 });
  });

  test("Export Workload button re-enables after download completes", async ({ page }) => {
    // Let the real request go through
    const btn = page.getByRole("button", { name: /Export Workload/i });
    await btn.click();
    await expect(btn).toBeEnabled({ timeout: 10_000 });
  });

  test("Export Risk button shows Exporting… while downloading", async ({ page }) => {
    await page.route("**/analytics/risk/export**", async (route) => {
      await new Promise((r) => setTimeout(r, 800));
      await route.continue();
    });

    const btn = page.getByRole("button", { name: /Export Risk/i });
    await btn.click();
    await expect(page.getByText("Exporting…").first()).toBeVisible({ timeout: 3_000 });
  });

  test("Export Risk button re-enables after download completes", async ({ page }) => {
    const btn = page.getByRole("button", { name: /Export Risk/i });
    await btn.click();
    await expect(btn).toBeEnabled({ timeout: 10_000 });
  });

  // ── Workload API endpoint returns CSV ──────────────────────────────────────

  test("workload export endpoint returns CSV content-type", async ({ page }) => {
    let contentType = "";
    page.on("response", (resp) => {
      if (resp.url().includes("/analytics/workload/export")) {
        contentType = resp.headers()["content-type"] ?? "";
      }
    });
    await page.getByRole("button", { name: /Export Workload/i }).click();
    await page.waitForTimeout(3_000);
    expect(contentType).toContain("text/csv");
  });

  test("risk export endpoint returns CSV content-type", async ({ page }) => {
    let contentType = "";
    page.on("response", (resp) => {
      if (resp.url().includes("/analytics/risk/export")) {
        contentType = resp.headers()["content-type"] ?? "";
      }
    });
    await page.getByRole("button", { name: /Export Risk/i }).click();
    await page.waitForTimeout(3_000);
    expect(contentType).toContain("text/csv");
  });

  // ── Risk Leaderboard section ───────────────────────────────────────────────

  test("AI Risk Leaderboard heading is visible", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /AI Risk Leaderboard/i })).toBeVisible();
  });

  test("risk leaderboard shows table or empty state", async ({ page }) => {
    const hasTable = await page.locator("table").isVisible().catch(() => false);
    const hasEmpty = await page.getByText(/No AI-analysed cases/i).isVisible().catch(() => false);
    expect(hasTable || hasEmpty).toBe(true);
  });

  test("risk leaderboard table has expected column headers when data exists", async ({ page }) => {
    const hasTable = await page.locator("table").isVisible().catch(() => false);
    if (hasTable) {
      await expect(page.getByRole("columnheader", { name: /Case/i })).toBeVisible();
      await expect(page.getByRole("columnheader", { name: /Risk Score/i })).toBeVisible();
      await expect(page.getByRole("columnheader", { name: /SLA/i })).toBeVisible();
    }
  });

  test("summary metrics strip renders at top of page", async ({ page }) => {
    await expect(page.getByText(/Total Active Cases/i)).toBeVisible();
    await expect(page.getByText(/Total Open Tasks/i)).toBeVisible();
    await expect(page.getByText(/Overdue Tasks/i)).toBeVisible();
  });

  test("workload refresh cadence note is visible", async ({ page }) => {
    await expect(page.getByText(/Workload updates every 30 seconds/i)).toBeVisible();
  });

  test("risk refresh cadence note is visible", async ({ page }) => {
    await expect(page.getByText(/Risk data updates every 60 seconds/i)).toBeVisible();
  });
});
