import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase } from "./helpers/auth";

test.describe("Breadcrumbs & Sub-page Navigation", () => {
  let token: string;
  let caseId: string;
  let caseNumber: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E Breadcrumb Test Case", "medium");
    caseId = c.id;
    caseNumber = c.case_number;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/cases/${caseId}`);
  });

  // ─── Case Detail breadcrumbs ────────────────────────────────────────────────

  test("case detail has '← Cases' breadcrumb", async ({ page }) => {
    await expect(page.getByRole("button", { name: /← Cases/i })).toBeVisible();
  });

  test("'← Cases' breadcrumb navigates to /cases", async ({ page }) => {
    await page.getByRole("button", { name: /← Cases/i }).click();
    await expect(page).toHaveURL(/\/cases$/);
  });

  test("case detail sub-nav shows all six tab buttons", async ({ page }) => {
    // CaseSubNav renders tab buttons without "→" suffix in the new layout
    await expect(page.getByRole("button", { name: /^Notes$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Tasks$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Documents$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Comms$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^AI Insights$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Timeline$/i })).toBeVisible();
  });

  // ─── Tasks sub-page breadcrumbs ─────────────────────────────────────────────

  test("tasks subpage shows '← Cases' and case-number breadcrumbs", async ({ page }) => {
    await page.getByRole("button", { name: /^Tasks$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/tasks`));
    await expect(page.getByRole("button", { name: /← Cases/i })).toBeVisible();
    await expect(page.getByRole("button", { name: caseNumber })).toBeVisible();
  });

  test("tasks subpage '← Cases' navigates to /cases", async ({ page }) => {
    await page.getByRole("button", { name: /^Tasks$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/tasks`));
    await page.getByRole("button", { name: /← Cases/i }).click();
    await expect(page).toHaveURL(/\/cases$/);
  });

  test("tasks subpage case-number breadcrumb navigates back to case detail", async ({ page }) => {
    await page.getByRole("button", { name: /^Tasks$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/tasks`));
    await page.getByRole("button", { name: caseNumber }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}$`));
  });

  // ─── Notes sub-page breadcrumbs ─────────────────────────────────────────────

  test("notes subpage shows '← Case' breadcrumb", async ({ page }) => {
    await page.getByRole("button", { name: /^Notes$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/notes`));
    await expect(page.getByRole("button", { name: /← Case/i })).toBeVisible();
  });

  test("notes subpage '← Case' navigates back to case detail", async ({ page }) => {
    await page.getByRole("button", { name: /^Notes$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/notes`));
    await page.getByRole("button", { name: /← Case/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}$`));
  });

  test("notes subpage shows cross-tab navigation buttons", async ({ page }) => {
    await page.getByRole("button", { name: /^Notes$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/notes`));
    // CaseSubNav renders tab labels without "→" suffix
    await expect(page.getByRole("button", { name: /^Tasks$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Documents$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^AI Logs$/i })).toBeVisible();
    // Verify Timeline tab navigates correctly
    const timelineCrossNav = page.getByRole("button", { name: /^Timeline$/i });
    await timelineCrossNav.click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/timeline`));
  });

  // ─── Timeline sub-page breadcrumbs ──────────────────────────────────────────

  test("timeline subpage shows '← Case' breadcrumb", async ({ page }) => {
    await page.getByRole("button", { name: /^Timeline$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/timeline`));
    await expect(page.getByRole("button", { name: /← Case/i })).toBeVisible();
  });

  test("timeline subpage '← Case' navigates back to case detail", async ({ page }) => {
    await page.getByRole("button", { name: /^Timeline$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/timeline`));
    await page.getByRole("button", { name: /← Case/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}$`));
  });

  // ─── Documents sub-page breadcrumbs ─────────────────────────────────────────

  test("documents subpage shows '← Cases' and case-label breadcrumbs", async ({ page }) => {
    await page.getByRole("button", { name: /^Documents$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/documents`));
    await expect(page.getByRole("button", { name: /← Cases/i })).toBeVisible();
    // Documents page shows cfg.case_label ("Case" or workspace-specific label) not the case number
    await expect(page.getByRole("button", { name: /^Case$|^Claim$|^Incident$|^Matter$/i }).first()).toBeVisible();
  });

  test("documents subpage '← Cases' navigates to /cases", async ({ page }) => {
    await page.getByRole("button", { name: /^Documents$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/documents`));
    await page.getByRole("button", { name: /← Cases/i }).click();
    await expect(page).toHaveURL(/\/cases$/);
  });

  test("documents subpage cross-nav navigates to tasks and timeline correctly", async ({ page }) => {
    await page.getByRole("button", { name: /^Documents$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/documents`));
    // Tasks tab (no "→" suffix in new CaseSubNav layout)
    await expect(page.getByRole("button", { name: /^Tasks$/i })).toBeVisible();
    await page.getByRole("button", { name: /^Tasks$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/tasks`));
  });

  // ─── Comms sub-page ──────────────────────────────────────────────────────────

  test("comms subpage is reachable from case detail nav", async ({ page }) => {
    await page.getByRole("button", { name: /^Comms$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/comms`));
  });

  // ─── AI Insights sub-page ───────────────────────────────────────────────────

  test("AI insights subpage is reachable from case detail nav", async ({ page }) => {
    await page.getByRole("button", { name: /^AI Insights$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/insights`));
  });

  // ─── Sidebar navigation ─────────────────────────────────────────────────────

  test("sidebar 'Cases' link navigates to /cases", async ({ page }) => {
    const sidebar = page.locator("aside");
    await sidebar.getByRole("link", { name: /Cases/i }).click();
    await expect(page).toHaveURL(/\/cases$/);
  });

  test("sidebar 'Dashboard' link navigates to /dashboard", async ({ page }) => {
    const sidebar = page.locator("aside");
    await sidebar.getByRole("link", { name: "Dashboard", exact: true }).click();
    await expect(page).toHaveURL(/\/dashboard$/);
  });
});
