import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase } from "./helpers/auth";

test.describe("Cases — bulk operations", () => {
  let token: string;
  const createdIds: string[] = [];

  test.beforeAll(async () => {
    token = await apiToken();
    // Seed three cases for bulk testing
    for (let i = 1; i <= 3; i++) {
      const c = await createCase(token, `E2E Bulk Case ${i}`, "medium");
      createdIds.push(c.id);
    }
  });

  test.afterAll(async () => {
    for (const id of createdIds) {
      await deleteCase(token, id);
    }
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/cases");
    await page.waitForTimeout(1500); // let cases load
  });

  test("checkbox column is visible in the table header", async ({ page }) => {
    const selectAll = page.getByRole("checkbox", { name: /Select all/i });
    await expect(selectAll).toBeVisible();
  });

  test("row checkboxes are present", async ({ page }) => {
    const rows = page.locator("tbody tr").filter({ hasNotText: "animate-pulse" });
    const firstRow = rows.first();
    const checkbox = firstRow.locator("input[type='checkbox']");
    await expect(checkbox).toBeVisible({ timeout: 5_000 });
  });

  test("selecting a row shows the bulk action bar", async ({ page }) => {
    const rows = page.locator("tbody tr").filter({ hasNotText: "animate-pulse" });
    await rows.first().locator("input[type='checkbox']").check();
    await expect(page.getByTestId("bulk-action-bar")).toBeVisible();
    await expect(page.getByText(/1 selected/)).toBeVisible();
  });

  test("select-all selects all visible rows", async ({ page }) => {
    await page.getByRole("checkbox", { name: /Select all/i }).check();
    const bar = page.getByTestId("bulk-action-bar");
    await expect(bar).toBeVisible();
    // Count should be > 0
    await expect(bar.locator("text=/\\d+ selected/")).toBeVisible();
  });

  test("deselecting all hides the action bar", async ({ page }) => {
    await page.getByRole("checkbox", { name: /Select all/i }).check();
    await expect(page.getByTestId("bulk-action-bar")).toBeVisible();
    await page.getByRole("checkbox", { name: /Select all/i }).uncheck();
    await expect(page.getByTestId("bulk-action-bar")).not.toBeVisible();
  });

  test("clear button in action bar deselects all", async ({ page }) => {
    await page.getByRole("checkbox", { name: /Select all/i }).check();
    await expect(page.getByTestId("bulk-action-bar")).toBeVisible();
    await page.getByTestId("bulk-action-bar").locator("button", { hasText: "✕" }).click();
    await expect(page.getByTestId("bulk-action-bar")).not.toBeVisible();
  });

  test("Change Status button opens status modal", async ({ page }) => {
    const rows = page.locator("tbody tr").filter({ hasNotText: "animate-pulse" });
    await rows.first().locator("input[type='checkbox']").check();
    await page.getByTestId("bulk-action-bar").getByRole("button", { name: /Change Status/i }).click();
    await expect(page.getByRole("heading", { name: /Change Status/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Apply/i })).toBeVisible();
  });

  test("bulk status modal cancel closes it", async ({ page }) => {
    const rows = page.locator("tbody tr").filter({ hasNotText: "animate-pulse" });
    await rows.first().locator("input[type='checkbox']").check();
    await page.getByTestId("bulk-action-bar").getByRole("button", { name: /Change Status/i }).click();
    await page.getByRole("button", { name: /Cancel/i }).click();
    await expect(page.getByRole("heading", { name: /Change Status/i })).not.toBeVisible();
  });

  test("Reassign button opens assign modal", async ({ page }) => {
    const rows = page.locator("tbody tr").filter({ hasNotText: "animate-pulse" });
    await rows.first().locator("input[type='checkbox']").check();
    await page.getByTestId("bulk-action-bar").getByRole("button", { name: /Reassign/i }).click();
    await expect(page.getByRole("heading", { name: /Assign Cases/i })).toBeVisible();
  });

  test("Delete button opens delete confirm modal", async ({ page }) => {
    const rows = page.locator("tbody tr").filter({ hasNotText: "animate-pulse" });
    await rows.first().locator("input[type='checkbox']").check();
    await page.getByTestId("bulk-action-bar").getByRole("button", { name: /Delete/i }).click();
    await expect(page.getByRole("heading", { name: /Delete Cases/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Delete \d+ case/i })).toBeVisible();
  });

  test("bulk status change applies and shows toast", async ({ page }) => {
    // Select one of the seeded e2e cases
    const row = page.locator("tbody tr", { hasText: "E2E Bulk Case 1" });
    await expect(row).toBeVisible({ timeout: 6_000 });
    await row.locator("input[type='checkbox']").check();

    await page.getByTestId("bulk-action-bar").getByRole("button", { name: /Change Status/i }).click();
    await page.locator("select").last().selectOption("open");
    await page.getByRole("button", { name: /Apply/i }).click();

    // Toast confirms update
    await expect(page.locator("text=/Updated \\d+ case/")).toBeVisible({ timeout: 8_000 });
    // Action bar is gone
    await expect(page.getByTestId("bulk-action-bar")).not.toBeVisible({ timeout: 5_000 });
  });
});
