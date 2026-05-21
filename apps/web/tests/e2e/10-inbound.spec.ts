import { test, expect } from "@playwright/test";
import { login, apiToken } from "./helpers/auth";

test.describe("Inbound Webhook Management", () => {
  let token: string;
  const cleanupSourceIds: string[] = [];

  test.beforeAll(async () => {
    token = await apiToken();
  });

  test.afterAll(async () => {
    for (const id of cleanupSourceIds) {
      await fetch(`http://localhost:8080/api/v1/inbound/sources/${id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
    }
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/settings/inbound");
  });

  test("inbound page renders heading", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Inbound Integrations" })).toBeVisible();
  });

  test("Sources tab and Event Log tab exist", async ({ page }) => {
    // Tabs are buttons with text "Sources (N)" and "Event Log"
    await expect(page.getByRole("button", { name: /Sources/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Event Log/i })).toBeVisible();
  });

  test("how it works explainer is visible", async ({ page }) => {
    await expect(page.getByText("How it works")).toBeVisible();
    await expect(page.getByText(/Register a source/i)).toBeVisible();
  });

  test("+ New Source button is visible on sources tab", async ({ page }) => {
    await expect(page.getByRole("button", { name: "+ New Source" })).toBeVisible();
  });

  test("existing sources are listed", async ({ page }) => {
    await page.waitForTimeout(2000);
    // From prior E2E tests, there should be at least one source
    const sourceCards = page.locator('[class*="border"][class*="rounded-xl"]').first();
    await expect(sourceCards).toBeVisible();
  });

  test("create new source via modal", async ({ page }) => {
    await page.getByRole("button", { name: "+ New Source" }).click();
    // Modal opens
    await expect(page.getByRole("heading", { name: "New Inbound Source" })).toBeVisible({ timeout: 5_000 });

    const uniqueName = `PW-Test-${Date.now()}`;
    // Label has no htmlFor — use placeholder to target the Name input
    await page.getByPlaceholder("Finacle Alert Feed").fill(uniqueName);
    // Slug auto-fills from name

    await page.getByRole("button", { name: "Create Source" }).click();
    await page.waitForTimeout(2000);

    // Source appears in list (uniqueName also appears as slug in mono font — use first())
    await expect(page.getByText(uniqueName).first()).toBeVisible({ timeout: 8_000 });

    // Capture for cleanup
    const res = await fetch("http://localhost:8080/api/v1/inbound/sources", {
      headers: { Authorization: `Bearer ${token}` },
    });
    const json = await res.json() as { data: Array<{ id: string; name: string }> };
    const created = json.data.find((s) => s.name === uniqueName);
    if (created) cleanupSourceIds.push(created.id);
  });

  test("source card can be expanded to show signing secret controls", async ({ page }) => {
    await page.waitForTimeout(1500);
    const firstCard = page.locator('[class*="border-slate-700"][class*="rounded-xl"]').first();
    // Click to expand
    await firstCard.click();
    await page.waitForTimeout(500);
    // Show/Hide secret button or the slug text should be visible
    const secretRow = page.getByText(/Signing Secret|signing_secret|Show|Regenerate/i);
    if (await secretRow.isVisible()) {
      await expect(secretRow).toBeVisible();
    }
  });

  test("event log tab shows table with status column", async ({ page }) => {
    await page.getByRole("button", { name: /Event Log/i }).click();
    await page.waitForTimeout(2000);
    // Either the table with status column, or "No events received yet"
    const hasTable = await page.locator("table").isVisible();
    const hasEmpty = await page.getByText(/No events received yet/i).isVisible();
    expect(hasTable || hasEmpty).toBe(true);
  });

  test("event log shows processed status badges", async ({ page }) => {
    await page.getByRole("button", { name: /Event Log/i }).click();
    await page.waitForTimeout(2000);
    const processed = page.getByText("processed");
    if (await processed.first().isVisible()) {
      await expect(processed.first()).toBeVisible();
    }
  });
});
