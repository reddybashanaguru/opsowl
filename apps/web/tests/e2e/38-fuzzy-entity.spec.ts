import { test, expect } from "@playwright/test";
import { login, apiToken, API_URL } from "./helpers/auth";

/**
 * QA suite for Feature 4: Fuzzy Entity Resolution
 *
 * Tests cover:
 *  — API: GET /subjects/:id/similar returns scored matches
 *  — API: phone overlap scoring
 *  — API: POST /subjects/:id/confirm-match creates SAME_ENTITY relationship
 *  — UI: Similar Entities panel visible on subject detail page
 *  — UI: Match score badges displayed
 *  — UI: Link button opens confirm options (Same Entity / Alias)
 *  — UI: Confirming a match updates the entity graph
 */

const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASS  = "OpsOwl@2024";

async function createSubject(token: string, name: string, phones: string[] = [], dob?: string) {
  const body: Record<string, unknown> = {
    canonical_name: name,
    entity_type: "individual",
    id_type: "PAN",
    id_number: `FUZZ${Date.now() % 99999}`,
    phones,
  };
  if (dob) body.date_of_birth = dob;

  const res = await fetch(`${API_URL}/subjects`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify(body),
  });
  const json = await res.json() as { data: { id: string; canonical_name: string } };
  return json.data;
}

async function deleteSubject(token: string, id: string) {
  await fetch(`${API_URL}/subjects/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
}

test.describe("Fuzzy Entity Resolution", () => {
  let token: string;
  // Two subjects with very similar names (trigram similarity should be high)
  let subjectA: { id: string; canonical_name: string };
  let subjectB: { id: string; canonical_name: string };
  // Two subjects with shared phone number
  let subjectC: { id: string; canonical_name: string };
  let subjectD: { id: string; canonical_name: string };

  const sharedPhone = `+91${Date.now().toString().slice(-10)}`;

  test.beforeAll(async () => {
    token = await apiToken();
    // Similar names: "Rajesh Kumar Sharma" vs "Rajesh K Sharma"
    const suffix = Date.now();
    subjectA = await createSubject(token, `Rajesh Kumar Sharma ${suffix}`);
    subjectB = await createSubject(token, `Rajesh K Sharma ${suffix}`);
    // Shared phone
    subjectC = await createSubject(token, `Priya Nair ${suffix}`, [sharedPhone]);
    subjectD = await createSubject(token, `Priya S Nair ${suffix}`, [sharedPhone]);
  });

  test.afterAll(async () => {
    await deleteSubject(token, subjectA.id);
    await deleteSubject(token, subjectB.id);
    await deleteSubject(token, subjectC.id);
    await deleteSubject(token, subjectD.id);
  });

  // ── API: similarity scoring ────────────────────────────────────────────────

  test("GET /subjects/:id/similar returns 200 with data array", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/similar`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const body = await res.json() as { data: unknown[] };
    expect(Array.isArray(body.data)).toBe(true);
  });

  test("similar name subject appears in results with name_similarity > 0", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/similar?threshold=0.2&limit=20`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: Array<{ id: string; name_similarity: number; total_score: number }> };
    const match = body.data.find(s => s.id === subjectB.id);
    expect(match).toBeDefined();
    expect(match!.name_similarity).toBeGreaterThan(0);
    expect(match!.total_score).toBeGreaterThan(0);
  });

  test("subject with shared phone shows phone_match=true and higher score", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectC.id}/similar?threshold=0.1&limit=20`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: Array<{ id: string; phone_match: boolean; total_score: number }> };
    const match = body.data.find(s => s.id === subjectD.id);
    expect(match).toBeDefined();
    expect(match!.phone_match).toBe(true);
    // Phone match adds 0.35 bonus — total_score should reflect this
    expect(match!.total_score).toBeGreaterThan(0.3);
  });

  test("match_reasons array is populated", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/similar?threshold=0.2&limit=20`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: Array<{ id: string; match_reasons: string[] }> };
    const match = body.data.find(s => s.id === subjectB.id);
    expect(match).toBeDefined();
    expect(Array.isArray(match!.match_reasons)).toBe(true);
    expect(match!.match_reasons.length).toBeGreaterThan(0);
  });

  test("self is never returned in similar results", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/similar?threshold=0.1&limit=50`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: Array<{ id: string }> };
    expect(body.data.find(s => s.id === subjectA.id)).toBeUndefined();
  });

  test("threshold parameter filters low-similarity results", async () => {
    // High threshold should return fewer results than low threshold
    const high = await fetch(`${API_URL}/subjects/${subjectA.id}/similar?threshold=0.9&limit=20`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const highBody = await high.json() as { data: unknown[] };

    const low = await fetch(`${API_URL}/subjects/${subjectA.id}/similar?threshold=0.1&limit=20`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const lowBody = await low.json() as { data: unknown[] };

    // Low threshold always returns >= high threshold count
    expect(lowBody.data.length).toBeGreaterThanOrEqual(highBody.data.length);
  });

  // ── API: confirm match ─────────────────────────────────────────────────────

  test("POST /subjects/:id/confirm-match creates SAME_ENTITY relationship", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/confirm-match`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        target_subject_id: subjectB.id,
        relationship_type: "SAME_ENTITY",
        notes: "E2E fuzzy resolution test",
      }),
    });
    expect(res.status).toBe(201);
  });

  test("POST confirm-match with ALIAS_OF type succeeds", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectC.id}/confirm-match`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        target_subject_id: subjectD.id,
        relationship_type: "ALIAS_OF",
        notes: "Phone-matched alias",
      }),
    });
    expect(res.status).toBe(201);
  });

  test("POST confirm-match with self returns 400", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/confirm-match`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        target_subject_id: subjectA.id,
        relationship_type: "SAME_ENTITY",
      }),
    });
    expect(res.status).toBe(400);
  });

  test("POST confirm-match with invalid relationship_type returns 400", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/confirm-match`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        target_subject_id: subjectB.id,
        relationship_type: "INVALID_TYPE",
      }),
    });
    expect(res.status).toBe(400);
  });

  test("SAME_ENTITY relationship is reflected in entity graph", async () => {
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/relationships`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: { relationships: Array<{ relationship_type: string }> } };
    const rels = body.data?.relationships ?? [];
    const rel = rels.find((r: { relationship_type: string }) => r.relationship_type === "SAME_ENTITY");
    expect(rel).toBeDefined();
  });

  // ── UI tests ───────────────────────────────────────────────────────────────

  test("Similar Entities panel is visible on subject detail page", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectA.id}`);
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText("fuzzy name + phone match")).toBeVisible();
  });

  test("similar name subject appears in the panel", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectA.id}`);
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    // SubjectB has a very similar name — should appear
    await expect(page.getByText(subjectB.canonical_name)).toBeVisible({ timeout: 8_000 });
  });

  test("match score badge is shown for each similar entity", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectA.id}`);
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    // Score badges show "X% match"
    await expect(page.getByText(/% match/).first()).toBeVisible({ timeout: 8_000 });
  });

  test("match reason tags are displayed", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectA.id}`);
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    // At least one reason tag should appear (e.g. "Similar name", "Near-identical name")
    await expect(
      page.getByText(/Similar name|Near-identical name|Partial name match|Shared phone/).first()
    ).toBeVisible({ timeout: 8_000 });
  });

  test("Link button is present for each similar entity", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectC.id}`);
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByRole("button", { name: "Link", exact: true }).first()).toBeVisible({ timeout: 8_000 });
  });

  test("clicking Link button shows Same Entity and Alias options", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectA.id}`);
    // Wait for panel and similar subject to load before clicking
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText(subjectB.canonical_name)).toBeVisible({ timeout: 8_000 });
    // Use exact:true to avoid matching "Link Entity" button in the entity network section
    await page.getByRole("button", { name: "Link", exact: true }).first().click();
    // Confirm options appear as text inside buttons (SVG icon + text)
    await expect(page.getByText("Same Entity")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText("Alias")).toBeVisible({ timeout: 3_000 });
  });

  test("Cancel button dismisses the confirm options", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectA.id}`);
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText(subjectB.canonical_name)).toBeVisible({ timeout: 8_000 });
    await page.getByRole("button", { name: "Link", exact: true }).first().click();
    await expect(page.getByText("Same Entity")).toBeVisible({ timeout: 5_000 });
    await page.getByRole("button", { name: "Cancel" }).first().click();
    await expect(page.getByText("Same Entity")).not.toBeVisible({ timeout: 2_000 });
  });

  test("shared phone subjects have Shared phone number reason", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/subjects/${subjectC.id}`);
    await expect(page.getByText("Similar Entities")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText("Shared phone number")).toBeVisible({ timeout: 8_000 });
  });
});
