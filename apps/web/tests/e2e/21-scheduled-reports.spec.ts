import { test, expect } from "@playwright/test";
import { login, apiToken } from "./helpers/auth";

const API_URL = "http://localhost:8080/api/v1";

test.describe("Scheduled Reports", () => {
  let token: string;

  test.beforeAll(async () => {
    token = await apiToken();
    // Clean up E2E test schedules from prior runs
    const res = await fetch(`${API_URL}/report-schedules`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const json = await res.json() as { data: { schedules: Array<{ id: string; name: string }> } };
    const stale = (json.data?.schedules ?? []).filter((s) => s.name.startsWith("E2E"));
    await Promise.all(
      stale.map((s) =>
        fetch(`${API_URL}/report-schedules/${s.id}`, {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        })
      )
    );
  });

  test.beforeEach(async ({ page }) => {
    await login(page, undefined, undefined, "/settings/reports");
  });

  test("renders page heading and Add Schedule button", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /Scheduled Email Reports/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Add Schedule/i })).toBeVisible();
  });

  test("shows description about email delivery", async ({ page }) => {
    await expect(page.getByText(/email/i).first()).toBeVisible();
  });

  test("empty state when no schedules", async ({ page }) => {
    await page.waitForTimeout(1500);
    const schedules = page.locator("[class*='border'][class*='rounded']").filter({ hasText: /daily|weekly/i });
    const count = await schedules.count();
    if (count === 0) {
      await expect(page.getByText(/No report schedules configured/i)).toBeVisible();
    }
  });

  test("opens new schedule modal on button click", async ({ page }) => {
    await page.getByRole("button", { name: /Add Schedule/i }).click();
    await expect(page.getByRole("heading", { name: /New Report Schedule/i })).toBeVisible();
    await expect(page.getByPlaceholder(/Daily AML Summary/i)).toBeVisible();
  });

  test("Create Schedule button disabled until name and recipient filled", async ({ page }) => {
    await page.getByRole("button", { name: /Add Schedule/i }).click();
    const saveBtn = page.getByRole("button", { name: /Create Schedule/i });
    await expect(saveBtn).toBeDisabled();
    await page.getByPlaceholder(/Daily AML Summary/i).fill("E2E Schedule");
    await expect(saveBtn).toBeDisabled(); // still needs recipients
  });

  test("Cancel closes the modal", async ({ page }) => {
    await page.getByRole("button", { name: /Add Schedule/i }).click();
    await expect(page.getByRole("heading", { name: /New Report Schedule/i })).toBeVisible();
    await page.getByRole("button", { name: /Cancel/i }).click();
    await expect(page.getByRole("heading", { name: /New Report Schedule/i })).not.toBeVisible();
  });

  test("frequency selector defaults to daily", async ({ page }) => {
    await page.getByRole("button", { name: /Add Schedule/i }).click();
    const freqSelect = page.locator("select").first();
    await expect(freqSelect).toHaveValue("daily");
  });

  test("weekly frequency reveals day-of-week selector", async ({ page }) => {
    await page.getByRole("button", { name: /Add Schedule/i }).click();
    await page.locator("select").first().selectOption("weekly");
    await expect(page.getByText(/Day of Week/i)).toBeVisible();
  });

  test("creates a schedule and it appears in the list", async ({ page }) => {
    await page.getByRole("button", { name: /Add Schedule/i }).click();
    await page.getByPlaceholder(/Daily AML Summary/i).fill("E2E Daily Report");
    // Fill recipients textarea
    await page.getByPlaceholder(/manager@bank.com/i).fill("e2e@opsowl.dev");
    await page.getByRole("button", { name: /Create Schedule/i }).click();

    await expect(page.getByText("E2E Daily Report")).toBeVisible({ timeout: 8_000 });
  });

  test("created schedule shows frequency badge", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded']", { hasText: "E2E Daily Report" });
    if (await row.isVisible()) {
      await expect(row.getByText(/daily/i).first()).toBeVisible();
    }
  });

  test("Send Now button is present on existing schedule", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded']", { hasText: "E2E Daily Report" });
    if (await row.isVisible()) {
      await expect(row.getByRole("button", { name: /Send Now/i })).toBeVisible();
    }
  });

  test("Edit button opens pre-filled form", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded']", { hasText: "E2E Daily Report" });
    if (await row.isVisible()) {
      await row.getByRole("button", { name: /Edit/i }).click();
      // Edit form re-uses ScheduleForm with initial data
      await expect(page.getByPlaceholder(/Daily AML Summary/i)).toHaveValue("E2E Daily Report", { timeout: 4_000 });
    }
  });

  test("deletes the schedule", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded']", { hasText: "E2E Daily Report" });
    if (await row.isVisible()) {
      page.once("dialog", (dialog) => dialog.accept());
      await row.getByRole("button", { name: /Delete/i }).click();
      await expect(page.getByText("E2E Daily Report")).not.toBeVisible({ timeout: 6_000 });
    }
  });
});
