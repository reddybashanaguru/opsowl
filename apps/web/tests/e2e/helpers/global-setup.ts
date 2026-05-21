import { chromium } from "@playwright/test";
import path from "path";
import fs from "fs";

const AUTH_FILE = path.join(__dirname, "../.auth/admin.json");

export default async function globalSetup() {
  // Ensure auth directory exists
  const authDir = path.dirname(AUTH_FILE);
  if (!fs.existsSync(authDir)) {
    fs.mkdirSync(authDir, { recursive: true });
  }

  const browser = await chromium.launch();
  const page = await browser.newPage();

  await page.goto("http://localhost:3001/login");
  await page.getByLabel("Email address").fill("admin@opsowl.dev");
  await page.getByLabel("Password").fill("OpsOwl@2024");
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForURL("**/dashboard", { timeout: 15_000 });

  // Capture cookies (refresh token httpOnly cookie if any)
  await page.context().storageState({ path: AUTH_FILE });
  await browser.close();
}
