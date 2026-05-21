import { test, expect } from "@playwright/test";

test.describe("Authentication", () => {
  test("login page renders correctly", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByText("OpsOwl AI", { exact: true })).toBeVisible();
    await expect(page.getByText("Sign in to your workspace")).toBeVisible();
    await expect(page.getByLabel("Email address")).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
  });

  test("invalid credentials shows error", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email address").fill("wrong@opsowl.dev");
    await page.getByLabel("Password").fill("WrongPass@123");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByText("Invalid email or password")).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test("empty form shows validation errors", async ({ page }) => {
    await page.goto("/login");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByText("Enter a valid email address")).toBeVisible();
  });

  test("protected route redirects to login", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(page).toHaveURL(/\/login/);
  });

  test("login with valid credentials redirects to dashboard", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email address").fill("admin@opsowl.dev");
    await page.getByLabel("Password").fill("OpsOwl@2024");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 });
    // Sidebar brand visible
    await expect(page.locator("aside").getByText("OpsOwl AI")).toBeVisible();
  });

  test("sidebar shows admin navigation after login", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email address").fill("admin@opsowl.dev");
    await page.getByLabel("Password").fill("OpsOwl@2024");
    await page.getByRole("button", { name: "Sign in" }).click();
    await page.waitForURL("**/dashboard");
    await expect(page.getByRole("link", { name: /Users/ })).toBeVisible();
    await expect(page.getByRole("link", { name: /Audit/ })).toBeVisible();
    await expect(page.getByRole("link", { name: /Inbound/ })).toBeVisible();
  });

  test("sign out redirects to login", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email address").fill("admin@opsowl.dev");
    await page.getByLabel("Password").fill("OpsOwl@2024");
    await page.getByRole("button", { name: "Sign in" }).click();
    await page.waitForURL("**/dashboard");
    await page.getByRole("button", { name: /Sign out/ }).click();
    await expect(page).toHaveURL(/\/login/);
  });
});
