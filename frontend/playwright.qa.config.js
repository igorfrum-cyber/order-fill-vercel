import path from "node:path";
import { fileURLToPath } from "node:url";

import { defineConfig, devices } from "@playwright/test";

const ci = Boolean(process.env.CI);
const frontendDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(frontendDir, "..");
const live = Boolean(process.env.QA_LIVE);
const port = process.env.QA_PORT || (live ? "3200" : "3219");
const baseURL = process.env.QA_BASE_URL || `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*.smoke.spec.js",
  outputDir: path.join(repoRoot, "qa/artifacts/test-results"),
  fullyParallel: false,
  workers: 1,
  forbidOnly: ci,
  retries: 0,
  reporter: [
    ["list"],
    ["html", { outputFolder: path.join(repoRoot, "qa/artifacts/html-report"), open: "never" }],
  ],
  use: {
    baseURL,
    headless: ci,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: live
    ? undefined
    : {
        command: `npm run dev -- --port ${port}`,
        cwd: frontendDir,
        url: baseURL,
        reuseExistingServer: !ci,
      },
});
