import { Page } from "@playwright/test";
import fs from "fs";
import path from "path";

export const ADMIN = { email: "admin@opsowl.dev", password: "OpsOwl@2024" };
export const BASE_URL = "http://localhost:3001";
export const API_URL = "http://localhost:8080/api/v1";

const AUTH_FILE = path.join(__dirname, "../.auth/admin.json");

interface StoredAuth {
  cookies: Array<{ name: string; value: string; domain: string; path: string; expires: number; httpOnly: boolean; secure: boolean; sameSite: string }>;
  user?: { id: string; email: string; full_name: string; role: string; last_login_at?: string };
}

function loadAuthState(): StoredAuth | null {
  if (!fs.existsSync(AUTH_FILE)) return null;
  try {
    const state: StoredAuth = JSON.parse(fs.readFileSync(AUTH_FILE, "utf-8"));
    const accessToken = state.cookies?.find((c) => c.name === "access_token");
    if (!accessToken || accessToken.expires * 1000 <= Date.now() + 60_000) return null;
    return state;
  } catch {
    return null;
  }
}

/** Login helper.
 *
 * Fast path (valid admin.json): injects the access_token cookie AND pre-populates
 * sessionStorage with the user object via addInitScript. ProtectedRoute sees the
 * user immediately and renders without calling /auth/refresh — no token rotation,
 * no rate-limit hits, no race conditions.
 *
 * Slow path (expired tokens): falls back to the full UI login form.
 */
export async function login(
  page: Page,
  email = ADMIN.email,
  password = ADMIN.password,
  targetUrl?: string,
) {
  const auth = loadAuthState();
  const destination = targetUrl || "/dashboard";

  if (auth?.user) {
    // Pre-populate sessionStorage before the page loads so ProtectedRoute skips
    // the /auth/refresh call entirely — avoids token rotation conflicts across tests.
    const user = auth.user;
    await page.addInitScript((u) => {
      sessionStorage.setItem("opsowl_user", JSON.stringify(u));
    }, user);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await page.context().addCookies(auth.cookies as any[]);
    await page.goto(destination);
    await page.waitForSelector("aside", { timeout: 15_000 });
    return;
  }

  // Slow path: full UI login (only needed when tokens have expired).
  const loginUrl = targetUrl
    ? `/login?returnUrl=${encodeURIComponent(targetUrl)}`
    : "/login";
  await page.goto(loginUrl);
  await page.getByLabel("Email address").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForURL(targetUrl ? `**${targetUrl}` : "**/dashboard", { timeout: 15_000 });
  await page.waitForSelector("aside", { timeout: 15_000 });
}

/** Direct API helper — returns access_token for seeding data without browser. */
export async function apiToken(): Promise<string> {
  const res = await fetch(`${API_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(ADMIN),
  });
  const json = await res.json() as { data: { access_token: string } };
  return json.data.access_token;
}

/** Returns an access_token for any email/password pair. Useful for role-specific API tests. */
export async function apiTokenAs(email: string, password: string): Promise<string> {
  const res = await fetch(`${API_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const json = await res.json() as { data: { access_token: string } };
  return json.data.access_token;
}

/** Create a case via API and return its id. */
export async function createCase(
  token: string,
  title: string,
  priority = "medium"
): Promise<{ id: string; case_number: string }> {
  const res = await fetch(`${API_URL}/cases`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ title, description: "E2E test case", priority, tags: ["e2e"] }),
  });
  const json = await res.json() as { data: { id: string; case_number: string } };
  return json.data;
}

/** Delete a case via API. */
export async function deleteCase(token: string, id: string) {
  await fetch(`${API_URL}/cases/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
}
