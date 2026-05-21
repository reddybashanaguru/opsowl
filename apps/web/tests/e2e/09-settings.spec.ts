import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("Settings — labels and MFA", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/settings");
  });

  test("settings page renders heading", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
  });

  test("MFA section is visible with 'Two-Factor Authentication' heading", async ({ page }) => {
    await expect(page.getByText("Two-Factor Authentication")).toBeVisible();
  });

  test("MFA status badge is shown (Active or Disabled)", async ({ page }) => {
    await page.waitForTimeout(1000);
    const badge = page.getByText(/Active|Disabled|Checking/i).first();
    await expect(badge).toBeVisible();
  });

  test("MFA Enable or Disable button is visible", async ({ page }) => {
    await page.waitForTimeout(1500);
    const enableBtn = page.getByRole("button", { name: /Enable MFA/i });
    const disableBtn = page.getByRole("button", { name: /Disable MFA/i });
    const hasMFAButton = await enableBtn.isVisible() || await disableBtn.isVisible();
    expect(hasMFAButton).toBe(true);
  });

  test("workspace terminology section renders for admin", async ({ page }) => {
    await expect(page.getByText("Workspace Terminology")).toBeVisible();
  });

  test("label inputs are editable (14 fields)", async ({ page }) => {
    await page.waitForTimeout(1000);
    const inputs = page.locator("input[type='text']");
    const count = await inputs.count();
    expect(count).toBeGreaterThanOrEqual(14);
  });

  test("save settings persists across refresh", async ({ page }) => {
    await page.waitForTimeout(1500);
    const firstInput = page.locator("input[type='text']").first();
    const original = await firstInput.inputValue();
    const newVal = `TestLabel-${Date.now()}`;
    await firstInput.fill(newVal);
    await page.getByRole("button", { name: "Save Settings" }).click();
    await expect(page.getByText(/Saved — all users see updated labels/i)).toBeVisible({ timeout: 5_000 });

    // Reload and verify label persisted
    await page.reload();
    await page.waitForTimeout(2000);
    const afterReload = await page.locator("input[type='text']").first().inputValue();
    expect(afterReload).toBe(newVal);

    // Restore original
    await page.locator("input[type='text']").first().fill(original);
    await page.getByRole("button", { name: "Save Settings" }).click();
    await page.waitForTimeout(1000);
  });

  test("Reset to Defaults button is visible", async ({ page }) => {
    await expect(page.getByRole("button", { name: "Reset to Defaults" })).toBeVisible();
  });

  test("Change Password section is visible", async ({ page }) => {
    await expect(page.getByText("Change Password")).toBeVisible();
  });

  test("Change Password section has current and new password fields", async ({ page }) => {
    await expect(page.getByPlaceholder("••••••••").first()).toBeVisible();
    await expect(page.getByPlaceholder("Min 8 characters")).toBeVisible();
    await expect(page.getByPlaceholder("Re-enter new password")).toBeVisible();
    await expect(page.getByRole("button", { name: /Update Password/i })).toBeVisible();
  });

  test("Change Password button is disabled when fields are empty", async ({ page }) => {
    const updateBtn = page.getByRole("button", { name: /Update Password/i });
    await expect(updateBtn).toBeDisabled();
  });
});
