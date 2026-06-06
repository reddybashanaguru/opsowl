import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("renders greeting and date", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toContainText(/Good (morning|afternoon|evening)/);
  });

  test("stat cards render with labels", async ({ page }) => {
    // Labels are hardcoded, render immediately after hydration
    await expect(page.getByText("Active Cases").first()).toBeVisible();
    await expect(page.getByText("Pending Tasks").first()).toBeVisible();
    await expect(page.getByText("My Open Cases").first()).toBeVisible();
    await expect(page.getByText("SLA Breached").first()).toBeVisible();
    // Each is inside a link
    const link = page.locator("a").filter({ hasText: "Active Cases" }).first();
    await expect(link).toBeVisible();
  });

  test("case status donut chart is visible", async ({ page }) => {
    await expect(page.getByText("Case Status Breakdown")).toBeVisible();
    await expect(page.locator("svg circle").first()).toBeVisible();
  });

  test("priority bars section renders", async ({ page }) => {
    await expect(page.getByText("Active Cases by Priority")).toBeVisible();
    // Priority labels appear after stats load; wait up to 5s
    await expect(page.locator(".space-y-2\\.5").or(page.locator(".space-y-3")).first()).toBeVisible({ timeout: 6_000 });
  });

  test("attention required panel shows", async ({ page }) => {
    await expect(page.getByText("Attention Required")).toBeVisible();
    // After stats load, links inside the panel
    await page.waitForTimeout(2000);
    const panel = page.locator("div").filter({ hasText: "Attention Required" }).last();
    await expect(panel).toBeVisible();
  });

  test("quick action links navigate correctly", async ({ page }) => {
    // Cases link
    await page.locator("a[href='/cases']").first().click();
    await expect(page).toHaveURL(/\/cases/);
    // Navigate back
    await page.goto("/dashboard");
    await page.waitForURL("**/dashboard");

    // Tasks link
    await page.locator("a[href='/tasks']").first().click();
    await expect(page).toHaveURL(/\/tasks/);
    await page.goto("/dashboard");
    await page.waitForURL("**/dashboard");

    // SLA link
    await page.locator("a[href='/sla']").first().click();
    await expect(page).toHaveURL(/\/sla/);
  });

  test("sidebar navigation links are all visible", async ({ page }) => {
    const sidebar = page.locator("aside");
    await expect(sidebar.getByRole("link", { name: "Dashboard", exact: true })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Cases/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Tasks/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Documents/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Search/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Workload/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Users/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Audit/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Inbound/ })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /Settings/ }).last()).toBeVisible();
  });

  test("sidebar Settings does not highlight on /settings/inbound", async ({ page }) => {
    await page.goto("/settings/inbound");
    await page.waitForURL("**/settings/inbound");
    const inboundLink = page.locator("aside").getByRole("link", { name: /Inbound/ });
    const settingsLink = page.locator("aside").getByRole("link", { name: /Settings/ }).last();
    await expect(inboundLink).toHaveClass(/border-cyan-400/);
    await expect(settingsLink).not.toHaveClass(/border-cyan-400/);
  });

  test("notification bell is visible in sidebar", async ({ page }) => {
    const bell = page.locator("aside").getByTitle("Notifications");
    await expect(bell).toBeVisible();
  });

  test("user name and role visible in sidebar", async ({ page }) => {
    const sidebar = page.locator("aside");
    await expect(sidebar.getByText(/Priya Sharma/)).toBeVisible();
    await expect(sidebar.getByText(/super.admin/i)).toBeVisible();
  });
});
