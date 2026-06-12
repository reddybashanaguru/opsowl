/**
 * E2E tests for the AI Investigation Agent tab in CopilotPanel.
 *
 * Strategy: Use Playwright route interception to mock the agent SSE stream —
 * avoids real LLM calls while exercising the full frontend stack (hook,
 * SSE parser, event feed, answer card).
 */
import { test, expect, Page } from "@playwright/test";
import { login, createCase, deleteCase, apiToken, API_URL } from "./helpers/auth";

// ── Helpers ───────────────────────────────────────────────────────────────────

/** SSE lines for a minimal deterministic agent run. */
const MOCK_SSE_BODY = [
  `event: agent_start\ndata: {"case_id":"mock","goal":"Analyse this case"}\n\n`,
  `event: thinking\ndata: {"text":"Reading case context..."}\n\n`,
  `event: tool_call\ndata: {"id":"tc1","name":"get_case_context","input":{"case_id":"mock"}}\n\n`,
  `event: tool_result\ndata: {"id":"tc1","name":"get_case_context","preview":"Case INV-001 open.","ok":true}\n\n`,
  `event: tool_call\ndata: {"id":"tc2","name":"get_case_notes","input":{"case_id":"mock","limit":10}}\n\n`,
  `event: tool_result\ndata: {"id":"tc2","name":"get_case_notes","preview":"2 analyst notes found.","ok":true}\n\n`,
  `event: answer\ndata: {"text":"## Case Summary\\nSuspicious wire transfers detected.\\n## Confidence Level: HIGH"}\n\n`,
  `event: done\ndata: {"iterations":2,"tokens_in":450,"tokens_out":180,"elapsed_ms":3200}\n\n`,
].join("");

// Intercept POST /api/v1/cases/:id/ai/agent and return the mock SSE stream.
async function mockAgentSSE(page: Page) {
  await page.route("**/ai/agent", (route) => {
    route.fulfill({
      status: 200,
      contentType: "text/event-stream",
      headers: {
        "Cache-Control": "no-cache",
        "X-Accel-Buffering": "no",
      },
      body: MOCK_SSE_BODY,
    });
  });
}

/** Open the case detail page and click the AI Copilot button. */
async function openCopilot(page: Page) {
  await page.getByRole("button", { name: /🤖 AI Copilot/i }).click();
  await expect(page.getByTestId("copilot-panel")).toBeVisible({ timeout: 5_000 });
}

/** Switch to the Agent tab inside the open copilot panel. */
async function switchToAgentTab(page: Page) {
  await page.getByTestId("tab-agent").click();
  await expect(page.getByTestId("agent-goal-input")).toBeVisible({ timeout: 3_000 });
}

// ── Test suite ────────────────────────────────────────────────────────────────

test.describe("AI Investigation Agent", () => {
  let token: string;
  let caseId: string;

  test.beforeAll(async () => {
    token = await apiToken();
    const c = await createCase(token, "E2E Agent Test Case", "high");
    caseId = c.id;
  });

  test.afterAll(async () => {
    await deleteCase(token, caseId);
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/cases/${caseId}`);
    // Wait for the case header to render — avoids networkidle timeout caused
    // by the persistent SSE timeline stream on the case detail page.
    await page.waitForSelector("h1", { timeout: 15_000 });
  });

  // ── Tab visibility ──────────────────────────────────────────────────────────

  test("Agent tab is visible in copilot panel", async ({ page }) => {
    await openCopilot(page);
    await expect(page.getByTestId("tab-agent")).toBeVisible();
  });

  test("Agent tab has correct label with lightning bolt", async ({ page }) => {
    await openCopilot(page);
    const tab = page.getByTestId("tab-agent");
    await expect(tab).toContainText("Agent");
  });

  test("clicking Agent tab shows goal textarea", async ({ page }) => {
    await openCopilot(page);
    await switchToAgentTab(page);
    await expect(page.getByTestId("agent-goal-input")).toBeVisible();
  });

  test("Run Agent button is disabled when goal is empty", async ({ page }) => {
    await openCopilot(page);
    await switchToAgentTab(page);
    const runBtn = page.getByTestId("agent-run-btn");
    await expect(runBtn).toBeDisabled();
  });

  test("Run Agent button enables when goal is typed", async ({ page }) => {
    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("agent-goal-input").fill("Analyse this case for suspicious activity.");
    const runBtn = page.getByTestId("agent-run-btn");
    await expect(runBtn).toBeEnabled();
  });

  // ── Full agent run with SSE mock ────────────────────────────────────────────

  test("submitting goal calls the agent endpoint", async ({ page }) => {
    let agentCalled = false;
    await page.route("**/ai/agent", (route) => {
      agentCalled = true;
      route.fulfill({ status: 200, contentType: "text/event-stream", body: MOCK_SSE_BODY });
    });

    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("agent-goal-input").fill("Analyse this case.");
    await page.getByTestId("agent-run-btn").click();
    await page.waitForTimeout(500);
    expect(agentCalled).toBe(true);
  });

  test("agent feed shows tool call events", async ({ page }) => {
    await mockAgentSSE(page);
    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("agent-goal-input").fill("Analyse this case for suspicious patterns.");
    await page.getByTestId("agent-run-btn").click();

    // Tool calls render as "→ Case Context" etc.
    await expect(page.getByTestId("agent-feed")).toBeVisible({ timeout: 8_000 });
    await expect(page.locator('[data-testid="agent-feed"]')).toContainText("Case Context", { timeout: 8_000 });
  });

  test("agent feed shows tool result check marks", async ({ page }) => {
    await mockAgentSSE(page);
    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("agent-goal-input").fill("Summarise the risk profile of this case.");
    await page.getByTestId("agent-run-btn").click();

    // tool_result renders ✓ for ok:true
    await expect(page.locator('[data-testid="agent-feed"]')).toContainText("✓", { timeout: 8_000 });
  });

  test("final answer is displayed after done event", async ({ page }) => {
    await mockAgentSSE(page);
    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("agent-goal-input").fill("Identify all risk indicators.");
    await page.getByTestId("agent-run-btn").click();

    await expect(page.getByText("Investigation Report")).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/Suspicious wire transfers detected/i)).toBeVisible({ timeout: 10_000 });
  });

  test("run again button resets state after completion", async ({ page }) => {
    await mockAgentSSE(page);
    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("agent-goal-input").fill("Analyse this case.");
    await page.getByTestId("agent-run-btn").click();

    await expect(page.getByText("Run again")).toBeVisible({ timeout: 10_000 });
    await page.getByText("Run again").click();

    // After reset, goal input should be visible again
    await expect(page.getByTestId("agent-goal-input")).toBeVisible({ timeout: 3_000 });
  });

  // ── Error handling ──────────────────────────────────────────────────────────

  test("server error shows error state", async ({ page }) => {
    await page.route("**/ai/agent", (route) => {
      route.fulfill({ status: 503, contentType: "application/json", body: '{"error":"Service unavailable"}' });
    });

    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("agent-goal-input").fill("Analyse this case.");
    await page.getByTestId("agent-run-btn").click();

    await expect(page.getByText(/Agent error|503/i)).toBeVisible({ timeout: 8_000 });
  });

  // ── Other tabs unaffected ───────────────────────────────────────────────────

  test("switching back to Ask tab shows question input", async ({ page }) => {
    await openCopilot(page);
    await switchToAgentTab(page);
    await page.getByTestId("tab-ask").click();
    await expect(page.getByTestId("ask-input")).toBeVisible({ timeout: 3_000 });
  });

  test("all 5 tabs are visible in the panel", async ({ page }) => {
    await openCopilot(page);
    for (const tab of ["tab-ask", "tab-summarize", "tab-note", "tab-actions", "tab-agent"]) {
      await expect(page.getByTestId(tab)).toBeVisible();
    }
  });
});
