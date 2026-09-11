import { defineConfig, devices } from "@playwright/test";

const ci = Boolean(process.env.CI);

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: ci,
  workers: ci ? undefined : 1,
  forbidOnly: ci,
  retries: ci ? 1 : 0,
  reporter: ci ? "github" : "list",
  use: {
    baseURL: "http://127.0.0.1:3200",
    headless: ci,
    launchOptions: ci ? undefined : { slowMo: 120 },
    screenshot: "only-on-failure",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: "npm run dev",
    url: "http://127.0.0.1:3200",
    reuseExistingServer: !ci,
  },
});
