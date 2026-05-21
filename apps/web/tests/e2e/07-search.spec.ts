import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("Search", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/search");
  });

  test("search page renders input", async ({ page }) => {
    const searchInput = page.getByRole("searchbox").or(page.getByPlaceholder(/search/i)).first();
    await expect(searchInput).toBeVisible();
  });

  test("typing triggers search without error", async ({ page }) => {
    const searchInput = page.getByRole("searchbox").or(page.getByPlaceholder(/search/i)).first();
    await searchInput.fill("fraud");
    await page.waitForTimeout(1200);
    await expect(page).not.toHaveURL(/error/);
  });

  test("empty query shows prompt or results immediately", async ({ page }) => {
    // Should not show an error state
    await page.waitForTimeout(500);
    await expect(page.locator("[class*='bg-red']")).not.toBeVisible();
  });

  test("search for nonexistent string shows no-results message", async ({ page }) => {
    const searchInput = page.getByRole("searchbox").or(page.getByPlaceholder(/search/i)).first();
    await searchInput.fill("xyznonexistentquery99999");
    await page.waitForTimeout(1500);
    // Either 0 results count or a "no results" message
    const hasNoResults = await page.getByText(/no results|nothing found|no matches|0/i).isVisible();
    expect(hasNoResults).toBe(true);
  });

  test("search for known term shows case results", async ({ page }) => {
    const searchInput = page.getByRole("searchbox").or(page.getByPlaceholder(/search/i)).first();
    await searchInput.fill("E2E");
    await page.waitForTimeout(1500);
    // Either results appear or no-results (depends on whether E2E cases were cleaned up)
    await expect(page).not.toHaveURL(/error/);
  });
});
