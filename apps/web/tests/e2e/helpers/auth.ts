import { Page } from "@playwright/test";

export const ADMIN = { email: "admin@opsowl.dev", password: "OpsOwl@2024" };
export const BASE_URL = "http://localhost:3001";
export const API_URL = "http://localhost:8080/api/v1";

/** Full UI login flow — use this per-spec in beforeAll. */
export async function login(page: Page, email = ADMIN.email, password = ADMIN.password) {
  await page.goto("/login");
  await page.getByLabel("Email address").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForURL("**/dashboard", { timeout: 15_000 });
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
