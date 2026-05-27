import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 1,
  workers: 1,
  reporter: [["list"], ["html", { open: "never", outputFolder: "playwright-report" }]],
  expect: {
    timeout: 10_000, // raise default from 5s to 10s for SPA auth-refresh timing
  },
  use: {
    baseURL: "http://localhost:3001",
    screenshot: "only-on-failure",
    video: "on-first-retry",
    trace: "on-first-retry",
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  // Global setup creates a reusable auth state
  globalSetup: "./tests/e2e/helpers/global-setup.ts",
});
