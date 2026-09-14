import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*.real.spec.js",
  fullyParallel: false,
  workers: 1,
  forbidOnly: true,
  retries: 0,
  timeout: 180_000,
  reporter: "list",
  expect: { timeout: 30_000 },
  use: {
    baseURL: process.env.REAL_E2E_BASE_URL || "http://127.0.0.1:3200",
    headless: false,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium-real", use: { ...devices["Desktop Chrome"] } }],
});
