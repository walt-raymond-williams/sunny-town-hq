import { defineConfig, devices } from '@playwright/test'
import { execFileSync } from 'node:child_process'

const baseURL = process.env.PLAYWRIGHT_BASE_URL || defaultBaseURL()

function defaultBaseURL(): string {
  const publicHost = process.env.HQ_PUBLIC_HOST || publicHostFromRunningHQ() || 'localhost'
  return `http://${publicHost}:18080`
}

function publicHostFromRunningHQ(): string {
  try {
    const output = execFileSync(
      'docker',
      ['inspect', 'hq-server', '--format', '{{range .Config.Env}}{{println .}}{{end}}'],
      { encoding: 'utf8' },
    )
    const issuer = output
      .split(/\r?\n/)
      .find((line) => line.startsWith('KEYCLOAK_ISSUER='))
      ?.replace('KEYCLOAK_ISSUER=', '')
    return issuer ? new URL(issuer).hostname : ''
  } catch {
    return ''
  }
}

export default defineConfig({
  testDir: './e2e',
  timeout: 45_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  globalSetup: './e2e/global-setup.ts',
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
