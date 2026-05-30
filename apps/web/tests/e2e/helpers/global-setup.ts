import { chromium, request as apiRequest } from "@playwright/test";
import path from "path";
import fs from "fs";

const AUTH_FILE = path.join(__dirname, "../.auth/admin.json");

interface AuthState {
  cookies: Array<{ name: string; value: string; domain: string; path: string; expires: number; httpOnly: boolean; secure: boolean; sameSite: string }>;
  origins: unknown[];
  user?: { id: string; email: string; full_name: string; role: string; last_login_at?: string };
}

function isAuthFileValid(): boolean {
  if (!fs.existsSync(AUTH_FILE)) return false;
  try {
    const state: AuthState = JSON.parse(fs.readFileSync(AUTH_FILE, "utf-8"));
    const accessToken = state.cookies?.find((c) => c.name === "access_token");
    // Valid if access_token expires more than 60 seconds from now AND user data exists
    return !!(accessToken && accessToken.expires * 1000 > Date.now() + 60_000 && state.user);
  } catch {
    return false;
  }
}

export default async function globalSetup() {
  // Skip login if existing auth file has a valid access_token with user data.
  if (isAuthFileValid()) {
    console.log("[global-setup] Reusing existing valid auth state.");
    return;
  }

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
  await page.waitForURL("**/dashboard", { timeout: 30_000 });

  // Save cookies
  const storageState = await page.context().storageState();

  // Fetch the full user profile (JWT has no full_name)
  const apiCtx = await apiRequest.newContext({ baseURL: "http://localhost:8080" });
  const meRes = await apiCtx.get("/api/v1/auth/me", {
    headers: { Cookie: storageState.cookies.map((c) => `${c.name}=${c.value}`).join("; ") },
  });
  const meJson = await meRes.json() as { data: { id: string; email: string; full_name: string; role: string; last_login_at?: string } };
  await apiCtx.dispose();

  // Write enriched auth file with user data for sessionStorage pre-population
  const enriched: AuthState = { ...storageState, user: meJson.data };
  fs.writeFileSync(AUTH_FILE, JSON.stringify(enriched, null, 2));

  await browser.close();
}
