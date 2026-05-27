import { test, expect } from "@playwright/test";
import { login, apiToken } from "./helpers/auth";

const API_URL = "http://localhost:8080/api/v1";

test.describe("Outbound Webhooks", () => {
  let token: string;

  test.beforeAll(async () => {
    token = await apiToken();
    // Clean up leftover E2E webhooks from prior runs
    const res = await fetch(`${API_URL}/webhooks/`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const json = await res.json() as { data: { webhooks: Array<{ id: string; name: string }> } };
    const stale = (json.data?.webhooks ?? []).filter((w) => w.name.startsWith("E2E"));
    await Promise.all(
      stale.map((w) =>
        fetch(`${API_URL}/webhooks/${w.id}`, {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        })
      )
    );
  });

  test.beforeEach(async ({ page }) => {
    await login(page, undefined, undefined, "/settings/webhooks");
  });

  test("renders page heading and Add Webhook button", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /Outbound Webhooks/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Add Webhook/i })).toBeVisible();
  });

  test("shows HMAC signing note", async ({ page }) => {
    await expect(page.getByText(/X-OpsOwl-Signature/i)).toBeVisible();
  });

  test("empty state shown when no webhooks exist", async ({ page }) => {
    await page.waitForTimeout(1500);
    const hooks = page.locator("[class*='border'][class*='rounded-lg']").filter({ hasText: /http/ });
    const count = await hooks.count();
    if (count === 0) {
      await expect(page.getByText(/No webhooks configured/i)).toBeVisible();
    }
  });

  test("opens new webhook form modal on Add Webhook click", async ({ page }) => {
    await page.getByRole("button", { name: /Add Webhook/i }).click();
    await expect(page.getByRole("heading", { name: /New Webhook/i })).toBeVisible();
    await expect(page.getByPlaceholder(/Core Banking Alert Relay/i)).toBeVisible();
    await expect(page.getByPlaceholder(/https:\/\//i)).toBeVisible();
  });

  test("Create Webhook button disabled until name and URL filled", async ({ page }) => {
    await page.getByRole("button", { name: /Add Webhook/i }).click();
    const createBtn = page.getByRole("button", { name: /Create Webhook/i });
    await expect(createBtn).toBeDisabled();
    await page.getByPlaceholder(/Core Banking Alert Relay/i).fill("E2E Webhook");
    await expect(createBtn).toBeDisabled(); // still needs URL
    await page.getByPlaceholder(/https:\/\//i).fill("https://example.com/hook");
    await expect(createBtn).toBeEnabled();
  });

  test("Cancel closes the modal", async ({ page }) => {
    await page.getByRole("button", { name: /Add Webhook/i }).click();
    await expect(page.getByRole("heading", { name: /New Webhook/i })).toBeVisible();
    await page.getByRole("button", { name: /Cancel/i }).click();
    await expect(page.getByRole("heading", { name: /New Webhook/i })).not.toBeVisible();
  });

  test("creates a webhook and it appears in the list", async ({ page }) => {
    await page.getByRole("button", { name: /Add Webhook/i }).click();
    await page.getByPlaceholder(/Core Banking Alert Relay/i).fill("E2E Create Hook");
    await page.getByPlaceholder(/https:\/\//i).fill("https://example.com/e2e-hook");
    await page.getByRole("button", { name: /Create Webhook/i }).click();

    await expect(page.getByText("E2E Create Hook")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText("https://example.com/e2e-hook")).toBeVisible();
  });

  test("webhook row shows Edit and Delete buttons", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded-lg']", { hasText: "E2E Create Hook" });
    if (await row.isVisible()) {
      await expect(row.getByRole("button", { name: /Edit/i })).toBeVisible();
      await expect(row.getByRole("button", { name: /Delete/i })).toBeVisible();
    }
  });

  test("Edit opens pre-filled form", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded-lg']", { hasText: "E2E Create Hook" });
    if (await row.isVisible()) {
      await row.getByRole("button", { name: /Edit/i }).click();
      await expect(page.getByRole("heading", { name: /Edit Webhook/i })).toBeVisible();
      const nameInput = page.getByPlaceholder(/Core Banking Alert Relay/i);
      await expect(nameInput).toHaveValue("E2E Create Hook");
    }
  });

  test("View log button expands delivery log panel", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded-lg']", { hasText: "E2E Create Hook" });
    if (await row.isVisible()) {
      await row.getByText("View log").click();
      // Either shows table headers or "No deliveries yet"
      const hasTable = await page.getByText(/Event|No deliveries yet/i).first().isVisible();
      expect(hasTable).toBe(true);
    }
  });

  test("deletes the webhook after browser confirm", async ({ page }) => {
    await page.waitForTimeout(1500);
    const row = page.locator("[class*='border'][class*='rounded-lg']", { hasText: "E2E Create Hook" });
    if (await row.isVisible()) {
      page.once("dialog", (dialog) => dialog.accept());
      await row.getByRole("button", { name: /Delete/i }).click();
      await expect(page.getByText("E2E Create Hook")).not.toBeVisible({ timeout: 6_000 });
    }
  });
});
