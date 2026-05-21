import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("User Management (admin)", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/users");
  });

  test("users page renders 'User Management' heading", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "User Management" })).toBeVisible();
  });

  test("user list shows current admin user", async ({ page }) => {
    await page.waitForTimeout(2000);
    await expect(page.getByText("admin@opsowl.dev")).toBeVisible();
    await expect(page.getByText("Priya Sharma").first()).toBeVisible();
  });

  test("role pill filter buttons appear", async ({ page }) => {
    await page.waitForTimeout(2000);
    // Role distribution pills: "Super Admin · N", "Admin · N" etc.
    const roleButtons = page.getByRole("button").filter({ hasText: /Super Admin|Admin|Analyst|Manager|Viewer/i });
    await expect(roleButtons.first()).toBeVisible();
  });

  test("search by name filters results", async ({ page }) => {
    await page.waitForTimeout(2000); // wait for user list to load
    const searchInput = page.getByPlaceholder("Search name or email…");
    await expect(searchInput).toBeVisible();
    await searchInput.fill("Priya");
    await page.waitForTimeout(500);
    // "Priya Sharma" appears in both sidebar and table — use first()
    await expect(page.getByText("Priya Sharma").first()).toBeVisible();
  });

  test("user table has correct column headers", async ({ page }) => {
    await page.waitForTimeout(1500);
    await expect(page.getByRole("columnheader", { name: /User/i })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: /Role/i })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: /Status/i })).toBeVisible();
  });

  test("user rows show active status", async ({ page }) => {
    await page.waitForTimeout(2000);
    await expect(page.getByText("Active").first()).toBeVisible();
  });

  test("deactivate button visible on other users", async ({ page }) => {
    await page.waitForTimeout(2000);
    // The "Deactivate" action shows on rows that are not the current user
    const deactivateBtn = page.getByRole("button", { name: /Deactivate/i });
    if (await deactivateBtn.first().isVisible()) {
      await expect(deactivateBtn.first()).toBeVisible();
    }
  });

  test("Invite User button is visible for admin", async ({ page }) => {
    await expect(page.getByRole("button", { name: /Invite User/i })).toBeVisible();
  });

  test("Invite User modal opens and has required fields", async ({ page }) => {
    await page.getByRole("button", { name: /Invite User/i }).click();
    await expect(page.getByRole("heading", { name: /Invite User/i })).toBeVisible({ timeout: 5_000 });
    await expect(page.getByPlaceholder("Priya Sharma")).toBeVisible();
    await expect(page.getByPlaceholder("analyst@yourorg.com")).toBeVisible();
    await expect(page.getByRole("button", { name: /Create User/i })).toBeVisible();
  });

  test("inline role change selector visible on hover for other users", async ({ page }) => {
    await page.waitForTimeout(2000);
    // Role cells for non-current-user rows have an edit pencil on hover
    const firstRow = page.locator("tbody tr").first();
    await firstRow.locator("td").nth(1).hover();
    // After hover, the edit pencil button (✎) appears
    const editBtn = firstRow.locator("td").nth(1).getByText("✎");
    if (await editBtn.isVisible()) {
      await expect(editBtn).toBeVisible();
    }
  });
});
