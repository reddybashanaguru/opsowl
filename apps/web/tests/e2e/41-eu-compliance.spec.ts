import { test, expect } from "@playwright/test";
import { login, apiToken, apiTokenAs, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * EU Compliance E2E tests — Lithuania (FNTT) and Cyprus (MOKAS)
 *
 * Verifies:
 * - EU goAML STR filing with LT jurisdiction (EUR, IBAN, FNTT entity)
 * - EU goAML STR filing with CY jurisdiction (EUR, IBAN, MOKAS entity)
 * - goAML validation passes for complete EU filing
 * - goAML validation fails for missing FNTT/MOKAS rentity_id
 * - EU NATIONAL_ID accepted (not PAN/Aadhaar)
 * - IBAN account numbers accepted in subject and transactions
 * - Sanctions settings page loads (EU Consolidated + UN + OFAC)
 * - AI Act compliance page loads — conformity checklist visible
 * - GDPR settings page loads — DSAR queue visible
 * - UBO tab visible on case detail (when config.ubo_enabled=true)
 * - Maker-checker workflow works for EU filings
 * - Analyst cannot create EU STR (access control unchanged)
 */

const MANAGER_EMAIL = "ravi.kumar@opsowl.dev";
const MANAGER_PASSWORD = "OpsOwl@2024";
const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASSWORD = "OpsOwl@2024";
const ANALYST_EMAIL = "divya.reddy@opsowl.dev";
const ANALYST_PASSWORD = "OpsOwl@2024";

const EU_ENTITY_LT = {
  name: "Baltic EMI UAB",
  rentity_id: "FNTT-LT-EMI-0042",
  rentity_branch: "VILNIUS-HQ-001",
  swift_code: "BEMIUALT",
  submission_code: "E",
  officer_first_name: "Mantas",
  officer_last_name: "Žukauskas",
  officer_gender: "M",
  officer_title: "Mr",
  officer_id_number: "38901011234",
  officer_id_type: "NATIONAL_ID",
  officer_occupation: "MLRO",
  officer_phone: "+370-5-123-4567",
  officer_email: "mantas@balticemi.lt",
  entity_address: "Gedimino pr. 1",
  entity_city: "Vilnius",
  entity_state: "Vilnius County",
  entity_zip: "LT-01103",
  entity_country: "LT",
  fiu_name: "FNTT",
  local_currency: "EUR",
};

const EU_ENTITY_CY = {
  name: "CY Compliance Services Ltd",
  rentity_id: "MOKAS-CY-FX-0099",
  rentity_branch: "NICOSIA-HQ-001",
  swift_code: "CYCSLCYX",
  submission_code: "E",
  officer_first_name: "Andreas",
  officer_last_name: "Papadopoulos",
  officer_gender: "M",
  officer_title: "Mr",
  officer_id_number: "K123456",
  officer_id_type: "NATIONAL_ID",
  officer_occupation: "Chief Compliance Officer",
  officer_phone: "+357-22-123-456",
  officer_email: "andreas@cycompliance.com.cy",
  entity_address: "Arch. Makariou III 123",
  entity_city: "Nicosia",
  entity_state: "Nicosia District",
  entity_zip: "1065",
  entity_country: "CY",
  fiu_name: "MOKAS",
  local_currency: "EUR",
};

function euFilingBody(entityOverrides = {}, subjectOverrides = {}, txOverrides = {}) {
  return {
    report_type: "STR",
    suspicion_basis: "Suspicious SEPA Instant cluster — structuring below €10,000 EDD threshold",
    narrative: "EU AMLR 2024 — 23 transactions, €85,000, velocity 18x above account baseline.",
    evidence_summary: "SEPA export, CDD file, STR draft",
    filing_entity: { name: "Baltic EMI UAB", ...entityOverrides },
    subject: {
      first_name: "Dmitri",
      last_name: "Volkov",
      name: "Dmitri Volkov",
      id_type: "NATIONAL_ID",
      id_number: "38001011234",
      account_number: "LT12 3456 7890 1234 5678",
      ...subjectOverrides,
    },
    transactions: [
      {
        date: "2026-06-10",
        amount: 9800,
        currency: "EUR",
        description: "SEPA Instant — beneficiary LVXX",
        transmode_code: "T",
        from_institution: "Baltic EMI UAB",
        from_swift: "BEMIUALT",
        from_account_number: "LT12 3456 7890 1234 5678",
        ...txOverrides,
      },
    ],
  };
}

test.describe("EU compliance — STR filings (Lithuania / Cyprus)", () => {
  let token: string;
  let caseId: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E EU Compliance STR Test", "high");
    caseId = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  // ── Access control ─────────────────────────────────────────────────────────

  test("analyst cannot create EU STR filing", async () => {
    const analystToken = await apiTokenAs(ANALYST_EMAIL, ANALYST_PASSWORD);
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify(euFilingBody(EU_ENTITY_LT)),
    });
    expect(res.status).toBe(403);
  });

  // ── Lithuania / FNTT filing ────────────────────────────────────────────────

  test("LT jurisdiction filing created — EUR, IBAN, FNTT entity", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(euFilingBody(EU_ENTITY_LT)),
    });
    expect(res.status).toBe(201);
    const { data: draft } = await res.json();
    expect(draft.filing_entity.entity_country).toBe("LT");
    expect(draft.filing_entity.fiu_name).toBe("FNTT");
    expect(draft.filing_entity.local_currency).toBe("EUR");
    expect(draft.filing_entity.rentity_id).toBe("FNTT-LT-EMI-0042");
    expect(draft.subject.id_type).toBe("NATIONAL_ID");
    expect(draft.subject.account_number).toContain("LT12");
  });

  test("LT filing — goAML validation passes with complete FNTT fields", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(euFilingBody(EU_ENTITY_LT)),
    });
    const { data: draft } = await draftRes.json();

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    expect(valRes.status).toBe(200);
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
    expect((val.errors ?? []).length).toBe(0);
  });

  test("LT filing — goAML validation fails when FNTT rentity_id missing", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const incompleteEntity = { ...EU_ENTITY_LT };
    delete (incompleteEntity as Record<string, unknown>).rentity_id;

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(euFilingBody(incompleteEntity, {}, {})),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(false);
    const fields = val.errors.map((e: { field: string }) => e.field);
    expect(fields).toContain("filing_entity.rentity_id");
  });

  test("LT filing — goAML validation fails for empty transactions", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const body = { ...euFilingBody(EU_ENTITY_LT), transactions: [] };
    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(body),
    });
    const { data: draft } = await draftRes.json();

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(false);
    const fields = val.errors.map((e: { field: string }) => e.field);
    expect(fields).toContain("transactions");
  });

  // ── Cyprus / MOKAS filing ──────────────────────────────────────────────────

  test("CY jurisdiction filing created — EUR, IBAN, MOKAS entity", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(
        euFilingBody(
          EU_ENTITY_CY,
          {
            first_name: "Costas",
            last_name: "Georgiou",
            name: "Costas Georgiou",
            id_type: "NATIONAL_ID",
            id_number: "A123456",
            account_number: "CY17 0020 0128 0000 0012 0052 7600",
          },
          {
            currency: "EUR",
            from_institution: "CY Compliance Services Ltd",
            from_swift: "CYCSLCYX",
            from_account_number: "CY17 0020 0128 0000 0012 0052 7600",
          }
        )
      ),
    });
    expect(res.status).toBe(201);
    const { data: draft } = await res.json();
    expect(draft.filing_entity.entity_country).toBe("CY");
    expect(draft.filing_entity.fiu_name).toBe("MOKAS");
    expect(draft.filing_entity.local_currency).toBe("EUR");
    expect(draft.subject.account_number).toContain("CY17");
  });

  test("CY filing — goAML validation passes with complete MOKAS fields", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(
        euFilingBody(
          EU_ENTITY_CY,
          {
            first_name: "Costas",
            last_name: "Georgiou",
            name: "Costas Georgiou",
            id_type: "NATIONAL_ID",
            id_number: "A123456",
            account_number: "CY17 0020 0128 0000 0012 0052 7600",
          },
          {
            currency: "EUR",
            from_institution: "CY Compliance Services Ltd",
            from_swift: "CYCSLCYX",
            from_account_number: "CY17 0020 0128 0000 0012 0052 7600",
          }
        )
      ),
    });
    const { data: draft } = await draftRes.json();

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
  });

  // ── EU-specific field assertions ───────────────────────────────────────────

  test("EU filing accepts NATIONAL_ID as officer id_type (not PAN)", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(euFilingBody({
        ...EU_ENTITY_LT,
        officer_id_type: "NATIONAL_ID",
        officer_id_number: "38901011234",
      })),
    });
    expect(res.status).toBe(201);
    const { data: draft } = await res.json();
    expect(draft.filing_entity.officer_id_type).toBe("NATIONAL_ID");
  });

  test("EU filing accepts PASSPORT as subject id_type", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);
    const res = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(euFilingBody(
        EU_ENTITY_LT,
        { id_type: "PASSPORT", id_number: "P1234567LT" }
      )),
    });
    expect(res.status).toBe(201);
    const { data: draft } = await res.json();
    expect(draft.subject.id_type).toBe("PASSPORT");
  });

  test("EU SEPA transaction with EUR currency passes validation", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);
    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(euFilingBody(EU_ENTITY_LT, {}, { currency: "EUR", amount: 9800 })),
    });
    const { data: draft } = await draftRes.json();

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
  });

  test("maker-checker error message does not mention PMLA (EU context)", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(euFilingBody(EU_ENTITY_LT)),
    });
    const { data: draft } = await draftRes.json();

    await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/submit`, {
      method: "POST",
      headers: { Authorization: `Bearer ${mgr}` },
    });

    const approveRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/approve`, {
      method: "POST",
      headers: { Authorization: `Bearer ${mgr}` },
    });
    expect(approveRes.status).toBe(403);
    const body = await approveRes.json();
    const errorMsg: string = body?.error?.message ?? "";
    expect(errorMsg.toLowerCase()).not.toContain("pmla");
    expect(errorMsg.toLowerCase()).toContain("aml");
  });
});

// ── EU Settings pages ──────────────────────────────────────────────────────────

test.describe("EU compliance — settings pages", () => {
  test("Sanctions settings page loads — shows 3 list cards", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto("/settings/sanctions");

    await expect(page.getByText("EU Consolidated List")).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText("UN Security Council List")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText("OFAC Specially Designated Nationals")).toBeVisible({ timeout: 5_000 });
  });

  test("Sanctions page shows correct jurisdictions for all 3 lists", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto("/settings/sanctions");

    await expect(page.getByText("European Union")).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText("United Nations")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText("United States Treasury")).toBeVisible({ timeout: 5_000 });
  });

  test("AI Act compliance page loads — conformity checklist visible", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto("/settings/ai-act");

    await expect(page.getByText(/conformity checklist/i)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/EU AI Act/i)).toBeVisible({ timeout: 5_000 });
  });

  test("AI Act page shows pending review queue section", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto("/settings/ai-act");

    await expect(page.getByText(/pending review/i)).toBeVisible({ timeout: 10_000 });
  });

  test("GDPR settings page loads — DSAR queue visible", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto("/settings/gdpr");

    await expect(
      page.getByText(/DSAR|erasure request|data subject/i).first()
    ).toBeVisible({ timeout: 10_000 });
  });

  test("GDPR page shows log request button", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto("/settings/gdpr");

    await expect(
      page.getByRole("button", { name: /log.*request|new.*request/i })
    ).toBeVisible({ timeout: 10_000 });
  });

  test("analyst cannot access sanctions reload (admin-only)", async ({ page }) => {
    await login(page, ANALYST_EMAIL, ANALYST_PASSWORD);
    await page.goto("/settings/sanctions");

    // Reload buttons must not be visible to analyst
    const reloadBtns = page.getByRole("button", { name: /reload/i });
    await expect(reloadBtns.first()).not.toBeVisible({ timeout: 5_000 });
  });
});

// ── EU UBO tab ─────────────────────────────────────────────────────────────────

test.describe("EU compliance — UBO verification tab", () => {
  let token: string;
  let caseId: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E UBO EU Test", "high");
    caseId = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  test("UBO tab is accessible on case detail page", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto(`/cases/${caseId}`);

    // UBO tab should be present in the subnav
    await expect(
      page.getByRole("button", { name: /^UBO$/i })
        .or(page.getByRole("link", { name: /^UBO$/i }))
    ).toBeVisible({ timeout: 10_000 });
  });

  test("UBO tab shows add record form or empty state", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto(`/cases/${caseId}/ubo`);

    await expect(
      page.getByText(/beneficial owner|UBO|no.*UBO|add.*UBO/i).first()
    ).toBeVisible({ timeout: 10_000 });
  });

  test("UBO API returns 200 for authenticated user", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);
    const res = await fetch(`${API_URL}/cases/${caseId}/ubo`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    expect(res.status).toBe(200);
    const { data } = await res.json();
    expect(Array.isArray(data)).toBe(true);
  });

  test("UBO record can be created with AMLR 2024 fields", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);
    const res = await fetch(`${API_URL}/cases/${caseId}/ubo`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify({
        entity_name: "Baltic Holdings BV",
        owner_name: "Dmitri Volkov",
        owner_nationality: "LT",
        owner_id_type: "NATIONAL_ID",
        owner_id_number: "38001011234",
        owner_birth_date: "1978-03-15",
        ownership_pct: 34.5,
        control_type: "share",
        registry_url: "https://rekvizitai.vz.lt/",
      }),
    });
    expect(res.status).toBe(201);
    const { data } = await res.json();
    expect(data.owner_name).toBe("Dmitri Volkov");
    expect(data.ownership_pct).toBe(34.5);
    expect(data.status).toBe("pending");
  });
});
