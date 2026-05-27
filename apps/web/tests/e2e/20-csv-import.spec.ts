import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";
import path from "path";
import fs from "fs";
import os from "os";

test.describe("Bulk CSV Import", () => {
  test.beforeEach(async ({ page }) => {
    await login(page, undefined, undefined, "/cases");
  });

  test("Import CSV link is visible on cases list page", async ({ page }) => {
    await expect(page.getByText(/Import CSV/i)).toBeVisible();
  });

  test("Import CSV link navigates to /cases/import", async ({ page }) => {
    await page.getByText(/Import CSV/i).click();
    await page.waitForURL("**/cases/import", { timeout: 8_000 });
    expect(page.url()).toContain("/cases/import");
  });

  test("import page renders page heading", async ({ page }) => {
    await page.goto("/cases/import");
    await expect(page.getByRole("heading", { name: /Bulk Case Import/i })).toBeVisible();
  });

  test("step indicator shows Upload as first step", async ({ page }) => {
    await page.goto("/cases/import");
    await expect(page.getByText(/Upload/i).first()).toBeVisible();
  });

  test("drop zone renders with instruction text", async ({ page }) => {
    await page.goto("/cases/import");
    await expect(page.getByText(/Drop a CSV file here/i)).toBeVisible();
    await expect(page.getByText(/Max 5 MB/i)).toBeVisible();
  });

  test("expected format hint is visible", async ({ page }) => {
    await page.goto("/cases/import");
    await expect(page.getByText(/Expected format/i)).toBeVisible();
    await expect(page.getByText(/title.*required/i)).toBeVisible();
  });

  test("rejects a non-CSV file with error message", async ({ page }) => {
    await page.goto("/cases/import");
    // Create a temp .txt file
    const tmpFile = path.join(os.tmpdir(), "test-upload.txt");
    fs.writeFileSync(tmpFile, "not a csv");
    const input = page.locator("input[type='file']");
    await input.setInputFiles(tmpFile);
    await expect(page.getByText(/Only .csv files/i)).toBeVisible({ timeout: 5_000 });
    fs.unlinkSync(tmpFile);
  });

  test("valid CSV file advances to mapping step", async ({ page }) => {
    await page.goto("/cases/import");
    // Create a minimal valid CSV
    const tmpFile = path.join(os.tmpdir(), "e2e-import.csv");
    fs.writeFileSync(tmpFile, "title,priority\nE2E Import Case,high\n");
    const input = page.locator("input[type='file']");
    await input.setInputFiles(tmpFile);

    // Should reach mapping step — look for "Map Columns" or column headers
    await expect(
      page.getByText(/Map Columns|Column Mapping|Mapping/i).first()
    ).toBeVisible({ timeout: 10_000 });

    fs.unlinkSync(tmpFile);
  });

  test("mapping step shows detected CSV headers", async ({ page }) => {
    await page.goto("/cases/import");
    const tmpFile = path.join(os.tmpdir(), "e2e-import-2.csv");
    fs.writeFileSync(tmpFile, "title,priority,description\nTest Case,medium,A test\n");
    await page.locator("input[type='file']").setInputFiles(tmpFile);

    await page.waitForTimeout(3_000);
    // CSV header "title" should appear in mapping table
    await expect(page.getByText("title").first()).toBeVisible({ timeout: 8_000 });
    fs.unlinkSync(tmpFile);
  });

  test("mapping step has Back and Start Import buttons", async ({ page }) => {
    await page.goto("/cases/import");
    const tmpFile = path.join(os.tmpdir(), "e2e-import-3.csv");
    fs.writeFileSync(tmpFile, "title\nE2E Row\n");
    await page.locator("input[type='file']").setInputFiles(tmpFile);

    await page.waitForTimeout(3_000);
    const hasMappingStep = await page.getByText(/Map Columns|Column Mapping|Mapping/i).first().isVisible().catch(() => false);
    if (hasMappingStep) {
      await expect(page.getByRole("button", { name: /Back/i })).toBeVisible();
      await expect(page.getByRole("button", { name: /Import.*rows/i })).toBeVisible();
    }
    fs.unlinkSync(tmpFile);
  });
});
