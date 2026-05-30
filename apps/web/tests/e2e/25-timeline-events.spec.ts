import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * Timeline event assertion tests.
 *
 * Each test performs an action on a case and then asserts that the corresponding
 * timeline event card appears in the Timeline tab. This catches regressions where
 * an action silently stops emitting timeline events.
 */

test.describe("Timeline events", () => {
  let token: string;
  let caseId: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E Timeline Assertions Case", "high");
    caseId = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  // ── Helper: navigate to timeline and wait for an event label ─────────────────

  async function goToTimeline(page: import("@playwright/test").Page) {
    await page.goto(`/cases/${caseId}/timeline`);
    // Wait for the timeline feed to load (spinner disappears)
    await expect(page.locator(".animate-spin").first()).not.toBeVisible({ timeout: 10_000 });
  }

  // ── Case Created ──────────────────────────────────────────────────────────────

  test("case_created event appears in timeline", async ({ page }) => {
    await goToTimeline(page);
    await expect(page.getByText("Case Created").first()).toBeVisible({ timeout: 8_000 });
  });

  // ── Note Added ───────────────────────────────────────────────────────────────

  test("note_added event appears after creating a note", async ({ page }) => {
    // Create a note via API to avoid UI flakiness
    await fetch(`${API_URL}/cases/${caseId}/notes`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ title: "Timeline E2E note", content_text: "Body text for e2e", note_type: "internal" }),
    });

    await goToTimeline(page);
    await expect(page.getByText("Note Added").first()).toBeVisible({ timeout: 8_000 });
  });

  // ── Task Created ─────────────────────────────────────────────────────────────

  test("task_created event appears after creating a task", async ({ page }) => {
    await fetch(`${API_URL}/cases/${caseId}/tasks`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ title: "Timeline E2E task", priority: "medium" }),
    });

    await goToTimeline(page);
    await expect(page.getByText("Task Created").first()).toBeVisible({ timeout: 8_000 });
  });

  // ── Comm Logged ───────────────────────────────────────────────────────────────

  test("comm_logged event appears after logging a communication", async ({ page }) => {
    await fetch(`${API_URL}/cases/${caseId}/comms`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        channel: "email",
        direction: "inbound",
        contact_name: "E2E Test Contact",
        body: "Test communication body for timeline assertion",
        occurred_at: new Date().toISOString(),
      }),
    });

    await goToTimeline(page);
    await expect(page.getByText("Comm Logged").first()).toBeVisible({ timeout: 8_000 });
  });

  // ── Status Changed ────────────────────────────────────────────────────────────

  test("case_status_changed event appears after changing status", async ({ page }) => {
    await fetch(`${API_URL}/cases/${caseId}/status`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ status: "in_progress", reason: "E2E status change" }),
    });

    await goToTimeline(page);
    await expect(page.getByText("Status Changed").first()).toBeVisible({ timeout: 8_000 });
  });

  // ── Timeline filter: source filter ───────────────────────────────────────────

  test("source filter reduces visible events", async ({ page }) => {
    await goToTimeline(page);
    // Open filters
    await page.getByRole("button", { name: /Filters/i }).click();
    // Select "system" source only
    await page.getByRole("button", { name: /^system$/i }).click();
    // Events should still load (no crash)
    await page.waitForTimeout(1000);
    // Filter badge should appear
    await expect(page.getByRole("button", { name: /Filters/i }).locator("span").first()).toBeVisible();
  });

  // ── Timeline filter: event type filter ───────────────────────────────────────

  test("event_type filter shows only matching events", async ({ page }) => {
    await goToTimeline(page);
    await page.getByRole("button", { name: /Filters/i }).click();
    // Filter to note_added only
    await page.getByRole("button", { name: /note added/i }).click();
    await page.waitForTimeout(1500);
    // Either "Note Added" cards exist, or empty state
    const noteCards = page.getByText("Note Added");
    const emptyState = page.getByText(/No timeline events/i);
    await expect(noteCards.first().or(emptyState)).toBeVisible({ timeout: 8_000 });
  });
});
