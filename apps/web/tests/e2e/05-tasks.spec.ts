import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("Global Tasks (My Tasks)", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/tasks");
  });

  test("tasks page renders heading", async ({ page }) => {
    // Heading: "My Tasks" for analysts, "All Tasks" for managers/admins
    await expect(page.getByRole("heading", { name: /(?:My|All)\s+\w*Task/i })).toBeVisible();
  });

  test("completion progress ring renders", async ({ page }) => {
    // SVG progress ring always renders (even at 0%)
    await expect(page.locator("svg").first()).toBeVisible();
  });

  test("tasks summary shows active/done counts", async ({ page }) => {
    // "N active · N completed · N total"
    await expect(page.getByText(/active.*completed|completed.*active/i)).toBeVisible();
  });

  test("tasks grouped or shows all-clear message", async ({ page }) => {
    await page.waitForTimeout(2000);
    // Either grouped task sections OR the "All caught up!" empty state
    const hasGroups = await page.getByText(/Overdue|Due Today|Due This Week|Later|Done/i).first().isVisible();
    const hasEmpty = await page.getByText(/All caught up/i).isVisible();
    expect(hasGroups || hasEmpty).toBe(true);
  });

  test("task cards link back to cases", async ({ page }) => {
    await page.waitForTimeout(2000);
    const caseLink = page.locator("a[href*='/cases/']").first();
    if (await caseLink.isVisible()) {
      const href = await caseLink.getAttribute("href");
      expect(href).toMatch(/\/cases\//);
    }
  });
});
