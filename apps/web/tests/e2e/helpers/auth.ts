import { Page } from "@playwright/test";

export const ADMIN = { email: "admin@opsowl.dev", password: "OpsOwl@2024" };
export const BASE_URL = "http://localhost:3001";
export const API_URL = "http://localhost:8080/api/v1";

/** Full UI login flow — use this per-spec in beforeAll/beforeEach.
 *
 * Pass `targetUrl` (e.g. "/settings/webhooks") to land directly on that page
 * after login. This avoids the double-refresh race where ProtectedRoute on
 * /dashboard rotates the refresh token before the subsequent page.goto() can
 * receive the new httpOnly cookie.
 *
 * After the URL resolves we wait for the sidebar <aside> element — this is
 * rendered by DashboardLayout only after ProtectedRoute's /auth/refresh call
 * completes and sets checking=false, so it reliably signals that the full auth
 * handshake is done before any test assertions run.
 */
export async function login(
  page: Page,
  email = ADMIN.email,
  password = ADMIN.password,
  targetUrl?: string,
) {
  const loginUrl = targetUrl
    ? `/login?returnUrl=${encodeURIComponent(targetUrl)}`
    : "/login";
  await page.goto(loginUrl);
  await page.getByLabel("Email address").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  // Wait for URL then for the sidebar — sidebar only renders after ProtectedRoute
  // clears its checking state, ensuring the /auth/refresh rotation is complete.
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
