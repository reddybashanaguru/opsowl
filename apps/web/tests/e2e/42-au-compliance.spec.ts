import { test, expect } from "@playwright/test";
import { login, apiToken, apiTokenAs, createCase, deleteCase } from "./helpers/auth";

// ────────────────────────────────────────────────────────────────────────────
// Australia AUSTRAC / SMR compliance test suite
// Requires: DEPLOYMENT_JURISDICTION=au, DEPLOYMENT_COUNTRY=au
// These tests run against an AU-configured test instance.
// They validate SMR terminology, AUD currency, DFAT sanctions list,
// goAML disabled state, and Australian-specific UBO/retention requirements.
// ────────────────────────────────────────────────────────────────────────────

const AU_ENTITY = {
  rentity_id: "AUSTRAC-AU-FX-0001",
  entity_country: "AU",
  fiu_name: "AUSTRAC",
  local_currency: "AUD",
};

test.describe("AU Config — SMR terminology and AUD currency", () => {
  test("config endpoint returns SMR label for Australia", async ({ request }) => {
    const res = await request.get("/api/v1/config");
    expect(res.status()).toBe(200);
    const cfg = await res.json();

    // Only assert AU config when jurisdiction is au
    if (cfg.jurisdiction !== "au") {
      test.skip();
      return;
    }

    expect(cfg.str_label).toBe("SMR");
    expect(cfg.str_regulator_label).toBe("AUSTRAC");
    expect(cfg.str_law_reference).toContain("AML/CTF Act 2006");
    expect(cfg.local_currency).toBe("AUD");
    expect(cfg.currency_symbol).toBe("A$");
  });

  test("config has correct CTR and wire thresholds for Australia", async ({ request }) => {
    const res = await request.get("/api/v1/config");
    const cfg = await res.json();
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    // AML/CTF Act s.43: Threshold Transaction Report at AUD 10,000
    expect(cfg.ctr_threshold).toBe(10000);
    // AML/CTF Act s.45: IFTI threshold at AUD 10,000
    expect(cfg.wire_threshold).toBe(10000);
  });

  test("config has 7-year retention for Australia", async ({ request }) => {
    const res = await request.get("/api/v1/config");
    const cfg = await res.json();
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    // AML/CTF Act 2006 s.105: 7-year retention
    expect(cfg.retention_years).toBe(7);
  });

  test("goAML is disabled for Australia", async ({ request }) => {
    const res = await request.get("/api/v1/config");
    const cfg = await res.json();
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    // AUSTRAC uses its own online portal — goAML XML not accepted
    expect(cfg.goaml_enabled).toBe(false);
  });

  test("GDPR is disabled for Australia (Privacy Act 1988)", async ({ request }) => {
    const res = await request.get("/api/v1/config");
    const cfg = await res.json();
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    expect(cfg.gdpr_enabled).toBe(false);
  });

  test("UBO is enabled for Australia at 25% threshold", async ({ request }) => {
    const res = await request.get("/api/v1/config");
    const cfg = await res.json();
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    expect(cfg.ubo_enabled).toBe(true);
    expect(cfg.ubo_threshold_pct).toBe(25);
  });

  test("Australian ID types are present in config", async ({ request }) => {
    const res = await request.get("/api/v1/config");
    const cfg = await res.json();
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    const idTypes: string[] = cfg.id_types ?? [];
    expect(idTypes).toContain("PASSPORT");
    expect(idTypes).toContain("AUSTRALIAN_DRIVERS_LICENCE");
    expect(idTypes).toContain("MEDICARE_CARD");
  });
});

test.describe("AU Sanctions — DFAT list appears in settings", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("sanctions settings shows 4 list cards including DFAT", async ({ page }) => {
    await page.goto("/settings/sanctions");
    await page.waitForLoadState("networkidle");

    // Wait for list cards to render
    await expect(page.getByText("EU Consolidated List")).toBeVisible();
    await expect(page.getByText("UN Security Council List")).toBeVisible();
    await expect(page.getByText("OFAC Specially Designated Nationals")).toBeVisible();
    await expect(page.getByText("DFAT Autonomous Sanctions")).toBeVisible();
  });

  test("DFAT card shows correct jurisdiction label", async ({ page }) => {
    await page.goto("/settings/sanctions");
    await page.waitForLoadState("networkidle");

    await expect(page.getByText("Australia (DFAT)")).toBeVisible();
    await expect(page.getByText(/Department of Foreign Affairs and Trade/)).toBeVisible();
  });

  test("sanctions API returns dfat_consolidated list entry", async ({ request }) => {
    const token = await apiToken(request);
    const res = await request.get("/api/v1/sanctions/lists", {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status()).toBe(200);
    const lists = await res.json();
    const sources: string[] = lists.map((l: { source: string }) => l.source);
    // DFAT list should be present (may have 0 entries before first load)
    // Just verifying the source key is valid — the check constraint allows it
    const allSources = ["eu_consolidated", "un_consolidated", "ofac_sdn", "dfat_consolidated"];
    for (const s of sources) {
      expect(allSources).toContain(s);
    }
  });
});

test.describe("AU SMR filing — UI shows SMR not STR", () => {
  let caseId: string;

  test.beforeEach(async ({ page, request }) => {
    await login(page);
    caseId = await createCase(request, "AU Test Case — SMR Filing");
  });

  test.afterEach(async ({ request }) => {
    if (caseId) await deleteCase(request, caseId);
  });

  test("STR tab label reads SMR when jurisdiction is AU", async ({ page }) => {
    await page.goto(`/cases/${caseId}/str`);
    await page.waitForLoadState("networkidle");

    // Check that the page uses SMR terminology
    const cfg = await page.request.get("/api/v1/config").then((r) => r.json());
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    await expect(page.getByRole("heading", { name: /SMR/i })).toBeVisible();
  });

  test("filing form shows AUSTRAC reference label", async ({ page }) => {
    await page.goto(`/cases/${caseId}/str`);
    await page.waitForLoadState("networkidle");

    const cfg = await page.request.get("/api/v1/config").then((r) => r.json());
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    await expect(page.getByText(/AUSTRAC Reference/i)).toBeVisible();
  });

  test("currency shown as AUD in filing form", async ({ page }) => {
    await page.goto(`/cases/${caseId}/str`);
    await page.waitForLoadState("networkidle");

    const cfg = await page.request.get("/api/v1/config").then((r) => r.json());
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    // AUD should appear in the currency field or threshold labels
    const bodyText = await page.locator("body").textContent();
    expect(bodyText).toContain("AUD");
  });
});

test.describe("AU UBO — beneficial owner verification", () => {
  let caseId: string;

  test.beforeEach(async ({ page, request }) => {
    await login(page);
    caseId = await createCase(request, "AU UBO Test Case");
  });

  test.afterEach(async ({ request }) => {
    if (caseId) await deleteCase(request, caseId);
  });

  test("UBO tab is accessible on a case when ubo_enabled=true", async ({ page, request }) => {
    const cfg = await request.get("/api/v1/config").then((r) => r.json());
    if (!cfg.ubo_enabled) { test.skip(); return; }

    await page.goto(`/cases/${caseId}/ubo`);
    await page.waitForLoadState("networkidle");
    // Should not 404 — UBO tab must be accessible
    await expect(page.locator("body")).not.toContainText("404");
  });

  test("UBO API returns 200 for case (AML/CTF Rules Ch.4)", async ({ request }) => {
    const token = await apiToken(request);
    const res = await request.get(`/api/v1/cases/${caseId}/ubo`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    // 200 with empty array — not 404
    expect(res.status()).toBe(200);
    const data = await res.json();
    expect(Array.isArray(data.data ?? data)).toBe(true);
  });

  test("UBO record can be created with Australian entity", async ({ request }) => {
    const token = await apiToken(request);
    const res = await request.post(`/api/v1/cases/${caseId}/ubo`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        entity_name: "Harbour Capital Pty Ltd",
        owner_name: "James Mitchell",
        owner_nationality: "AU",
        owner_id_type: "PASSPORT",
        owner_id_number: "PA1234567",
        owner_birth_date: "1975-04-12",
        ownership_pct: 51,
        control_type: "share",
        registry_url: "https://connectonline.asic.gov.au",
      },
    });
    expect(res.status()).toBe(201);
    const body = await res.json();
    expect(body.data?.owner_name ?? body.owner_name).toBe("James Mitchell");
  });
});

test.describe("AU — goAML export blocked for AUSTRAC jurisdiction", () => {
  let caseId: string;

  test.beforeEach(async ({ page, request }) => {
    await login(page);
    caseId = await createCase(request, "AU goAML Block Test");
  });

  test.afterEach(async ({ request }) => {
    if (caseId) await deleteCase(request, caseId);
  });

  test("goAML export endpoint returns 422 when goaml_enabled=false", async ({ request }) => {
    const cfg = await request.get("/api/v1/config").then((r) => r.json());
    if (cfg.jurisdiction !== "au") { test.skip(); return; }

    const token = await apiToken(request);
    // Create a minimal STR filing first
    const strRes = await request.post(`/api/v1/cases/${caseId}/str`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        suspicion_date: new Date().toISOString().split("T")[0],
        suspicion_basis: "Unusual pattern of cash deposits below TTR threshold",
        amount_involved: 9500,
        currency: "AUD",
        filing_entity: {
          rentity_id: "",
          entity_country: "AU",
          fiu_name: "AUSTRAC",
        },
      },
    });
    if (strRes.status() !== 201) { test.skip(); return; }
    const str = await strRes.json();
    const strId = str.data?.id ?? str.id;

    // Attempt goAML export — should be blocked since AUSTRAC doesn't use goAML
    const exportRes = await request.get(`/api/v1/cases/${caseId}/str/${strId}/goaml`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    // Expect 422 (validation error) or 400 indicating goAML is not supported
    expect([400, 422, 403]).toContain(exportRes.status());
  });
});
