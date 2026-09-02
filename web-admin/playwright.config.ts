import { defineConfig, devices } from "@playwright/test";

const e2ePort = Number(process.env.E2E_WEB_PORT || 3133);
const baseURL = process.env.E2E_BASE_URL || `http://127.0.0.1:${e2ePort}`;

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [["line"], ["html", { open: "never" }]] : "line",
  timeout: 60_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: `npm run dev -- --host 127.0.0.1 --port ${e2ePort}`,
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: {
      ...process.env,
      QUIET_START: "1",
      POWERX_PROVIDER_MODE: "local",
      NUXT_PUBLIC_POWERX_PROVIDER_MODE: "local",
      POWERX_PROXY: "0",
      NUXT_PUBLIC_POWERX_PROXY: "0",
      NUXT_PUBLIC_API_BASE: "http://127.0.0.1:8078",
      NUXT_PUBLIC_API_PREFIX: "/api/v1",
    },
  },
});
