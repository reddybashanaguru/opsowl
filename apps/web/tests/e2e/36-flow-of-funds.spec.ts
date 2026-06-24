import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * QA suite for Feature 2: Flow of Funds Visualization (React Flow upgrade)
 *
 * QA feedback-loop lessons from Feature 1 applied here:
 *  — Always use { exact: true } for labels that also appear in the edge legend
 *  — "Net Flow" is used as the unique drawer anchor (only in AccountDrawer, not summary/legend)
 *  — Truly isolated case (no transactions) used for empty state — not just a fresh case
 *    that might still have the flow API return an empty but non-zero graph
 *  — `.first()` used when label can match both summary strip and legend
 *  — React Flow pane click used to close drawer (tests onPaneClick handler directly)
 */

const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASS  = "OpsOwl@2024";

async function createTxn(
  token: string,
  caseId: string,
  overrides: Record<string, unknown> = {},
) {
  const suffix = Math.floor(Math.random() * 999999);
  const body = {
    txn_ref:          `FOFTEST${suffix}`,
    txn_date:         "2024-03-15",
    amount:           50000,
    currency:         "INR",
    txn_type:         "debit",
    channel:          "neft",
    status:           "completed",
    description:      "Test transfer",
    narration:        "E2E flow test",
    from_account_raw: "ACC-SRC-001",
    to_account_raw:   "ACC-DST-001",
    from_name_raw:    "Source Corp",
    to_name_raw:      "Destination Ltd",
    is_flagged:       false,
    risk_score:       10,
    ...overrides,
  };
  const res = await fetch(`${API_URL}/cases/${caseId}/transactions`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify(body),
  });
  return (await res.json()).data as { id: string };
}

test.describe("Flow of Funds Visualization", () => {
  let token: string;
  let caseWithFlow: string;   // case with 3 transactions (creates a 3-node flow graph)
  let caseWithSusp: string;   // case with a suspicious (flagged) transaction
  let caseEmpty: string;      // case with zero transactions (empty state)

  test.beforeAll(async () => {
    token = await apiToken();

    // Case 1: normal multi-hop flow (A → B → C chain)
    const c1 = await createCase(token, "E2E Flow of Funds Main");
    caseWithFlow = c1.id;

    const suffix1 = Date.now();
    await createTxn(token, caseWithFlow, {
      txn_ref:          `FOF-AB-${suffix1}`,
      amount:           120000,
      from_account_raw: "ACC-ALPHA-001",
      to_account_raw:   "ACC-BETA-002",
      from_name_raw:    "Alpha Source",
      to_name_raw:      "Beta Intermediary",
    });
    await createTxn(token, caseWithFlow, {
      txn_ref:          `FOF-BC-${suffix1}`,
      amount:           95000,
      from_account_raw: "ACC-BETA-002",
      to_account_raw:   "ACC-GAMMA-003",
      from_name_raw:    "Beta Intermediary",
      to_name_raw:      "Gamma Destination",
    });

    // Case 2: suspicious flow
    const c2 = await createCase(token, "E2E Flow of Funds Suspicious");
    caseWithSusp = c2.id;

    const suffix2 = Date.now() + 1;
    await createTxn(token, caseWithSusp, {
      txn_ref:          `FOF-SUSP-${suffix2}`,
      amount:           250000,
      is_flagged:       true,
      flag_reason:      "Structuring",
      from_account_raw: "ACC-SUSP-SRC",
      to_account_raw:   "ACC-SUSP-DST",
      from_name_raw:    "Suspect Sender",
      to_name_raw:      "Suspect Receiver",
    });

    // Case 3: zero transactions — used for empty state
    const c3 = await createCase(token, "E2E Flow of Funds Empty");
    caseEmpty = c3.id;
    // Deliberately add no transactions
  });

  test.afterAll(async () => {
    await deleteCase(token, caseWithFlow);
    await deleteCase(token, caseWithSusp);
    await deleteCase(token, caseEmpty);
  });

  // ── Tab navigation ─────────────────────────────────────────────────────────

  test("transactions subnav button is visible on case detail", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}`);
    // CaseSubNav renders tabs as <button> elements (uses router.push), not <a> tags
    await expect(page.getByRole("button", { name: "Transactions" })).toBeVisible();
  });

  test("transactions page loads with Transaction List view by default", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await expect(page.getByText("Transaction List", { exact: true })).toBeVisible();
    await expect(page.getByText("Flow of Funds", { exact: true }).first()).toBeVisible();
  });

  // ── Switching to flow view ─────────────────────────────────────────────────

  test("clicking Flow of Funds shows React Flow canvas", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await expect(page.locator(".react-flow__renderer")).toBeVisible({ timeout: 8_000 });
  });

  test("flow graph renders nodes for transactions", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await expect(page.locator(".react-flow__node").first()).toBeVisible({ timeout: 8_000 });
  });

  test("at least 2 React Flow nodes rendered for a 2-hop flow", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    const nodeCount = await page.locator(".react-flow__node").count();
    expect(nodeCount).toBeGreaterThanOrEqual(2);
  });

  // ── Summary strip ──────────────────────────────────────────────────────────

  test("summary strip shows Total Flow label", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.getByText("Total Flow", { exact: true }).first()).toBeVisible();
  });

  test("summary strip shows Accounts count", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.getByText("Accounts", { exact: true }).first()).toBeVisible();
  });

  // ── Node click → account drawer ────────────────────────────────────────────

  test("clicking a node opens account detail drawer", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await page.locator(".react-flow__node").first().click();
    // "Net Flow" is unique to the AccountDrawer — it does not appear in summary strip or legend
    await expect(page.getByText("Net Flow")).toBeVisible({ timeout: 5_000 });
  });

  test("account drawer shows Total In and Total Out", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await page.locator(".react-flow__node").first().click();
    await expect(page.getByText("Total In")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText("Total Out")).toBeVisible();
  });

  test("account drawer shows Transactions count", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await page.locator(".react-flow__node").first().click();
    await expect(page.getByText("Net Flow")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText("Transactions").first()).toBeVisible();
  });

  test("clicking pane closes account drawer", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await page.locator(".react-flow__node").first().click();
    await expect(page.getByText("Net Flow")).toBeVisible({ timeout: 5_000 });
    // Click the pane (background) to close the drawer
    await page.locator(".react-flow__pane").click({ position: { x: 20, y: 20 } });
    await expect(page.getByText("Net Flow")).not.toBeVisible({ timeout: 3_000 });
  });

  // ── Suspicious highlight ───────────────────────────────────────────────────

  test("Highlight Suspicious button appears when there are flagged transactions", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithSusp}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.getByText("Highlight Suspicious")).toBeVisible({ timeout: 5_000 });
  });

  test("Highlight Suspicious toggles to Show All on click", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithSusp}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await page.getByText("Highlight Suspicious").click();
    await expect(page.getByText("Show All")).toBeVisible({ timeout: 3_000 });
  });

  test("Show All toggles back to Highlight Suspicious", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithSusp}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await page.getByText("Highlight Suspicious").click();
    await page.getByText("Show All").click();
    await expect(page.getByText("Highlight Suspicious")).toBeVisible({ timeout: 3_000 });
  });

  // ── Suspicious flag in summary ─────────────────────────────────────────────

  test("Suspicious count in summary strip is non-zero for flagged case", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithSusp}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.getByText("Suspicious", { exact: true })).toBeVisible();
  });

  // ── Controls & chrome ──────────────────────────────────────────────────────

  test("React Flow Controls panel is visible", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.locator(".react-flow__controls")).toBeVisible();
  });

  test("MiniMap is visible in flow view", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.locator(".react-flow__minimap")).toBeVisible();
  });

  test("PNG export button is present", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.getByText("PNG")).toBeVisible();
  });

  // ── Legend ─────────────────────────────────────────────────────────────────

  test("legend shows Source label", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    // Legend uses specific phrasing — "Source (outflow origin)" — distinct from node role label
    await expect(page.getByText("Source (outflow origin)")).toBeVisible();
  });

  test("legend shows Destination label", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.getByText("Destination (inflow end)")).toBeVisible();
  });

  test("legend shows suspicious flow label", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await expect(page.getByText("Suspicious flow (animated)")).toBeVisible();
  });

  // ── Empty state ────────────────────────────────────────────────────────────

  test("empty state shown when case has no transactions", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseEmpty}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    // Should NOT show React Flow nodes — show empty state message
    await expect(page.getByText("No transactions to visualize")).toBeVisible({ timeout: 8_000 });
  });

  test("empty state does not render React Flow nodes", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseEmpty}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await expect(page.getByText("No transactions to visualize")).toBeVisible({ timeout: 8_000 });
    const nodeCount = await page.locator(".react-flow__node").count();
    expect(nodeCount).toBe(0);
  });

  // ── Returning to list view ─────────────────────────────────────────────────

  test("clicking Transaction List switches back from flow view", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASS, `/cases/${caseWithFlow}/transactions`);
    await page.getByText("Flow of Funds", { exact: true }).first().click();
    await page.waitForSelector(".react-flow__node", { timeout: 8_000 });
    await page.getByText("Transaction List", { exact: true }).click();
    // React Flow should be gone; list filter should be visible
    await expect(page.locator(".react-flow__renderer")).not.toBeVisible({ timeout: 3_000 });
  });
});
