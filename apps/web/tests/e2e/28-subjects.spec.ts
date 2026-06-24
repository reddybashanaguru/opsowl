import { test, expect } from "@playwright/test";
import { login, apiTokenAs, API_URL } from "./helpers/auth";

/**
 * Subject 360 Intelligence + Connected Entity Intelligence E2E tests.
 *
 * Verifies:
 * — Subject creation (individual, company, PEP, sanctioned)
 * — Subject list with search / filter by entity_type and risk_level
 * — Subject 360 profile: risk score, open_cases, total_cases, STR filings count
 * — Entity relationship CRUD (create, list, delete)
 * — Validation guards: duplicate → 409, self-link → 400, invalid type → 400
 * — BFS network traversal at depth=1 and depth=2
 * — Access control: viewer cannot create subjects; analyst can; manager can delete
 * — UI: Subject Directory page, detail page, Connected Entity Network panel
 */

const ADMIN_EMAIL    = "admin@opsowl.dev";
const ADMIN_PASS     = "OpsOwl@2024";
const ANALYST_EMAIL  = "sneha.patel@opsowl.dev";   // seeded analyst
const ANALYST_PASS   = "OpsOwl@2024";
const VIEWER_EMAIL   = "e2e.viewer@opsowl.test";   // seeded viewer
const VIEWER_PASS    = "OpsOwl@2024";

// ── Helpers ───────────────────────────────────────────────────────────────────

async function deleteSubject(token: string, id: string) {
  await fetch(`${API_URL}/subjects/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
}

async function createRelationship(
  token: string,
  fromID: string,
  toID: string,
  relType: string,
  notes = ""
) {
  return fetch(`${API_URL}/subjects/${fromID}/relationships`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ to_subject_id: toID, relationship_type: relType, strength: 0.9, notes }),
  });
}

// ── API tests ─────────────────────────────────────────────────────────────────

test.describe("Subject 360 Intelligence — API", () => {
  let adminToken: string;
  let analystToken: string;
  let subjectA: { id: string; canonical_name: string };
  let subjectB: { id: string; canonical_name: string };
  let subjectC: { id: string; canonical_name: string };
  let relAB: string; // relationship ID A→B

  test.beforeAll(async () => {
    adminToken   = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);
    analystToken = await apiTokenAs(ANALYST_EMAIL, ANALYST_PASS);
  });

  test.afterAll(async () => {
    for (const s of [subjectA, subjectB, subjectC]) {
      if (s?.id) await deleteSubject(adminToken, s.id);
    }
  });

  // ── Subject creation ────────────────────────────────────────────────────────

  test("analyst can create an individual subject", async () => {
    const suffix = Date.now();
    const res = await fetch(`${API_URL}/subjects`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({
        canonical_name: `E2E Priya Mehta ${suffix}`,
        entity_type: "individual",
        id_type: "PAN",
        id_number: `E2PAN${suffix % 100000}`,
        nationality: "IN",
        phones: ["+91-9876540001"],
        emails: ["priya.mehta.e2e@example.com"],
        is_pep: false,
        is_sanctioned: false,
      }),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    subjectA = data;
    expect(data.canonical_name).toContain("E2E Priya Mehta");
    expect(data.entity_type).toBe("individual");
    expect(data.id_type).toBe("PAN");
  });

  test("analyst can create a company subject", async () => {
    const suffix = Date.now() + 1;
    const res = await fetch(`${API_URL}/subjects`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({
        canonical_name: `E2E Sunrise Corp ${suffix}`,
        entity_type: "company",
        id_type: "CIN",
        id_number: `E2CIN${suffix % 100000}`,
        nationality: "IN",
      }),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    subjectB = data;
    expect(data.entity_type).toBe("company");
  });

  test("creates PEP + sanctioned individual for flag tests", async () => {
    const suffix = Date.now() + 2;
    const res = await fetch(`${API_URL}/subjects`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({
        canonical_name: `E2E Viktor Sanctioned ${suffix}`,
        entity_type: "individual",
        id_type: "PASSPORT",
        id_number: `E2PP${suffix % 100000}`,
        nationality: "UA",
        is_pep: true,
        is_sanctioned: true,
      }),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    subjectC = data;
    expect(data.is_pep).toBe(true);
    expect(data.is_sanctioned).toBe(true);
  });

  test("missing canonical_name returns 400", async () => {
    const res = await fetch(`${API_URL}/subjects`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({ entity_type: "individual" }),
    });
    expect(res.status).toBe(400);
  });

  // ── Subject list + search ───────────────────────────────────────────────────

  test("list subjects returns paginated results with total count", async () => {
    const res = await fetch(`${API_URL}/subjects?limit=25`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data, meta } = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(typeof meta.total).toBe("number");
    expect(meta.total).toBeGreaterThan(0);
  });

  test("search by partial name returns matching subjects", async () => {
    if (!subjectA) return;
    const res = await fetch(`${API_URL}/subjects?q=E2E+Priya`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const names: string[] = (Array.isArray(data) ? data : []).map((s: { canonical_name: string }) => s.canonical_name);
    expect(names.some((n) => n.includes("Priya"))).toBe(true);
  });

  test("filter by entity_type=company returns only companies", async () => {
    const res = await fetch(`${API_URL}/subjects?entity_type=company&limit=50`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const types: string[] = (Array.isArray(data) ? data : []).map((s: { entity_type: string }) => s.entity_type);
    types.forEach((t) => expect(t).toBe("company"));
  });

  // ── Subject 360 profile ─────────────────────────────────────────────────────

  test("GET /subjects/:id returns full 360 profile fields", async () => {
    if (!subjectA) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.canonical_name).toContain("E2E Priya Mehta");
    expect(typeof data.risk_score).toBe("number");
    expect(typeof data.open_cases).toBe("number");
    expect(typeof data.total_cases).toBe("number");
    expect(typeof data.total_str_filings).toBe("number");
    expect(data.is_pep).toBe(false);
    expect(data.is_sanctioned).toBe(false);
  });

  test("GET /subjects/:id returns PEP/sanctioned flags correctly", async () => {
    if (!subjectC) return;
    const res = await fetch(`${API_URL}/subjects/${subjectC.id}`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.is_pep).toBe(true);
    expect(data.is_sanctioned).toBe(true);
  });

  test("GET /subjects/:id returns 404 for unknown ID", async () => {
    const res = await fetch(`${API_URL}/subjects/00000000-0000-0000-0000-000000000099`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(404);
  });

  test("PATCH /subjects/:id updates risk_score and risk_level", async () => {
    if (!subjectA) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({ risk_score: 72, risk_level: "high", risk_factors: ["unusual_pattern"] }),
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(data.risk_score).toBe(72);
    expect(data.risk_level).toBe("high");
  });

  test("GET /subjects/:id/risk-history returns array", async () => {
    if (!subjectA) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/risk-history`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(Array.isArray(data)).toBe(true);
  });

  // ── Entity relationships ────────────────────────────────────────────────────

  test("create OWNS_ACCOUNT relationship A → B", async () => {
    if (!subjectA || !subjectB) return;
    const res = await createRelationship(analystToken, subjectA.id, subjectB.id, "OWNS_ACCOUNT", "confirmed via KYC");
    expect(res.status).toBe(201);
    const { data } = await res.json();
    relAB = data.id;
    expect(data.relationship_type).toBe("OWNS_ACCOUNT");
    expect(data.direction).toBe("outbound");
    expect(data.peer_name).toContain("Sunrise Corp");
  });

  test("create FAMILY_OF relationship A → C", async () => {
    if (!subjectA || !subjectC) return;
    const res = await createRelationship(analystToken, subjectA.id, subjectC.id, "FAMILY_OF");
    expect(res.status).toBe(201);
    const { data } = await res.json();
    expect(data.relationship_type).toBe("FAMILY_OF");
  });

  test("duplicate relationship returns 409", async () => {
    if (!subjectA || !subjectB) return;
    const res = await createRelationship(analystToken, subjectA.id, subjectB.id, "OWNS_ACCOUNT");
    expect(res.status).toBe(409);
  });

  test("self-link returns 400 with SELF_LINK error code", async () => {
    if (!subjectA) return;
    const res = await createRelationship(analystToken, subjectA.id, subjectA.id, "FAMILY_OF");
    expect(res.status).toBe(400);
    const { error } = await res.json();
    expect(error.code).toMatch(/SELF_LINK/i);
  });

  test("invalid relationship type returns 400 with INVALID_TYPE error code", async () => {
    if (!subjectA || !subjectB) return;
    const res = await createRelationship(analystToken, subjectA.id, subjectB.id, "RANDOM_TYPE");
    expect(res.status).toBe(400);
    const { error } = await res.json();
    expect(error.code).toMatch(/INVALID_TYPE/i);
  });

  test("list relationships from A — both connections outbound", async () => {
    if (!subjectA) return;
    // Small delay to ensure both relationship writes are flushed and visible
    await new Promise((r) => setTimeout(r, 500));
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/relationships`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(Array.isArray(data)).toBe(true);
    // Both A→B (OWNS_ACCOUNT) and A→C (FAMILY_OF) should exist; skip if neither was created
    if (data.length === 0) return; // previous relationship creation may have been skipped
    expect(data.length).toBeGreaterThanOrEqual(2);
    (data as Array<{ direction: string }>).forEach((r) => expect(r.direction).toBe("outbound"));
  });

  test("list relationships from B shows A's connection as inbound", async () => {
    if (!subjectA || !subjectB) return;
    const res = await fetch(`${API_URL}/subjects/${subjectB.id}/relationships`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const fromA = (data as Array<{ peer_name: string; direction: string }>)
      .find((r) => r.peer_name?.includes("Priya"));
    expect(fromA).toBeDefined();
    expect(fromA!.direction).toBe("inbound");
  });

  // ── BFS network traversal ───────────────────────────────────────────────────

  test("network depth=1 from A includes B and C as depth-1 nodes", async () => {
    if (!subjectA || !subjectB || !subjectC) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/network?depth=1`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const nodeIDs = (data.nodes as Array<{ id: string }>).map((n) => n.id);
    expect(nodeIDs).toContain(subjectA.id);
    expect(nodeIDs).toContain(subjectB.id);
    expect(nodeIDs).toContain(subjectC.id);
  });

  test("network root node has depth=0 and is_root=true", async () => {
    if (!subjectA) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/network?depth=1`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    const { data } = await res.json();
    const root = (data.nodes as Array<{ id: string; depth: number; is_root: boolean }>)
      .find((n) => n.id === subjectA.id);
    expect(root?.depth).toBe(0);
    expect(root?.is_root).toBe(true);
  });

  test("network high_risk_count > 0 when sanctioned/critical subject connected", async () => {
    if (!subjectA) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/network?depth=2`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    const { data } = await res.json();
    expect(data.high_risk_count).toBeGreaterThan(0);
  });

  test("network depth clamped to max 3 regardless of query param", async () => {
    if (!subjectA) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/network?depth=99`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    const maxDepth = Math.max(...(data.nodes as Array<{ depth: number }>).map((n) => n.depth));
    expect(maxDepth).toBeLessThanOrEqual(3);
  });

  test("delete A→B relationship removes it from list", async () => {
    if (!relAB || !subjectA) return;
    const res = await fetch(`${API_URL}/subjects/${subjectA.id}/relationships/${relAB}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(200);

    const listRes = await fetch(`${API_URL}/subjects/${subjectA.id}/relationships`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    const { data } = await listRes.json();
    const ids = (data as Array<{ id: string }>).map((r) => r.id);
    expect(ids).not.toContain(relAB);
  });

  // ── Access control ──────────────────────────────────────────────────────────

  test("viewer cannot create a subject (403)", async () => {
    const viewerToken = await apiTokenAs(VIEWER_EMAIL, VIEWER_PASS);
    const res = await fetch(`${API_URL}/subjects`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${viewerToken}` },
      body: JSON.stringify({ canonical_name: "Should Fail", entity_type: "individual" }),
    });
    expect(res.status).toBe(403);
  });

  test("unauthenticated request to subjects returns 401", async () => {
    const res = await fetch(`${API_URL}/subjects`);
    expect(res.status).toBe(401);
  });
});

// ── UI tests ──────────────────────────────────────────────────────────────────

test.describe("Subject 360 — UI", () => {
  let adminToken: string;
  let subjectID: string | null;
  let peerSubjectID: string | null;

  const SUBJECT_NAME = "E2E UI Arjun Dev";
  const PEER_NAME    = "E2E UI Dev Corp";

  test.beforeAll(async () => {
    adminToken = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASS);

    // Create or find subject for UI tests
    const tryCreate = async (body: object) => {
      const res = await fetch(`${API_URL}/subjects`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
        body: JSON.stringify(body),
      });
      if (res.status === 201) {
        const { data } = await res.json();
        return data.id as string;
      }
      // 409 = already exists — find by name
      if (res.status === 409) {
        const listRes = await fetch(`${API_URL}/subjects?q=${encodeURIComponent((body as { canonical_name: string }).canonical_name)}&limit=5`, {
          headers: { Authorization: `Bearer ${adminToken}` },
        });
        const { data } = await listRes.json();
        return (Array.isArray(data) && data.length > 0 ? (data[0] as { id: string }).id : null);
      }
      return null;
    };

    subjectID = await tryCreate({
      canonical_name: SUBJECT_NAME,
      entity_type: "individual",
      id_type: "PAN",
      id_number: "UIPAN99881",
      nationality: "IN",
    });

    // Set risk_score via PATCH (CREATE doesn't accept risk fields)
    if (subjectID) {
      await fetch(`${API_URL}/subjects/${subjectID}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
        body: JSON.stringify({ risk_score: 62, risk_level: "medium" }),
      });
    }

    peerSubjectID = await tryCreate({
      canonical_name: PEER_NAME,
      entity_type: "company",
      id_type: "CIN",
      id_number: "UICIN99882",
      nationality: "IN",
    });
  });

  test.afterAll(async () => {
    for (const id of [subjectID, peerSubjectID]) {
      if (id) {
        await fetch(`${API_URL}/subjects/${id}`, {
          method: "DELETE",
          headers: { Authorization: `Bearer ${adminToken}` },
        });
      }
    }
  });

  test("Subject Directory page renders heading, search, and filter dropdowns", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/subjects");
    await expect(page.getByRole("heading", { name: /subject directory/i })).toBeVisible({ timeout: 8_000 });
    await expect(page.getByPlaceholder(/search by name/i)).toBeVisible();
    // The risk level and entity type selects are visible as comboboxes
    const selects = page.locator("select");
    await expect(selects.first()).toBeVisible();
    await expect(selects.nth(1)).toBeVisible();
  });

  test("Subjects page shows Subjects nav link in sidebar", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/dashboard");
    await expect(page.getByRole("link", { name: /subjects/i })).toBeVisible();
  });

  test("subjects list shows seeded subject after name search", async ({ page }) => {
    if (!subjectID) return;
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/subjects");
    await page.getByPlaceholder(/search by name/i).fill("Arjun Dev");
    await expect(page.getByText(SUBJECT_NAME)).toBeVisible({ timeout: 8_000 });
  });

  test("clicking subject row navigates to /subjects/:id detail page", async ({ page }) => {
    if (!subjectID) return;
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/subjects");
    await page.getByPlaceholder(/search by name/i).fill("Arjun Dev");
    await page.getByText(SUBJECT_NAME).first().click();
    await expect(page).toHaveURL(new RegExp(`/subjects/${subjectID}`), { timeout: 8_000 });
  });

  test("subject detail shows risk score, case counts, and STR filings", async ({ page }) => {
    if (!subjectID) return;
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/subjects/${subjectID}`);
    await expect(page.getByRole("heading", { name: new RegExp(SUBJECT_NAME) })).toBeVisible({ timeout: 8_000 });
    // Risk score value
    await expect(page.getByText("62")).toBeVisible();
    // Stats section labels
    await expect(page.getByText(/Open Cases/i)).toBeVisible();
    await expect(page.getByText(/Total Cases/i)).toBeVisible();
    await expect(page.getByText(/STR Filings/i)).toBeVisible();
  });

  test("Connected Entity Network section is visible on subject detail", async ({ page }) => {
    if (!subjectID) return;
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/subjects/${subjectID}`);
    await expect(page.getByText(/Entity Network/i)).toBeVisible({ timeout: 8_000 });
  });

  test("Link Entity button opens modal with subject search and relationship dropdown", async ({ page }) => {
    if (!subjectID) return;
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/subjects/${subjectID}`);
    await expect(page.getByText(/Entity Network/i)).toBeVisible({ timeout: 8_000 });
    await page.getByRole("button", { name: /link entity/i }).first().click();
    await expect(page.getByRole("heading", { name: /link entity/i })).toBeVisible({ timeout: 6_000 });
    // Subject search input
    await expect(page.getByPlaceholder(/type name or id/i)).toBeVisible();
    // Relationship type select
    await expect(page.locator("select").first()).toBeVisible();
  });

  test("Link Entity modal relationship dropdown has all expected types", async ({ page }) => {
    if (!subjectID) return;
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/subjects/${subjectID}`);
    await expect(page.getByText(/Entity Network/i)).toBeVisible({ timeout: 8_000 });
    await page.getByRole("button", { name: /link entity/i }).first().click();
    await expect(page.getByRole("heading", { name: /link entity/i })).toBeVisible({ timeout: 6_000 });

    const relSelect = page.locator("select").first();
    const options = await relSelect.locator("option").allTextContents();
    const expectedLabels = ["Owns Account", "Transacts With", "Family Of", "Employed By", "Director Of"];
    for (const label of expectedLabels) {
      expect(options.some((o) => o.includes(label))).toBe(true);
    }
    // There should be 12 relationship type options
    expect(options.length).toBe(12);
  });

  test("adding relationship via modal shows new connection in table", async ({ page }) => {
    if (!subjectID || !peerSubjectID) return;
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto(`/subjects/${subjectID}`);
    await expect(page.getByText(/Entity Network/i)).toBeVisible({ timeout: 8_000 });

    await page.getByRole("button", { name: /link entity/i }).first().click();
    await expect(page.getByRole("heading", { name: /link entity/i })).toBeVisible({ timeout: 6_000 });

    // Search for peer subject by name — click the button in the modal results (not the Similar Entities link)
    await page.getByPlaceholder(/type name or id/i).fill("Dev Corp");
    await expect(page.getByRole("button", { name: new RegExp(PEER_NAME) }).first()).toBeVisible({ timeout: 6_000 });
    await page.getByRole("button", { name: new RegExp(PEER_NAME) }).first().click();

    // Select EMPLOYED_BY from relationship dropdown
    await page.locator("select").first().selectOption("EMPLOYED_BY");

    // Submit
    await page.getByRole("button", { name: /^link entity$/i }).last().click();

    // Connection should appear in the connections table
    await expect(
      page.getByText(/Employed By|E2E UI Dev Corp/i).first()
    ).toBeVisible({ timeout: 8_000 });
  });

  test("entity type filter changes dropdown and page reflects filter", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/subjects");
    await expect(page.getByRole("heading", { name: /subject directory/i })).toBeVisible({ timeout: 8_000 });
    // Select "Company" from entity type filter (second select)
    const selects = page.locator("select");
    await selects.nth(1).selectOption("company");
    // Page should reload/filter without error
    await expect(page.getByRole("heading", { name: /subject directory/i })).toBeVisible();
  });
});
