import { test, expect } from "@playwright/test";
import { login, apiToken, apiTokenAs, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * goAML 4.0 STR filing E2E tests — global edition.
 *
 * Verifies:
 * - STR filing creation with full goAML fields (India and non-India jurisdictions)
 * - goAML readiness validation endpoint
 * - Export goAML XML button availability after approval
 * - Analyst cannot see the STR tab (access restriction)
 * - Analyst cannot call goAML export API (403)
 * - Maker-checker workflow and error message (AML/CFT, not PMLA)
 * - Global jurisdictions: UAE (AED), Kenya (KES), US (USD)
 * - entity_country / fiu_name / local_currency fields accepted
 * - PASSPORT and NATIONAL_ID are valid subject id_type values
 * - currency field can be empty — backend derives from entity_country
 */

const MANAGER_EMAIL = "ravi.kumar@opsowl.dev";
const MANAGER_PASSWORD = "OpsOwl@2024";
const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASSWORD = "OpsOwl@2024";
const ANALYST_EMAIL = "divya.reddy@opsowl.dev";
const ANALYST_PASSWORD = "OpsOwl@2024";

const FULL_ENTITY_IN = {
  name: "Sunrise Bank Ltd",
  rentity_id: "FIU-IN-SUN-0042",
  rentity_branch: "BRANCH-MUM-001",
  swift_code: "SUNLINBB",
  submission_code: "E",
  officer_first_name: "Priya",
  officer_last_name: "Sharma",
  officer_gender: "F",
  officer_id_number: "PQRST5678Y",
  officer_id_type: "PAN",
  officer_occupation: "Chief Compliance Officer",
  officer_phone: "+91-9876543210",
  officer_email: "priya@sunrisebank.in",
  entity_address: "Fort, Mumbai",
  entity_city: "Mumbai",
  entity_state: "MH",
  entity_zip: "400001",
  entity_country: "IN",
  fiu_name: "FIU-IND",
  local_currency: "INR",
};

const FULL_ENTITY_AE = {
  name: "Gulf Trust Bank",
  rentity_id: "UAEFIU-GTB-001",
  rentity_branch: "BRANCH-DXB-001",
  swift_code: "GTBKAEAD",
  submission_code: "E",
  officer_first_name: "Fatima",
  officer_last_name: "Al-Rashid",
  officer_gender: "F",
  officer_id_number: "UAE-784-1990-0001234",
  officer_id_type: "NATIONAL_ID",
  officer_occupation: "Chief Compliance Officer",
  officer_phone: "+971-4-555-0100",
  officer_email: "fatima@gulftrust.ae",
  entity_address: "DIFC Gate District",
  entity_city: "Dubai",
  entity_state: "Dubai",
  entity_zip: "00000",
  entity_country: "AE",
  fiu_name: "UAEFIU",
  local_currency: "AED",
};

const FULL_ENTITY_KE = {
  name: "Savanna Finance Ltd",
  rentity_id: "FIUKE-SFL-007",
  rentity_branch: "BRANCH-NBI-001",
  swift_code: "SFKEKENA",
  submission_code: "E",
  officer_first_name: "Amara",
  officer_last_name: "Odhiambo",
  officer_gender: "F",
  officer_id_number: "34567890",
  officer_id_type: "NATIONAL_ID",
  officer_occupation: "Chief Compliance Officer",
  officer_phone: "+254-20-555-0100",
  officer_email: "amara@savanna.co.ke",
  entity_address: "Upper Hill, Nairobi",
  entity_city: "Nairobi",
  entity_state: "Nairobi",
  entity_zip: "00100",
  entity_country: "KE",
  fiu_name: "FIU Kenya",
  local_currency: "KES",
};

function minimalFilingBody(entityOverrides = {}, subjectOverrides = {}, txOverrides = {}) {
  return {
    report_type: "STR",
    suspicion_basis: "Unusual transaction pattern inconsistent with customer profile",
    narrative: "E2E test narrative",
    evidence_summary: "",
    filing_entity: { name: "Test Entity", ...entityOverrides },
    subject: { first_name: "John", last_name: "Doe", name: "John Doe", ...subjectOverrides },
    transactions: [
      {
        date: "2026-04-15",
        amount: 950000,
        currency: "",
        description: "Test transaction",
        transmode_code: "C",
        from_institution: "Test Entity",
        ...txOverrides,
      },
    ],
  };
}

test.describe("goAML STR filing — global", () => {
  let token: string;
  let caseId: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E goAML Global STR Test Case", "high");
    caseId = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  // ── Access control ────────────────────────────────────────────────────────────

  test("analyst does not see STR tab on case detail", async ({ page }) => {
    await login(page, ANALYST_EMAIL, ANALYST_PASSWORD);
    await page.goto(`/cases/${caseId}`);
    await expect(page.getByRole("button", { name: /^STR$/i })).not.toBeVisible();
  });

  test("analyst cannot call goAML export API", async () => {
    const analystToken = await apiTokenAs(ANALYST_EMAIL, ANALYST_PASSWORD);
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    // Create a draft first so we have a filing ID to test against
    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(minimalFilingBody()),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();

    const res = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/export/goaml`, {
      headers: { Authorization: `Bearer ${analystToken}` },
    });
    expect(res.status).toBe(403);
  });

  // ── Validation endpoint ───────────────────────────────────────────────────────

  test("validate endpoint returns errors for incomplete filing", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify({
        report_type: "STR",
        suspicion_basis: "Test suspicion",
        narrative: "Test narrative",
        evidence_summary: "",
        filing_entity: { name: "Test Bank" },
        subject: { name: "John Doe" },
        transactions: [],
      }),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    expect(valRes.status).toBe(200);
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(false);
    expect(val.errors.length).toBeGreaterThan(0);

    const fields = val.errors.map((e: { field: string }) => e.field);
    expect(fields).toContain("filing_entity.rentity_id");
    expect(fields).toContain("transactions");
  });

  test("validate endpoint passes for complete India filing (IN/INR/PAN)", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const body = minimalFilingBody(
      {
        ...FULL_ENTITY_IN,
        officer_title: "Ms",
        officer_gender: "F",
      },
      {
        first_name: "Ramesh", last_name: "Kumar", name: "Ramesh Kumar",
        gender: "M", id_type: "PAN", id_number: "ABCDE1234F", pan: "ABCDE1234F",
        account_number: "1234567890", account_name: "Ramesh Kumar",
      },
      { currency: "INR" }
    );

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(body),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
    expect((val.errors ?? []).length).toBe(0);
  });

  // ── Global jurisdictions ──────────────────────────────────────────────────────

  test("UAE jurisdiction filing accepted — entity_country=AE, local_currency=AED, NATIONAL_ID", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const body = minimalFilingBody(
      { ...FULL_ENTITY_AE, officer_title: "Ms", officer_gender: "F" },
      {
        first_name: "Mohammed", last_name: "Al-Amin", name: "Mohammed Al-Amin",
        gender: "M", id_type: "NATIONAL_ID", id_number: "784-1985-1234567-8",
        account_number: "AE070331234567890123456",
      },
      { currency: "AED" }
    );

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(body),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();
    expect(draft.filing_entity.entity_country).toBe("AE");
    expect(draft.filing_entity.fiu_name).toBe("UAEFIU");
    expect(draft.filing_entity.local_currency).toBe("AED");

    // Validate — should pass with complete fields
    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
  });

  test("Kenya jurisdiction filing accepted — entity_country=KE, local_currency=KES", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const body = minimalFilingBody(
      { ...FULL_ENTITY_KE, officer_title: "Ms", officer_gender: "F" },
      {
        first_name: "Peter", last_name: "Kamau", name: "Peter Kamau",
        gender: "M", id_type: "NATIONAL_ID", id_number: "34567891",
        account_number: "KE1234567890",
      },
      { currency: "KES" }
    );

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(body),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();
    expect(draft.filing_entity.entity_country).toBe("KE");
    expect(draft.filing_entity.local_currency).toBe("KES");
  });

  test("currency empty — backend derives from entity_country", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    // currency: "" — backend should derive AED from entity_country=AE
    const body = minimalFilingBody(
      { ...FULL_ENTITY_AE, officer_title: "Ms", officer_gender: "F" },
      {
        first_name: "Layla", last_name: "Hassan", name: "Layla Hassan",
        gender: "F", id_type: "PASSPORT", id_number: "P123456789",
        account_number: "AE070331234567890123457",
      },
      { currency: "" }  // intentionally empty
    );

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(body),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();

    // goAML export should still succeed (backend fills currency from entity_country)
    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
  });

  // ── Global ID types ───────────────────────────────────────────────────────────

  test("PASSPORT id_type accepted — filing passes validation", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const body = minimalFilingBody(
      { ...FULL_ENTITY_AE, officer_title: "Mr", officer_gender: "M", officer_id_type: "PASSPORT", officer_id_number: "P98765432" },
      {
        first_name: "Sofia", last_name: "Reyes", name: "Sofia Reyes",
        gender: "F", id_type: "PASSPORT", id_number: "P12345678",
        account_number: "AE070331234567890099999",
      },
      { currency: "AED" }
    );

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(body),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();
    expect(draft.subject.id_type).toBe("PASSPORT");

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
  });

  test("TAX_ID id_type accepted for US jurisdiction filing", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    const usEntity = {
      name: "First National Bank",
      rentity_id: "FINCEN-FNB-00123",
      rentity_branch: "BRANCH-NYC-001",
      swift_code: "FNBKUS33",
      submission_code: "E",
      officer_first_name: "James",
      officer_last_name: "Carter",
      officer_gender: "M",
      officer_id_number: "123-45-6789",
      officer_id_type: "TAX_ID",
      officer_occupation: "BSA Officer",
      officer_phone: "+1-212-555-0100",
      officer_email: "james@fnb.com",
      entity_address: "350 Park Ave",
      entity_city: "New York",
      entity_state: "NY",
      entity_zip: "10022",
      entity_country: "US",
      fiu_name: "FinCEN",
      local_currency: "USD",
    };

    const body = minimalFilingBody(
      { ...usEntity, officer_title: "Mr" },
      {
        first_name: "Emily", last_name: "Morgan", name: "Emily Morgan",
        gender: "F", id_type: "TAX_ID", id_number: "987-65-4321",
        account_number: "US-1234567890",
      },
      { currency: "USD" }
    );

    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(body),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();
    expect(draft.filing_entity.entity_country).toBe("US");
    expect(draft.filing_entity.fiu_name).toBe("FinCEN");

    const valRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/validate/goaml`, {
      headers: { Authorization: `Bearer ${mgr}` },
    });
    const { data: val } = await valRes.json();
    expect(val.ready).toBe(true);
  });

  // ── Maker-checker error message ───────────────────────────────────────────────

  test("maker-checker violation returns AML/CFT message (not PMLA)", async () => {
    const mgr = await apiTokenAs(MANAGER_EMAIL, MANAGER_PASSWORD);

    // Create and submit a filing as manager
    const draftRes = await fetch(`${API_URL}/cases/${caseId}/str`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${mgr}` },
      body: JSON.stringify(minimalFilingBody()),
    });
    expect(draftRes.status).toBe(201);
    const { data: draft } = await draftRes.json();

    // Submit
    const submitRes = await fetch(`${API_URL}/cases/${caseId}/str/${draft.id}/submit`, {
      method: "POST",
      headers: { Authorization: `Bearer ${mgr}` },
    });
    expect(submitRes.status).toBe(200);

    // Same manager tries to approve (maker-checker violation)
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

  // ── UI: full create flow with global fields ───────────────────────────────────

  test("UI: create filing with global fields — NATIONAL_ID default, generic placeholders", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASSWORD);
    await page.goto(`/cases/${caseId}/str`);

    await page.getByRole("button", { name: /new filing/i }).click();

    // Basic fields
    await page.getByPlaceholder(/unusual cash/i).fill("Suspicious large transfers inconsistent with customer profile");
    await page.getByPlaceholder(/detailed description/i).fill("Cross-border wire transfers with no clear business purpose.");

    // Subject — ID type dropdown should default to NATIONAL_ID
    const idTypeSelect = page.locator("select").filter({ hasText: "NATIONAL_ID" }).first();
    await expect(idTypeSelect).toBeVisible({ timeout: 5_000 });

    await page.getByPlaceholder("First name").fill("Ahmed");
    await page.getByPlaceholder("Last name").fill("Al-Farsi");
    // Placeholder is now "Identifier value" (not ABCDE1234F) since NATIONAL_ID is default
    await page.getByPlaceholder(/identifier value/i).first().fill("784-1990-7654321-0");

    // Add a transaction
    await page.getByRole("button", { name: /add transaction/i }).click();
    const dateInputs = page.locator("input[type='date']");
    await dateInputs.first().fill("2026-04-20");
    const amountInputs = page.getByPlaceholder(/e.g. 1500000/i);
    await amountInputs.first().fill("250000");

    // Expand goAML fields
    await page.getByRole("button", { name: /reporting officer/i }).click();

    // FIU Registration ID label (was FIU-IND Entity ID)
    await expect(page.getByText(/fiu registration id/i)).toBeVisible();

    await page.getByPlaceholder("Your bank / NBFC name").fill("Gulf Trust Bank");
    await page.getByPlaceholder(/FIU-IN-BANK-0001/i).fill("UAEFIU-GTB-001");

    // Entity Country field
    await page.getByPlaceholder(/e\.g\. IN, AE, KE/i).fill("AE");

    // FIU Name field
    await page.getByPlaceholder(/FIU-IND, UAEFIU/i).fill("UAEFIU");

    // Local Currency field
    await page.getByPlaceholder(/INR, AED, KES/i).fill("AED");

    await page.getByPlaceholder("Officer first name").fill("Fatima");
    await page.getByPlaceholder("Officer last name").fill("Al-Rashid");

    // Officer ID number — placeholder is now "National ID / Tax ID" (not "Officer PAN")
    await page.getByPlaceholder(/national id \/ tax id/i).fill("UAE-784-1990-0001234");

    // Save draft
    await page.getByRole("button", { name: /save draft/i }).click();

    // Wait for modal to close then verify filing card appeared
    await expect(page.getByRole("heading", { name: /new regulatory filing draft/i })).not.toBeVisible({ timeout: 8_000 });
    await expect(page.getByText("Suspicious large transfers")).toBeVisible({ timeout: 5_000 });
  });

  // ── Export gate ───────────────────────────────────────────────────────────────

  test("export goAML XML button not visible for draft filings", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto(`/cases/${caseId}/str`);

    const exportBtn = page.getByRole("button", { name: /export goaml xml/i });
    await expect(exportBtn).not.toBeVisible();
  });

  // ── goAML readiness panel ─────────────────────────────────────────────────────

  test("goAML readiness panel appears in filing card for managers", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASSWORD);
    await page.goto(`/cases/${caseId}/str`);

    // Wait for the page to fully load with manager-role controls (Submit for Approval
    // is also gated on isManagerOrAbove — ensures user context is resolved before
    // we look for the readiness panel button)
    await expect(page.getByRole("button", { name: /submit for approval/i }).first()).toBeVisible({ timeout: 10_000 });

    await page.getByRole("button", { name: /goaml 4.0 readiness/i }).first().click();
    await expect(
      page.getByText(/required fields present|required before export/i).first()
    ).toBeVisible({ timeout: 6_000 });
  });

  // ── UI: page footer shows UNODC (not FIU-IND) ────────────────────────────────

  test("page header shows goAML 4.0 (UNODC) not FIU-IND", async ({ page }) => {
    await login(page, MANAGER_EMAIL, MANAGER_PASSWORD);
    await page.goto(`/cases/${caseId}/str`);

    // The subtitle row should contain UNODC
    await expect(page.getByText(/goAML 4\.0 \(UNODC\)/i)).toBeVisible({ timeout: 6_000 });
    // Should NOT say FIU-IND in the subtitle
    await expect(page.getByText(/goAML 4\.0 \/ FIU-IND/i)).not.toBeVisible();
  });
});
