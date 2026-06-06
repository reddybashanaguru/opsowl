import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase, API_URL } from "./helpers/auth";

let token: string;
let caseId: string;
let narrativeId: string;

test.describe("AI Investigation Narrative", () => {
  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "Narrative E2E Test Case");
    caseId = c.id;

    // Create and approve a finding so the AI has evidence to work with
    const fRes = await fetch(`${API_URL}/cases/${caseId}/findings`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        title: "Structured Cash Deposits",
        description: "Subject made 15 cash deposits of ₹89,000 each over 45 days — classic structuring pattern below the ₹1L reporting threshold.",
        category: "structuring",
        severity: "medium",
        narrative_weight: "primary",
      }),
    });
    const finding = (await fRes.json()).data;

    // Submit + approve
    await fetch(`${API_URL}/cases/${caseId}/findings/${finding.id}/submit`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: "{}",
    });
    await fetch(`${API_URL}/cases/${caseId}/findings/${finding.id}/approve`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: "{}",
    });
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  test("narrative tab is visible in case subnav", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}`);
    await expect(page.getByRole("button", { name: /Narrative/i })).toBeVisible();
  });

  test("narrative page renders empty state with CTA", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/narrative`);
    await expect(page.getByText(/No narratives yet/i)).toBeVisible();
    await expect(page.getByRole("button", { name: /Generate.*narrative/i }).first()).toBeVisible();
  });

  test("API: generate investigation_summary narrative", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/narratives/generate`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        narrative_type: "investigation_summary",
        investigator_notes: "Focus on the structuring pattern",
      }),
    });
    expect(res.status).toBe(201);
    const json = await res.json();
    narrativeId = json.data.id;
    expect(narrativeId).toBeTruthy();
    expect(json.data.status).toBe("draft");
    expect(json.data.body.length).toBeGreaterThan(100);
    expect(json.data.findings_included).toBeGreaterThanOrEqual(1);
    expect(json.data.model_used).toBeTruthy();
  });

  test("API: ai_log_id is set on generated narrative", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/narratives/${narrativeId}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const json = await res.json();
    // ai_log_id links narrative to the full AI audit trail
    expect(json.data.ai_log_id).toBeTruthy();
  });

  test("API: list narratives returns generated narrative", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/narratives`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.length).toBe(1);
    expect(json.data[0].narrative_type).toBe("investigation_summary");
  });

  test("API: edit narrative body", async () => {
    const editedBody = "This is an edited investigation narrative with sufficient detail about the structuring pattern observed in this case. The subject made multiple cash deposits below the reporting threshold over a 45-day period.";
    const res = await fetch(`${API_URL}/cases/${caseId}/narratives/${narrativeId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ body: editedBody }),
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.body_edited).toBe(editedBody);
    expect(json.data.active_body).toBe(editedBody);
  });

  test("API: approve narrative", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/narratives/${narrativeId}/approve`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: "{}",
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.status).toBe("approved");
    expect(json.data.reviewed_by).toBeTruthy();
    expect(json.data.reviewed_at).toBeTruthy();
  });

  test("API: cannot edit approved narrative", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/narratives/${narrativeId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ body: "Trying to edit approved narrative" }),
    });
    expect(res.status).toBe(409); // ALREADY_APPROVED
  });

  test("API: generate STR suspicion basis narrative", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/narratives/generate`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ narrative_type: "str_suspicion_basis" }),
    });
    expect(res.status).toBe(201);
    const json = await res.json();
    expect(json.data.narrative_type).toBe("str_suspicion_basis");
    expect(json.data.body.length).toBeGreaterThan(50);
  });

  test("narrative page shows narrative card after generation", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/narrative`);
    await page.waitForTimeout(1500);
    // At least one narrative card should be visible
    await expect(page.getByText(/Investigation Summary|STR Suspicion/i).first()).toBeVisible();
  });

  test("narrative page shows approved badge", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/narrative`);
    await page.waitForTimeout(1500);
    await expect(page.getByText(/approved/i).first()).toBeVisible();
  });

  test("narrative card shows word count and model used", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/narrative`);
    await page.waitForTimeout(1500);
    await expect(page.getByText(/words/i).first()).toBeVisible();
    await expect(page.getByText(/template|claude|gpt|ollama/i).first()).toBeVisible();
  });
});
