import type { Page } from '@playwright/test'

export type TestRole = 'student' | 'teacher'

const credentials: Record<TestRole, { username: string; password: string }> = {
  student: { username: 'playwright-student', password: 'playwright' },
  teacher: { username: 'playwright-teacher', password: 'playwright' },
}

export function keycloakURLFor(baseURL: string): string {
  const url = new URL(baseURL)
  url.port = process.env.PLAYWRIGHT_KEYCLOAK_PORT || '18081'
  return (process.env.PLAYWRIGHT_KEYCLOAK_URL || url.origin).replace(/\/$/, '')
}

export async function loginAs(page: Page, role: TestRole, baseURL: string) {
  const tokens = await getToken(role, baseURL)
  await page.addInitScript((authTokens) => {
    window.sessionStorage.setItem('hq.auth.tokens', JSON.stringify(authTokens))
  }, tokens)
}

export async function hqFetch<T>(
  role: TestRole,
  baseURL: string,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const tokens = await getToken(role, baseURL)
  const response = await fetch(new URL(path, baseURL), {
    ...init,
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
      'Content-Type': 'application/json',
      ...init.headers,
    },
  })
  if (!response.ok) {
    throw new Error(`${init.method || 'GET'} ${path} failed: ${response.status} ${await response.text()}`)
  }
  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}

async function getToken(role: TestRole, baseURL: string): Promise<Record<string, unknown> & { access_token: string }> {
  const user = credentials[role]
  const response = await fetch(`${keycloakURLFor(baseURL)}/realms/hq/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      client_id: 'hq-web',
      grant_type: 'password',
      username: user.username,
      password: user.password,
      scope: 'openid profile email',
    }),
  })
  if (!response.ok) {
    throw new Error(`Token request for ${role} failed: ${response.status} ${await response.text()}`)
  }
  const body = await response.json() as Record<string, unknown> & { access_token?: string }
  if (!body.access_token) {
    throw new Error(`Token request for ${role} did not return access_token`)
  }
  return body as Record<string, unknown> & { access_token: string }
}
