import { test, expect } from "@playwright/test";
import { login } from "./helpers/auth";

test.describe("SLA Policy Engine", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto("/sla");
  });

  test("SLA page renders heading", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "SLA Policy Engine" })).toBeVisible();
  });

  test("info banner about how SLA works is visible", async ({ page }) => {
    await expect(page.getByText(/How SLA works/i)).toBeVisible();
  });

  test("shows all four priority policies", async ({ page }) => {
    await page.waitForTimeout(2000);
    // Policies render as priority labels in cards
    await expect(page.getByText("critical").first()).toBeVisible();
    await expect(page.getByText("high").first()).toBeVisible();
    await expect(page.getByText("medium").first()).toBeVisible();
    await expect(page.getByText("low").first()).toBeVisible();
  });

  test("SLA policy cards show breach and warning hours", async ({ page }) => {
    await page.waitForTimeout(2000);
    // Each policy card shows "Breach at Xh · Warn at Yh"
    await expect(page.getByText(/Breach at/i).first()).toBeVisible();
    await expect(page.getByText(/Warn at/i).first()).toBeVisible();
  });

  test("SLA progress bar renders", async ({ page }) => {
    await page.waitForTimeout(2000);
    const progressBar = page.locator(".h-2.rounded-full").first();
    await expect(progressBar).toBeVisible();
  });

  test("Edit button visible on each policy card", async ({ page }) => {
    await page.waitForTimeout(2000);
    const editBtn = page.getByRole("button", { name: "Edit" }).first();
    await expect(editBtn).toBeVisible();
  });

  test("clicking Edit opens inline form with warning/breach inputs", async ({ page }) => {
    await page.waitForTimeout(2000);
    await page.getByRole("button", { name: "Edit" }).first().click();
    // Labels have no htmlFor — match by adjacent number input
    const warningInput = page.locator("input[type='number']").first();
    await expect(warningInput).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText("Warning Hours")).toBeVisible();
    await expect(page.getByText("Breach Hours")).toBeVisible();
  });

  test("can save SLA policy changes", async ({ page }) => {
    await page.waitForTimeout(2000);
    await page.getByRole("button", { name: "Edit" }).first().click();
    // Labels have no htmlFor — use first number input (Warning Hours)
    const warningInput = page.locator("input[type='number']").first();
    await expect(warningInput).toBeVisible({ timeout: 5_000 });
    await warningInput.fill("2");
    await page.getByRole("button", { name: "Save" }).click();
    await page.waitForTimeout(1000);
    // Modal closes (Edit button re-appears)
    await expect(page.getByRole("button", { name: "Edit" }).first()).toBeVisible({ timeout: 5_000 });
  });
});
