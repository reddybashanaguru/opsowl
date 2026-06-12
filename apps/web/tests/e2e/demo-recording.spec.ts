// OpsOwl AI — Full Narrated Product Demo
// Audience: CTO / Chief Compliance Officer / Compliance Manager
// Personas: Priya Sharma (Admin) → Divya Reddy (Analyst) → Ravi Kumar (Manager)
// Run: cd apps/web && npx playwright test tests/e2e/demo-recording.spec.ts --project=chromium
// Output: test-results/.../video.webm  (mux with build-demo.sh to add narration)

import { test, expect, type Page } from "@playwright/test";
import { execSync, spawn } from "child_process";
import path from "path";

const AUDIO_DIR = "/tmp/opsowl_demo/audio";

test.use({
  video: { mode: "on", size: { width: 1440, height: 900 } },
  viewport: { width: 1440, height: 900 },
});

const wait = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));

// Get AIFF duration in milliseconds using afinfo
function audioDurationMs(scene: string): number {
  const file = path.join(AUDIO_DIR, `${scene}.aiff`);
  try {
    const out = execSync(`afinfo "${file}" 2>/dev/null`, { encoding: "utf8" });
    const match = out.match(/estimated duration:\s+([\d.]+)/);
    if (match) return Math.ceil(parseFloat(match[1]) * 1000) + 2500; // +2.5s buffer
  } catch {
    // ignore
  }
  return 30_000; // fallback 30s
}

// Play audio in background (non-blocking) and return scene duration in ms
function playScene(scene: string): number {
  const file = path.join(AUDIO_DIR, `${scene}.aiff`);
  spawn("afplay", [file], { detached: true, stdio: "ignore" });
  return audioDurationMs(scene);
}

// Runs scene content, then waits out any remaining audio duration
async function scene(
  name: string,
  duration: number,
  content: () => Promise<void>
): Promise<void> {
  const start = Date.now();
  await content();
  const elapsed = Date.now() - start;
  const remaining = duration - elapsed;
  if (remaining > 500) await wait(remaining);
}

async function loginAs(page: Page, email: string, password: string) {
  await page.goto("/login");
  await wait(800);
  await page.getByLabel("Email address").fill(email);
  await wait(300);
  await page.getByLabel("Password").fill(password);
  await wait(400);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForURL("**/dashboard", { timeout: 25_000 });
  await page.waitForSelector("aside", { timeout: 15_000 });
  await wait(1500);
}

async function hoverCards(page: Page, labels: string[], hoverEach: number) {
  for (const label of labels) {
    const el = page.locator("a, div, h3, span").filter({ hasText: new RegExp(`^${label}`, "i") }).first();
    if (await el.isVisible().catch(() => false)) {
      await el.hover();
      await wait(hoverEach);
    }
  }
}

// ─────────────────────────────────────────────────────────────────────────────

test("OpsOwl AI — Narrated Product Demo", async ({ page }) => {
  test.skip(process.env.DEMO_MODE !== "true", "Demo recording — run with DEMO_MODE=true and audio assets present");
  test.setTimeout(3_600_000);

  // Pre-compute all scene durations (based on audio files)
  const DUR = {
    s01: playScene("s01_login"),
    s02: playScene("s02_dashboard"),
    s03: playScene("s03_cases"),
    s04: playScene("s04_case_detail"),
    s05: playScene("s05_tasks"),
    s06: playScene("s06_documents"),
    s07: playScene("s07_notes"),
    s08: playScene("s08_timeline"),
    s09: playScene("s09_copilot"),
    s10: playScene("s10_ai_logs"),
    s11: playScene("s11_sla"),
    s12: playScene("s12_audit"),
    s13: playScene("s13_analyst_divya"),
    s14: playScene("s14_manager_ravi"),
    s15: playScene("s15_users"),
    s16: playScene("s16_search"),
    s17: playScene("s17_close"),
  };

  // Kill any lingering afplay from pre-computing
  try { execSync("pkill -x afplay 2>/dev/null || true"); } catch { /* ok */ }
  await wait(500);

  // ── SCENE 1: Login (Admin — Priya Sharma) ───────────────────────────────────
  const d1 = playScene("s01_login");
  await scene("login", d1, async () => {
    await page.goto("/login");
    await wait(1000);
    await page.getByLabel("Email address").click();
    await page.getByLabel("Email address").pressSequentially("admin@opsowl.dev", { delay: 90 });
    await wait(700);
    await page.getByLabel("Password").click();
    await page.getByLabel("Password").pressSequentially("OpsOwl@2024", { delay: 90 });
    await wait(800);
    await page.getByRole("button", { name: "Sign in" }).click();
    await page.waitForURL("**/dashboard", { timeout: 25_000 });
    await page.waitForSelector("aside", { timeout: 15_000 });
  });

  // ── SCENE 2: Dashboard ───────────────────────────────────────────────────────
  const d2 = playScene("s02_dashboard");
  await scene("dashboard", d2, async () => {
    await wait(1500);
    await hoverCards(page, ["Active Cases", "Pending Tasks", "AI Queries", "SLA Breached"], 4000);
    await page.mouse.wheel(0, 380);
    await wait(4000);
    const statusChart = page.locator("text=Case Status Breakdown").first();
    if (await statusChart.isVisible().catch(() => false)) {
      await statusChart.hover(); await wait(5000);
    }
    const prioritySection = page.locator("text=Active Cases by Priority").first();
    if (await prioritySection.isVisible().catch(() => false)) {
      await prioritySection.hover(); await wait(5000);
    }
    const attn = page.locator("text=Attention Required").first();
    if (await attn.isVisible().catch(() => false)) {
      await attn.hover(); await wait(4000);
    }
    await page.mouse.wheel(0, -380);
    await wait(2000);
  });

  // ── SCENE 3: Cases List ──────────────────────────────────────────────────────
  const d3 = playScene("s03_cases");
  await scene("cases", d3, async () => {
    await page.goto("/cases");
    await page.waitForLoadState("networkidle");
    await wait(2500);
    await page.mouse.wheel(0, 250); await wait(2000);
    await page.mouse.wheel(0, 250); await wait(2000);
    await page.mouse.wheel(0, -500); await wait(1500);

    // Filter by critical
    const allSelects = page.locator("select");
    const n = await allSelects.count();
    for (let i = 0; i < n; i++) {
      const sel = allSelects.nth(i);
      const opts = await sel.locator("option").allTextContents().catch(() => []);
      if (opts.some((o) => /critical/i.test(o))) {
        await sel.selectOption({ label: "Critical" });
        await wait(5000);
        await sel.selectOption({ index: 0 });
        await wait(2000);
        break;
      }
    }
    const searchInput = page.locator("input[type='text'], input[type='search']").first();
    if (await searchInput.isVisible().catch(() => false)) {
      await searchInput.click();
      await searchInput.pressSequentially("UPI", { delay: 120 }); await wait(3000);
      await searchInput.clear(); await wait(600);
      await searchInput.pressSequentially("insider", { delay: 120 }); await wait(3000);
      await searchInput.clear(); await wait(800);
    }
  });

  // ── SCENE 4: Flagship Case — UPI Cluster ────────────────────────────────────
  const d4 = playScene("s04_case_detail");
  // Get case ID
  const token = (await page.evaluate(async () => {
    const r = await fetch("http://localhost:8080/api/v1/auth/login", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: "admin@opsowl.dev", password: "OpsOwl@2024" }),
    });
    return ((await r.json()) as { data: { access_token: string } }).data.access_token;
  })) as string;

  const casesData = await page.evaluate(async (t) => {
    const r = await fetch("http://localhost:8080/api/v1/cases?limit=50", {
      headers: { Authorization: `Bearer ${t}` },
    });
    return r.json();
  }, token) as { data: Array<{ id: string; title: string; priority: string }> };

  const flagshipCase = casesData.data.find((c) => /UPI.*Cluster/i.test(c.title))
    ?? casesData.data.find((c) => c.priority === "critical")
    ?? casesData.data[0];
  const caseBase = `/cases/${flagshipCase.id}`;

  await scene("case-detail", d4, async () => {
    await page.goto(caseBase);
    await page.waitForLoadState("networkidle");
    await wait(3000);

    const h1 = page.locator("h1, h2").first();
    if (await h1.isVisible().catch(() => false)) { await h1.hover(); await wait(4000); }

    const critBadge = page.locator("span, div, [class*='badge']").filter({ hasText: /critical/i }).first();
    if (await critBadge.isVisible().catch(() => false)) { await critBadge.hover(); await wait(4000); }

    await page.mouse.wheel(0, 350); await wait(5000);
    await page.mouse.wheel(0, 350); await wait(5000);
    await page.mouse.wheel(0, -700); await wait(3000);
  });

  // ── SCENE 5: Tasks ───────────────────────────────────────────────────────────
  const d5 = playScene("s05_tasks");
  await scene("tasks", d5, async () => {
    await page.goto(`${caseBase}/tasks`);
    await page.waitForURL(/\/tasks/, { timeout: 15_000 });
    await page.waitForLoadState("networkidle");
    await wait(3000);

    const rows = page.locator("tbody tr, [class*='task']");
    const count = await rows.count();
    for (let i = 0; i < Math.min(count, 5); i++) {
      const row = rows.nth(i);
      if (await row.isVisible().catch(() => false)) { await row.hover(); await wait(5000); }
    }

    const aiBadge = page.locator("span").filter({ hasText: /AI.suggested/i }).first();
    if (await aiBadge.isVisible().catch(() => false)) { await aiBadge.hover(); await wait(5000); }

    await page.mouse.wheel(0, 300); await wait(3000);
    await page.mouse.wheel(0, -300); await wait(2000);
  });

  // ── SCENE 6: Documents ───────────────────────────────────────────────────────
  const d6 = playScene("s06_documents");
  await scene("documents", d6, async () => {
    await page.goto(`${caseBase}/documents`);
    await page.waitForURL(/\/documents/, { timeout: 15_000 });
    await page.waitForLoadState("networkidle");
    await wait(3000);

    const docRows = page.locator("tbody tr, li");
    const count = await docRows.count();
    for (let i = 0; i < Math.min(count, 4); i++) {
      const row = docRows.nth(i);
      if (await row.isVisible().catch(() => false)) { await row.hover(); await wait(6000); }
    }

    await page.mouse.wheel(0, 250); await wait(5000);
    await page.mouse.wheel(0, -250); await wait(3000);
  });

  // ── SCENE 7: Notes ───────────────────────────────────────────────────────────
  const d7 = playScene("s07_notes");
  await scene("notes", d7, async () => {
    await page.goto(`${caseBase}/notes`);
    await page.waitForURL(/\/notes/, { timeout: 15_000 });
    await page.waitForLoadState("networkidle");
    await wait(3000);

    const note1 = page.locator("article, [class*='card'], [class*='note']").first();
    if (await note1.isVisible().catch(() => false)) { await note1.hover(); await wait(10000); }

    await page.mouse.wheel(0, 400); await wait(2000);
    const note2 = page.locator("article, [class*='card'], [class*='note']").nth(1);
    if (await note2.isVisible().catch(() => false)) { await note2.hover(); await wait(10000); }

    await page.mouse.wheel(0, 300); await wait(4000);
    await page.mouse.wheel(0, -700); await wait(3000);
  });

  // ── SCENE 8: Timeline ────────────────────────────────────────────────────────
  const d8 = playScene("s08_timeline");
  await scene("timeline", d8, async () => {
    await page.goto(`${caseBase}/timeline`);
    await page.waitForURL(/\/timeline/, { timeout: 15_000 });
    await page.waitForLoadState("networkidle");
    await wait(5000); // SSE events load

    for (let i = 0; i < 6; i++) {
      await page.mouse.wheel(0, 200); await wait(3500);
    }
    await page.mouse.wheel(0, -1200); await wait(3000);
  });

  // ── SCENE 9: AI Copilot ──────────────────────────────────────────────────────
  const d9 = playScene("s09_copilot");
  await scene("copilot", d9, async () => {
    await page.goto(caseBase);
    await page.waitForLoadState("networkidle");
    await wait(2500);

    const copilotBtn = page.getByRole("button", { name: /AI Copilot/i });
    if (await copilotBtn.isVisible().catch(() => false)) {
      await copilotBtn.hover(); await wait(1000);
      await copilotBtn.click();
      await page.waitForSelector('[data-testid="copilot-panel"]', { timeout: 10_000 });
      await wait(2000);

      const panel = page.locator('[data-testid="copilot-panel"]');
      const input = panel.getByPlaceholder(/Ask.*question/i);

      if (await input.isVisible().catch(() => false)) {
        await input.click();
        await input.pressSequentially(
          "What are the AML red flags in this case? Should we file an STR with FIU-IND?",
          { delay: 60 }
        );
        await wait(1200);
        const sendBtn = panel.getByRole("button", { name: /send/i });
        if (await sendBtn.isVisible().catch(() => false)) {
          await sendBtn.click();
          await wait(14000); // wait for AI response
        }

        // Scroll panel to show full response
        const panelEl = await panel.elementHandle();
        if (panelEl) { await page.evaluate((el) => { el.scrollTop += 500; }, panelEl); }
        await wait(8000);

        // Second question — task extraction
        const input2 = panel.getByPlaceholder(/Ask.*question/i);
        if (await input2.isVisible().catch(() => false)) {
          await input2.click();
          await input2.pressSequentially(
            "What specific investigation tasks should I create for this case?",
            { delay: 60 }
          );
          await wait(1000);
          const send2 = panel.getByRole("button", { name: /send/i });
          if (await send2.isVisible().catch(() => false)) {
            await send2.click();
            await wait(12000);
          }
          if (panelEl) { await page.evaluate((el) => { el.scrollTop += 400; }, panelEl); }
          await wait(5000);
        }
      }
    }
    await page.keyboard.press("Escape");
    await wait(1500);
  });

  // ── SCENE 10: AI Logs ────────────────────────────────────────────────────────
  const d10 = playScene("s10_ai_logs");
  await scene("ai-logs", d10, async () => {
    await page.goto("/ai-logs");
    await page.waitForLoadState("networkidle");
    await wait(3000);

    const approvedRows = page.locator("tr").filter({ hasText: /approved/i });
    const ac = await approvedRows.count();
    for (let i = 0; i < Math.min(ac, 2); i++) {
      if (await approvedRows.nth(i).isVisible().catch(() => false)) {
        await approvedRows.nth(i).hover(); await wait(6000);
      }
    }

    const pendingRow = page.locator("tr").filter({ hasText: /pending/i }).first();
    if (await pendingRow.isVisible().catch(() => false)) {
      await pendingRow.hover(); await wait(6000);
    }

    const firstRow = page.locator("tbody tr").first();
    if (await firstRow.isVisible().catch(() => false)) {
      await firstRow.click().catch(() => {});
      await wait(5000);
      await page.goBack().catch(() => {});
      await page.waitForLoadState("networkidle").catch(() => {});
      await wait(2000);
    }

    await page.mouse.wheel(0, 300); await wait(3000);
    await page.mouse.wheel(0, -300); await wait(2000);
  });

  // ── SCENE 11: SLA Policies ───────────────────────────────────────────────────
  const d11 = playScene("s11_sla");
  await scene("sla", d11, async () => {
    await page.goto("/sla");
    await page.waitForLoadState("networkidle");
    await wait(3000);

    const rows = page.locator("tr, li, [class*='policy']");
    const count = await rows.count();
    for (let i = 1; i <= Math.min(count, 4); i++) {
      const row = rows.nth(i);
      if (await row.isVisible().catch(() => false)) { await row.hover(); await wait(8000); }
    }

    await page.mouse.wheel(0, 300); await wait(4000);
    await page.mouse.wheel(0, -300); await wait(3000);
  });

  // ── SCENE 12: Audit Log ──────────────────────────────────────────────────────
  const d12 = playScene("s12_audit");
  await scene("audit", d12, async () => {
    await page.goto("/audit");
    await page.waitForLoadState("networkidle");
    await wait(3000);

    const critRow = page.locator("tr").filter({ hasText: /critical/i }).first();
    if (await critRow.isVisible().catch(() => false)) { await critRow.hover(); await wait(7000); }

    const warnRow = page.locator("tr").filter({ hasText: /fail|warn/i }).first();
    if (await warnRow.isVisible().catch(() => false)) { await warnRow.hover(); await wait(7000); }

    for (let i = 0; i < 4; i++) {
      await page.mouse.wheel(0, 300); await wait(3000);
    }
    await page.mouse.wheel(0, -1200); await wait(3000);
  });

  // ── SCENE 13: Analyst Persona — Divya Reddy ─────────────────────────────────
  const d13 = playScene("s13_analyst_divya");
  await scene("analyst", d13, async () => {
    await loginAs(page, "divya.reddy@opsowl.dev", "OpsOwl@2024");
    await wait(2000);
    await hoverCards(page, ["Active Cases", "Pending Tasks", "AI Queries"], 3000);
    await page.mouse.wheel(0, 250); await wait(3000);
    await page.mouse.wheel(0, -250); await wait(2000);
    await page.goto("/cases");
    await page.waitForLoadState("networkidle");
    await wait(4000);
    await page.mouse.wheel(0, 200); await wait(2000);
    await page.mouse.wheel(0, -200);
  });

  // ── SCENE 14: Manager Persona — Ravi Kumar ───────────────────────────────────
  const d14 = playScene("s14_manager_ravi");
  await scene("manager", d14, async () => {
    await loginAs(page, "ravi.kumar@opsowl.dev", "OpsOwl@2024");
    await wait(2000);
    await hoverCards(page, ["Active Cases", "Pending Tasks"], 3500);
    await page.mouse.wheel(0, 300); await wait(3000);
    await page.mouse.wheel(0, -300); await wait(2000);
    await page.goto("/cases?status=pending_review");
    await page.waitForLoadState("networkidle");
    await wait(4000);
    const pendingLink = page.locator("a[href*='/cases/']").first();
    if (await pendingLink.isVisible().catch(() => false)) {
      await pendingLink.hover(); await wait(3000);
    }
  });

  // ── SCENE 15: User Management / RBAC (back to admin) ─────────────────────────
  const d15 = playScene("s15_users");
  await scene("users", d15, async () => {
    await loginAs(page, "admin@opsowl.dev", "OpsOwl@2024");
    await page.goto("/users");
    await page.waitForLoadState("networkidle");
    await wait(2500);

    const userRows = page.locator("tbody tr, [role='row']").filter({ hasText: /@opsowl\.dev/i });
    const uCount = await userRows.count();
    for (let i = 0; i < Math.min(uCount, 5); i++) {
      const row = userRows.nth(i);
      if (await row.isVisible().catch(() => false)) { await row.hover(); await wait(5500); }
    }
    await page.mouse.wheel(0, 200); await wait(3000);
    await page.mouse.wheel(0, -200); await wait(2000);
  });

  // ── SCENE 16: Search ─────────────────────────────────────────────────────────
  const d16 = playScene("s16_search");
  await scene("search", d16, async () => {
    await page.goto("/cases");
    await page.waitForLoadState("networkidle");
    await wait(2000);

    const inp = page.locator("input[type='text'], input[type='search']").first();
    if (await inp.isVisible().catch(() => false)) {
      for (const term of ["STR filing", "NACH mandate", "ATM skimming", "insider"]) {
        await inp.click();
        await inp.pressSequentially(term, { delay: 110 });
        await wait(3500);
        await inp.clear(); await wait(500);
      }
    }
  });

  // ── SCENE 17: Dashboard — Closing Shot ───────────────────────────────────────
  const d17 = playScene("s17_close");
  await scene("close", d17, async () => {
    await page.goto("/dashboard");
    await page.waitForLoadState("networkidle");
    await wait(2500);
    await hoverCards(page, ["Active Cases", "Pending Tasks", "AI Queries", "SLA"], 3000);
    await page.mouse.wheel(0, 350); await wait(4000);
    await page.mouse.wheel(0, 350); await wait(4000);
    await page.mouse.wheel(0, -700); await wait(5000);
  });

  expect(true).toBe(true);
});
