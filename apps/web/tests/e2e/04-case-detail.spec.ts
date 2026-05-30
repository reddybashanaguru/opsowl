import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase } from "./helpers/auth";

test.describe("Case Detail", () => {
  let token: string;
  let caseId: string;
  let caseNumber: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E Detail Test Case", "high");
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

  test("renders case number and title", async ({ page }) => {
    await expect(page.getByText(caseNumber)).toBeVisible();
    await expect(page.getByText("E2E Detail Test Case")).toBeVisible();
  });

  test("priority badge visible", async ({ page }) => {
    // PriorityBadge renders priority text
    await expect(page.getByText(/High/i).first()).toBeVisible();
  });

  test("status badge visible", async ({ page }) => {
    // Case is newly created — status should be 'open' (after auto-transition) or 'draft'
    // StatusBadge renders the status text
    await page.waitForTimeout(1000);
    const statusEl = page.locator('[class*="rounded"]').filter({
      hasText: /Open|Draft|In Progress|Pending Review|Escalated|Closed/i,
    }).first();
    await expect(statusEl).toBeVisible();
  });

  test("tab navigation — case detail nav bar visible", async ({ page }) => {
    // CaseSubNav renders inline tab buttons (no arrows)
    await expect(page.getByRole("button", { name: /^Notes$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Tasks$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Documents$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Comms$/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /AI Insights/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /^Timeline$/i })).toBeVisible();
  });

  test("tab navigation — Tasks", async ({ page }) => {
    await page.getByRole("button", { name: /^Tasks$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/tasks`));
  });

  test("tab navigation — Notes", async ({ page }) => {
    await page.getByRole("button", { name: /^Notes$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/notes`));
  });

  test("tab navigation — Documents", async ({ page }) => {
    await page.getByRole("button", { name: /^Documents$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/documents`));
  });

  test("tab navigation — Timeline", async ({ page }) => {
    await page.getByRole("button", { name: /^Timeline$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/timeline`));
    // TimelineEventCard renders EVENT_CONFIG label "Case Created" for case_created events
    await expect(page.getByText(/Case Created/i).first()).toBeVisible({ timeout: 8_000 });
  });

  test("tab navigation — Comms", async ({ page }) => {
    await page.getByRole("button", { name: /^Comms$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/comms`));
  });

  test("tab navigation — AI Insights", async ({ page }) => {
    await page.getByRole("button", { name: /AI Insights/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/insights`));
  });

  test("can create a task from the tasks tab", async ({ page }) => {
    await page.getByRole("button", { name: /^Tasks$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/tasks`));
    const newTaskBtn = page.getByRole("button", { name: /New Task/i });
    await expect(newTaskBtn).toBeVisible({ timeout: 10_000 });
    await newTaskBtn.click();
    await expect(page.getByPlaceholder("Task title…")).toBeVisible({ timeout: 5_000 });
    await page.getByPlaceholder("Task title…").fill("E2E automated task");
    await page.getByRole("button", { name: "Create Task" }).click();
    await page.waitForTimeout(1000);
    await expect(page.getByText("E2E automated task")).toBeVisible({ timeout: 5_000 });
  });

  test("can add a note from the notes tab", async ({ page }) => {
    await page.getByRole("button", { name: /^Notes$/i }).click();
    await expect(page).toHaveURL(new RegExp(`/cases/${caseId}/notes`));
    const newNoteBtn = page.getByRole("button", { name: "+ New Note" });
    await expect(newNoteBtn).toBeVisible({ timeout: 10_000 });
    await newNoteBtn.click();
    const titleInput = page.getByPlaceholder("Note title (optional)");
    await expect(titleInput).toBeVisible({ timeout: 5_000 });
    await titleInput.fill("E2E test note");
    const editor = page.locator(".ProseMirror").first();
    if (await editor.isVisible()) {
      await editor.click();
      await page.keyboard.type("This note was created by an E2E test.");
    }
    await page.getByRole("button", { name: "Save Note" }).click();
    await page.waitForTimeout(1000);
    await expect(page.getByText("E2E test note")).toBeVisible({ timeout: 5_000 });
  });

  test("AI Copilot button opens panel", async ({ page }) => {
    const copilotBtn = page.getByRole("button", { name: /🤖 AI Copilot/ });
    await expect(copilotBtn).toBeVisible();
    await copilotBtn.click();
    // Panel should open with an input
    await expect(page.getByPlaceholder(/Ask|message/i)).toBeVisible({ timeout: 5_000 });
  });

  test("export PDF button visible", async ({ page }) => {
    await expect(page.getByRole("button", { name: /PDF|Export/i })).toBeVisible();
  });
});
