import { defineConfig, devices } from '@playwright/test';

const reactUrl = process.env.REACT_URL ?? 'http://localhost:5173';
const vueUrl = process.env.VUE_URL ?? 'http://localhost:5174';
const vanillaUrl = process.env.VANILLA_URL ?? 'http://localhost:5175';

export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  retries: 0,
  reporter: [['list']],
  use: {
    headless: true,
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
    ignoreHTTPSErrors: true,
  },
  projects: [
    {
      name: 'react',
      use: { ...devices['Desktop Chrome'], baseURL: reactUrl },
    },
    {
      name: 'vue',
      use: { ...devices['Desktop Chrome'], baseURL: vueUrl },
    },
    {
      name: 'vanilla',
      use: { ...devices['Desktop Chrome'], baseURL: vanillaUrl },
    },
  ],
});
