import { test, expect } from "@playwright/test";
import { login, apiToken, createCase, deleteCase, API_URL } from "./helpers/auth";

let token: string;
let caseId: string;
let txnId: string;

test.describe("Transaction Timeline & Flow-of-Funds", () => {
  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "Transaction E2E Test Case");
    caseId = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  test("transactions tab visible in case subnav", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}`);
    await expect(page.getByRole("button", { name: /Transactions/i })).toBeVisible();
  });

  test("transactions page renders empty state", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/transactions`);
    await expect(page.getByText(/No transactions|linked to this case yet/i)).toBeVisible();
  });

  test("API: create a transaction", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        txn_ref: `E2E-TXN-${Date.now()}`,
        txn_date: "2026-05-15",
        amount: 450000,
        currency: "INR",
        txn_type: "debit",
        channel: "neft",
        narration: "NEFT/RTGS/HDFC0000123/Rahul Sharma",
        from_account_raw: "HDFC0000123456789",
        to_account_raw: "ICIC0000987654321",
        from_name_raw: "Test Subject",
        to_name_raw: "Counterparty Ltd",
        counterparty_bank: "ICICI Bank",
        counterparty_country: "IN",
      }),
    });
    expect(res.status).toBe(201);
    const json = await res.json();
    txnId = json.data.id;
    expect(txnId).toBeTruthy();
  });

  test("API: link transaction to case", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions/link`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        transaction_id: txnId,
        relevance_note: "Round number transfer, just under reporting threshold",
      }),
    });
    expect(res.status).toBe(201);
  });

  test("API: transaction appears in case list", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.length).toBe(1);
    expect(json.data[0].amount).toBe(450000);
  });

  test("API: transaction summary returns counts", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions/summary`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.total_count).toBe(1);
    expect(json.data.total_amount).toBeGreaterThan(0);
  });

  test("API: flag a transaction", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions/${txnId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ is_flagged: true, flag_reason: "Round-number transfer, structuring pattern" }),
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.is_flagged).toBe(true);
  });

  test("API: flow graph returns nodes and edges", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions/flow`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.nodes.length).toBeGreaterThanOrEqual(2);
    expect(json.data.edges.length).toBeGreaterThanOrEqual(1);
    expect(json.data.edges[0].is_suspicious).toBe(true);
  });

  test("transactions page renders with data", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/transactions`);
    await page.waitForTimeout(1500);
    await expect(page.getByText(/E2E-TXN-/i)).toBeVisible();
  });

  test("summary bar shows total flow and flagged count", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/transactions`);
    await page.waitForTimeout(1500);
    await expect(page.getByText(/Transactions/i).first()).toBeVisible();
    await expect(page.getByText(/Flagged/i).first()).toBeVisible();
  });

  test("flow of funds view renders SVG", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/transactions`);
    await page.waitForTimeout(1000);
    await page.getByRole("button", { name: "Flow of Funds" }).click();
    await page.waitForTimeout(1500);
    await expect(page.locator("svg").first()).toBeVisible();
  });

  test("flagged filter shows only flagged transactions", async ({ page }) => {
    await login(page, undefined, undefined, `/cases/${caseId}/transactions`);
    await page.waitForTimeout(1000);
    await page.getByRole("button", { name: /Flagged only/i }).click();
    await page.waitForTimeout(500);
    // The flagged transaction should still be visible
    await expect(page.getByText(/E2E-TXN-/i)).toBeVisible();
  });

  test("API: unlink transaction from case", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions/${txnId}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
  });

  test("API: transaction no longer in case list after unlink", async () => {
    const res = await fetch(`${API_URL}/cases/${caseId}/transactions`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const json = await res.json();
    expect(json.data.length).toBe(0);
  });
});
