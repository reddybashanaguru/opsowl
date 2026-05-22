import { test, expect } from "@playwright/test";
import { login, apiToken } from "./helpers/auth";

const API_URL = "http://localhost:8080/api/v1";

test.describe("API Keys management", () => {
  let token: string;

  test.beforeAll(async () => {
    token = await apiToken();
    // Clean up E2E test keys left over from previous runs
    const res = await fetch(`${API_URL}/api-keys`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const json = await res.json() as { data: { api_keys: Array<{ id: string; name: string }> } };
    const testKeys = (json.data?.api_keys ?? []).filter((k) => k.name.startsWith("E2E"));
    await Promise.all(testKeys.map((k) =>
      fetch(`${API_URL}/api-keys/${k.id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      })
    ));
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/settings/api-keys");
  });

  test("renders page heading and generate button", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /API Keys/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Generate Key/i })).toBeVisible();
  });

  test("table headers are present", async ({ page }) => {
    await expect(page.getByRole("columnheader", { name: /Name/i })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: /Scopes/i })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: /Expires/i })).toBeVisible();
  });

  test("opens create key modal", async ({ page }) => {
    await page.getByRole("button", { name: /Generate Key/i }).click();
    await expect(page.getByRole("heading", { name: /New API Key/i })).toBeVisible();
    await expect(page.getByLabel(/Key Name/i)).toBeVisible();
  });

  test("create key modal requires a name", async ({ page }) => {
    await page.getByRole("button", { name: /Generate Key/i }).click();
    const createBtn = page.getByRole("button", { name: /Create Key/i });
    await expect(createBtn).toBeDisabled();
    await page.getByLabel(/Key Name/i).fill("E2E Test Key");
    await expect(createBtn).toBeEnabled();
  });

  test("creates key and shows reveal banner", async ({ page }) => {
    await page.getByRole("button", { name: /Generate Key/i }).click();
    await page.getByLabel(/Key Name/i).fill("E2E Playwright Key");
    await page.getByRole("button", { name: /Create Key/i }).click();

    // Reveal banner should appear with the full key
    const banner = page.locator("text=API key created — copy it now");
    await expect(banner).toBeVisible({ timeout: 8_000 });

    // Copy button is present
    await expect(page.getByRole("button", { name: /Copy key/i })).toBeVisible();

    // The new key appears in the table after banner is dismissed
    await page.locator("button", { hasText: "✕" }).first().click();
    await expect(page.getByText("E2E Playwright Key").first()).toBeVisible();
  });

  test("revoke flow shows inline confirm and removes key", async ({ page }) => {
    // First ensure we have a key to revoke (create one)
    await page.getByRole("button", { name: /Generate Key/i }).click();
    await page.getByLabel(/Key Name/i).fill("E2E Revoke Key");
    await page.getByRole("button", { name: /Create Key/i }).click();
    await expect(page.locator("text=API key created")).toBeVisible({ timeout: 8_000 });
    await page.locator("button", { hasText: "✕" }).first().click();

    // Find the revoke button for our key
    const row = page.locator("tr", { hasText: "E2E Revoke Key" });
    await expect(row).toBeVisible();
    await row.getByRole("button", { name: /Revoke/i }).click();

    // Confirm step appears
    await expect(row.getByText("Revoke?")).toBeVisible();
    await row.getByRole("button", { name: "Yes" }).click();

    // Key is removed from the table
    await expect(page.getByText("E2E Revoke Key")).not.toBeVisible({ timeout: 5_000 });
  });

  test("cancel modal on close button", async ({ page }) => {
    await page.getByRole("button", { name: /Generate Key/i }).click();
    await expect(page.getByRole("heading", { name: /New API Key/i })).toBeVisible();
    await page.getByRole("button", { name: /Cancel/i }).click();
    await expect(page.getByRole("heading", { name: /New API Key/i })).not.toBeVisible();
  });
});
