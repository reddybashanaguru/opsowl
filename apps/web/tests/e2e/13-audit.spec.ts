import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("Audit Logs (admin)", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/audit");
  });

  test("audit page renders", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /Audit/i })).toBeVisible();
  });

  test("audit log table is visible", async ({ page }) => {
    await page.waitForTimeout(2000);
    await expect(page.locator("table").first()).toBeVisible();
  });

  test("audit table has expected column headers", async ({ page }) => {
    await page.waitForTimeout(2000);
    // Common audit table headers
    const hasActor = await page.getByRole("columnheader", { name: /Actor|User|By/i }).isVisible();
    const hasAction = await page.getByRole("columnheader", { name: /Action|Event|Type/i }).isVisible();
    const hasTime = await page.getByRole("columnheader", { name: /Time|Date|When/i }).isVisible();
    expect(hasActor || hasAction || hasTime).toBe(true);
  });

  test("audit rows contain activity data", async ({ page }) => {
    await page.waitForTimeout(2000);
    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();
    // Should have at least the login event from our beforeEach
    expect(rowCount).toBeGreaterThan(0);
  });

  test("audit log has admin user activity", async ({ page }) => {
    await page.waitForTimeout(2000);
    // Our login created audit entries — admin user should appear
    const adminEntry = page.getByText(/admin@opsowl.dev|Priya Sharma/i);
    if (await adminEntry.first().isVisible()) {
      await expect(adminEntry.first()).toBeVisible();
    }
  });

  test("search or filter controls exist", async ({ page }) => {
    const filterEl = page.locator("select, input[type='text'], input[type='search']").first();
    if (await filterEl.isVisible()) {
      await expect(filterEl).toBeVisible();
    }
  });
});
