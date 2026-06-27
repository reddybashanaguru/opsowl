import { test, expect } from "@playwright/test";
import { login, apiToken, API_URL } from "./helpers/auth";

/**
 * Sweden FIPO/FI Compliance E2E Tests
 *
 * Lag (2017:630) om åtgärder mot penningtvätt och finansiering av terrorism
 * EU AMLR 2024 / AMLD6 (transposed into Swedish law)
 * Finanspolisen (FIPO) — Swedish FIU (goAML)
 * Finansinspektionen (FI) — AML supervisory authority
 *
 * Key Sweden differentiators vs other EU countries:
 * - Currency: SEK (kr) — Sweden is EU but NOT eurozone
 * - FIU: Finanspolisen (FIPO), not a generic "FIU"
 * - Deadline: "utan dröjsmål" — same-day/next-day (1 day)
 * - CTR threshold: SEK 15,000
 * - ID: Personnummer (Swedish personal number)
 * - AI Act: ENABLED (EU AI Act applies to all obliged entities)
 *
 * Covers:
 * 1. Sweden config — Finanspolisen (FIPO) regulator label
 * 2. SEK currency (not EUR) — Sweden not in eurozone
 * 3. kr symbol displayed correctly
 * 4. CTR threshold SEK 15,000
 * 5. Deadline 1 day — "utan dröjsmål" (without delay)
 * 6. goAML enabled — FIPO is a goAML user
 * 7. GDPR enabled — IMY (Swedish DPA) enforces EU GDPR
 * 8. UBO enabled at 25%
 * 9. EU AMLR 2024 law reference
 * 10. Personnummer in ID type list
 * 11. STR label (not SAR — Sweden uses STR terminology)
 * 12. Sanctions page loads — EU Consolidated list visible
 * 13. GDPR DSAR visible with Sweden config
 * 14. Cases page loads without error with Sweden config
 */

const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASS = "OpsOwl@2024";
const MANAGER_EMAIL = "ravi.kumar@opsowl.dev";
const MANAGER_PASS = "OpsOwl@2024";

async function applySwedenConfig(token: string) {
  const res = await fetch(`${API_URL}/config`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({
      str_label: "STR",
      str_reference_label: "FIPO Reference",
      str_regulator_label: "Finanspolisen (FIPO)",
      str_law_reference: "AML Act 2017 (2017:630) / EU AMLR 2024",
      str_id_label: "Personnummer / National ID",
      str_deadline_days: 1,
      local_currency: "SEK",
      currency_symbol: "kr",
      ctr_threshold: 15000,
      wire_threshold: 11000,
      retention_years: 5,
      goaml_enabled: true,
      ubo_enabled: true,
      ubo_threshold_pct: 25,
      gdpr_enabled: true,
      id_types: ["PASSPORT", "NATIONAL_ID", "TAX_ID", "DRIVING_LICENSE", "EU_RESIDENCE_PERMIT", "PERSONNUMMER", "OTHER"],
    }),
  });
  if (!res.ok) throw new Error(`applySwedenConfig failed: ${res.status}`);
}

test.describe("Sweden FIPO/FI Config", () => {
  test("Sweden config: Finanspolisen (FIPO) regulator label", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.str_regulator_label).toBe("Finanspolisen (FIPO)");
    expect(cfg.str_reference_label).toBe("FIPO Reference");
  });

  test("Sweden config: SEK currency — Sweden not in eurozone", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.local_currency).toBe("SEK");
    expect(cfg.local_currency).not.toBe("EUR");
  });

  test("Sweden config: kr symbol (not €)", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.currency_symbol).toBe("kr");
    expect(cfg.currency_symbol).not.toBe("€");
  });

  test("Sweden config: CTR threshold SEK 15,000", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.ctr_threshold).toBe(15000);
  });

  test("Sweden config: 1-day deadline — utan dröjsmål (without delay)", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    // Strictest STR deadline in the portfolio — FIPO expects same/next day
    expect(cfg.str_deadline_days).toBe(1);
  });

  test("Sweden config: goAML enabled — FIPO is a goAML user", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.goaml_enabled).toBe(true);
  });

  test("Sweden config: GDPR enabled — IMY (Swedish DPA) enforces EU GDPR", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.gdpr_enabled).toBe(true);
  });

  test("Sweden config: UBO enabled at 25% (EU AMLA 2024)", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.ubo_enabled).toBe(true);
    expect(cfg.ubo_threshold_pct).toBe(25);
  });

  test("Sweden config: EU AMLR 2024 law reference", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.str_law_reference).toContain("AMLR 2024");
    expect(cfg.str_law_reference).toContain("2017:630");
  });

  test("Sweden config: Personnummer in ID type list", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.id_types).toContain("PERSONNUMMER");
    expect(cfg.id_types).toContain("PASSPORT");
    expect(cfg.id_types).toContain("EU_RESIDENCE_PERMIT");
  });
});

test.describe("STR filing UI with Sweden config", () => {
  test("STR label shown (Sweden uses STR, not SAR)", async ({ page }) => {
    await applySwedenConfig(await apiToken(ADMIN_EMAIL, ADMIN_PASS));

    await login(page, MANAGER_EMAIL, MANAGER_PASS);
    await page.goto("/cases");
    await expect(page.getByText("STR", { exact: false }).first()).toBeVisible();
  });

  test("FIPO Reference label in config", async ({ page }) => {
    const token = await apiToken(ADMIN_EMAIL, ADMIN_PASS);
    await applySwedenConfig(token);

    const res = await fetch(`${API_URL}/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const cfg = await res.json();
    expect(cfg.str_reference_label).toBe("FIPO Reference");
  });
});

test.describe("Sanctions and GDPR with Sweden config", () => {
  test("Sanctions page loads — EU Consolidated list visible", async ({ page }) => {
    await applySwedenConfig(await apiToken(ADMIN_EMAIL, ADMIN_PASS));

    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/settings/sanctions");
    await expect(page.getByText("EU Consolidated", { exact: false })).toBeVisible();
  });

  test("GDPR DSAR settings page visible with Sweden config", async ({ page }) => {
    await applySwedenConfig(await apiToken(ADMIN_EMAIL, ADMIN_PASS));

    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/settings");
    await expect(page.getByText("GDPR", { exact: false }).first()).toBeVisible();
  });
});

test.describe("Cases with Sweden config", () => {
  test("Cases page loads without error with Sweden config", async ({ page }) => {
    await applySwedenConfig(await apiToken(ADMIN_EMAIL, ADMIN_PASS));

    await login(page, ADMIN_EMAIL, ADMIN_PASS);
    await page.goto("/cases");
    await expect(page).not.toHaveURL(/error/);
    const body = await page.textContent("body");
    expect(body).toBeTruthy();
  });
});
