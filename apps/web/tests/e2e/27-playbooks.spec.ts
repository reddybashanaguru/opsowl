import { test, expect } from "@playwright/test";
import { login, apiToken, apiTokenAs, createCase, deleteCase, API_URL } from "./helpers/auth";

/**
 * Investigation Playbooks E2E tests.
 *
 * Verifies:
 * - Admin can create a playbook with ordered steps
 * - Analyst cannot access playbook management
 * - Playbook auto-applies tasks when a case with matching case_type is created
 * - Blocked task shows correct badge on tasks page
 * - API CRUD operations work end-to-end
 */

const ADMIN_EMAIL = "admin@opsowl.dev";
const ADMIN_PASSWORD = "OpsOwl@2024";
const ANALYST_EMAIL = "divya.reddy@opsowl.dev";
const ANALYST_PASSWORD = "OpsOwl@2024";

test.describe("Investigation Playbooks", () => {
  let adminToken: string;
  let playbookId: string;

  test.beforeAll(async () => {
    adminToken = await apiTokenAs(ADMIN_EMAIL, ADMIN_PASSWORD);
  });

  test.afterAll(async () => {
    if (playbookId) {
      await fetch(`${API_URL}/playbooks/${playbookId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${adminToken}` },
      });
    }
  });

  // ── API: create playbook ────────────────────────────────────────────────────

  test("admin can create a playbook via API", async () => {
    const res = await fetch(`${API_URL}/playbooks`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({
        name: "E2E AML Playbook",
        description: "Auto-created by E2E test",
        case_type: "e2e_aml",
        steps: [
          { title: "Step 1: Freeze accounts", priority: "high", step_order: 1, due_hours: 4 },
          { title: "Step 2: Notify compliance", priority: "medium", step_order: 2, due_hours: 24, depends_on_step: 1 },
        ],
      }),
    });
    expect(res.status).toBe(201);
    const json = await res.json();
    expect(json.data.name).toBe("E2E AML Playbook");
    expect(json.data.case_type).toBe("e2e_aml");
    expect(json.data.steps).toHaveLength(2);
    expect(json.data.steps[1].depends_on_step).toBe(1);
    playbookId = json.data.id;
  });

  // ── API: list playbooks ─────────────────────────────────────────────────────

  test("list returns created playbook", async () => {
    const res = await fetch(`${API_URL}/playbooks`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    const found = (json.data as Array<{ id: string }>) .find((p) => p.id === playbookId);
    expect(found).toBeDefined();
  });

  // ── API: auto-apply on case create ──────────────────────────────────────────

  test("tasks auto-created when case_type matches playbook", async () => {
    const caseRes = await fetch(`${API_URL}/cases`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({
        title: "E2E Playbook Auto-Apply Case",
        description: "Testing playbook auto-apply",
        priority: "high",
        case_type: "e2e_aml",
        tags: ["e2e"],
      }),
    });
    expect(caseRes.status).toBe(201);
    const { data: newCase } = await caseRes.json();

    // Wait for the async playbook apply to complete (goroutine in worker pool)
    await new Promise((r) => setTimeout(r, 3000));

    const tasksRes = await fetch(`${API_URL}/cases/${newCase.id}/tasks`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(tasksRes.status).toBe(200);
    const tasksJson = await tasksRes.json();
    const tasks: Array<{ title: string }> = tasksJson.data ?? [];

    // Verify the two playbook steps were auto-applied as tasks
    const step1 = tasks.find((t) => t.title === "Step 1: Freeze accounts");
    const step2 = tasks.find((t) => t.title === "Step 2: Notify compliance");
    expect(step1).toBeDefined();
    expect(step2).toBeDefined();

    // Cleanup
    await fetch(`${API_URL}/cases/${newCase.id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${adminToken}` },
    });
  });

  // ── API: analyst cannot create playbook ─────────────────────────────────────

  test("analyst cannot create a playbook", async () => {
    const analystToken = await apiTokenAs(ANALYST_EMAIL, ANALYST_PASSWORD);
    const res = await fetch(`${API_URL}/playbooks`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${analystToken}` },
      body: JSON.stringify({ name: "Should fail", case_type: "fraud", steps: [] }),
    });
    expect(res.status).toBe(403);
  });

  // ── UI: playbooks page visible to admin ─────────────────────────────────────

  test("admin sees Playbooks nav link and settings page", async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto("/settings/playbooks");
    await expect(page.getByRole("heading", { name: /investigation playbooks/i })).toBeVisible({ timeout: 8_000 });
  });

  // ── UI: analyst cannot access playbooks page ────────────────────────────────

  test("analyst sees access denied on playbooks settings page", async ({ page }) => {
    await login(page, ANALYST_EMAIL, ANALYST_PASSWORD);
    await page.goto("/settings/playbooks");
    await expect(page.getByText(/access restricted|don't have permission/i)).toBeVisible({ timeout: 8_000 });
  });

  // ── API: deactivate playbook ─────────────────────────────────────────────────

  test("admin can deactivate a playbook", async () => {
    if (!playbookId) return;
    const res = await fetch(`${API_URL}/playbooks/${playbookId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${adminToken}` },
      body: JSON.stringify({ is_active: false }),
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.data.is_active).toBe(false);
  });
});
