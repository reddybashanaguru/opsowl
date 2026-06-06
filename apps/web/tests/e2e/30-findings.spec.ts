import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase, API_URL } from "./helpers/auth";

let token: string;
let caseId: string;
let findingId: string;

test.describe("Findings & Evidence Linking", () => {
  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "Findings E2E Test Case");
    caseId = c.id;

    // Create a task and complete it so the closure checklist passes later
    const taskRes = await fetch(`${API_URL}/cases/${caseId}/tasks`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ title: "Investigate transaction", priority: "medium" }),
    });
    const task = (await taskRes.json()).data;
    await fetch(`${API_URL}/cases/${caseId}/tasks/${task.id}/status`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ status: "done" }),
    });
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  test("findings tab is visible in case subnav", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}`);
    await expect(page.getByRole("button", { name: /Findings/i })).toBeVisible();
  });

  test("findings page renders empty state", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/findings`);
    await expect(page.getByText(/No formal findings yet|Create First Finding/i).first()).toBeVisible();
  });

  test("API: create a finding", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/findings`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        title: "Unusual Cash Deposits",
        description: "Subject made 12 cash deposits of ₹90,000 each in a 30-day period, just below the ₹1L reporting threshold.",
        category: "structuring",
        severity: "high",
        narrative_weight: "primary",
      }),
    });
    expect(res.status).toBe(201);
    const json = await res.json();
    findingId = json.data.id;
    expect(findingId).toBeTruthy();
    expect(json.data.status).toBe("draft");
  });

  test("API: finding appears in list", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/findings`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThan(0);
    expect(json.data[0].title).toMatch(/Unusual Cash/i);
  });

  test("API: findings summary includes high count", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/findings/summary`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.total).toBeGreaterThanOrEqual(1);
    expect(json.data.high).toBeGreaterThanOrEqual(1);
  });

  test("API: link evidence to finding", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/findings/${findingId}/evidence`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        evidence_type: "note",
        evidence_id: caseId, // reuse any UUID for test
        citation: "Subject confirmed deposits during interview",
        relevance_note: "Admission of structuring behaviour",
      }),
    });
    expect(res.status).toBe(201);
  });

  test("API: submit finding for review", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/findings/${findingId}/submit`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: "{}",
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.status).toBe("under_review");
  });

  test("API: approve finding", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/findings/${findingId}/approve`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: "{}",
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.status).toBe("approved");
  });

  test("findings page renders finding cards", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/findings`);
    await page.waitForTimeout(1500);
    await expect(page.getByText(/Unusual Cash/i)).toBeVisible();
  });

  test("findings summary stats visible", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/findings`);
    await page.waitForTimeout(1500);
    // Approved count should show at least 1
    await expect(page.getByText(/Approved/i).first()).toBeVisible();
  });

  test("finding card expands on click", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/findings`);
    await page.waitForTimeout(1500);
    const card = page.getByText(/Unusual Cash/i).first();
    await card.click();
    await page.waitForTimeout(500);
    // Evidence section should appear after expand
    await expect(page.getByText(/Evidence|evidence/i).first()).toBeVisible();
  });
});
