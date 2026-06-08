import type { FullConfig } from '@playwright/test'

const users = [
  {
    username: 'playwright-student',
    firstName: 'Playwright',
    lastName: 'Student',
    email: 'playwright-student@example.test',
    role: 'student',
  },
  {
    username: 'playwright-teacher',
    firstName: 'Playwright',
    lastName: 'Teacher',
    email: 'playwright-teacher@example.test',
    role: 'teacher',
  },
]
const password = 'playwright'

export default async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0]?.use.baseURL?.toString() || 'http://localhost:18080'
  const keycloakURL = process.env.PLAYWRIGHT_KEYCLOAK_URL || keycloakURLFor(baseURL)
  await waitForKeycloak(keycloakURL)

  const adminToken = await getAdminToken(keycloakURL)
  await ensureClientConfiguration(keycloakURL, adminToken, baseURL)
  for (const user of users) {
    await ensureUser(keycloakURL, adminToken, user)
  }
}

function keycloakURLFor(baseURL: string): string {
  const url = new URL(baseURL)
  url.port = process.env.PLAYWRIGHT_KEYCLOAK_PORT || '18081'
  return url.origin
}

async function waitForKeycloak(keycloakURL: string) {
  const deadline = Date.now() + 60_000
  let lastError = ''
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${keycloakURL}/realms/hq/.well-known/openid-configuration`)
      if (response.ok) {
        return
      }
      lastError = `${response.status} ${response.statusText}`
    } catch (error) {
      lastError = error instanceof Error ? error.message : String(error)
    }
    await new Promise((resolve) => setTimeout(resolve, 1_000))
  }
  throw new Error(`Keycloak is not ready at ${keycloakURL}: ${lastError}`)
}

async function getAdminToken(keycloakURL: string): Promise<string> {
  const response = await fetch(`${keycloakURL}/realms/master/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      client_id: 'admin-cli',
      grant_type: 'password',
      username: process.env.PLAYWRIGHT_KEYCLOAK_ADMIN || 'admin',
      password: process.env.PLAYWRIGHT_KEYCLOAK_ADMIN_PASSWORD || 'admin',
    }),
  })
  if (!response.ok) {
    throw new Error(`Keycloak admin login failed: ${response.status} ${await response.text()}`)
  }
  const body = await response.json() as { access_token?: string }
  if (!body.access_token) {
    throw new Error('Keycloak admin login did not return an access token')
  }
  return body.access_token
}

async function ensureClientConfiguration(keycloakURL: string, token: string, baseURL: string) {
  const clients = await keycloakAdminFetch<Array<{ id: string }>>(
    keycloakURL,
    token,
    '/clients?clientId=hq-web',
  )
  const client = clients[0]
  if (!client) {
    throw new Error('Keycloak hq-web client was not found')
  }

  const representation = await keycloakAdminFetch<Record<string, unknown>>(
    keycloakURL,
    token,
    `/clients/${client.id}`,
  )
  const protocolMappers = Array.isArray(representation.protocolMappers)
    ? representation.protocolMappers as Array<Record<string, unknown>>
    : []
  const redirectUris = Array.isArray(representation.redirectUris)
    ? representation.redirectUris as string[]
    : []
  const webOrigins = Array.isArray(representation.webOrigins)
    ? representation.webOrigins as string[]
    : []
  const appOrigin = new URL(baseURL).origin
  const hasAudienceMapper = protocolMappers.some((mapper) => mapper.name === 'hq-web audience')
  const nextRedirectUris = includeString(redirectUris, `${appOrigin}/*`)
  const nextWebOrigins = includeString(webOrigins, appOrigin)
  if (
    representation.directAccessGrantsEnabled === true &&
    hasAudienceMapper &&
    nextRedirectUris === redirectUris &&
    nextWebOrigins === webOrigins
  ) {
    return
  }
  await keycloakAdminFetch(keycloakURL, token, `/clients/${client.id}`, {
    method: 'PUT',
    body: JSON.stringify({
      ...representation,
      directAccessGrantsEnabled: true,
      redirectUris: nextRedirectUris,
      webOrigins: nextWebOrigins,
      protocolMappers: hasAudienceMapper
        ? protocolMappers
        : [
            ...protocolMappers,
            {
              name: 'hq-web audience',
              protocol: 'openid-connect',
              protocolMapper: 'oidc-audience-mapper',
              consentRequired: false,
              config: {
                'included.client.audience': 'hq-web',
                'id.token.claim': 'false',
                'access.token.claim': 'true',
              },
            },
          ],
    }),
  })
}

function includeString(values: string[], value: string): string[] {
  return values.includes(value) ? values : [...values, value]
}

async function ensureUser(
  keycloakURL: string,
  token: string,
  user: {
    username: string
    firstName: string
    lastName: string
    email: string
    role: string
  },
) {
  const matches = await keycloakAdminFetch<Array<{ id: string }>>(
    keycloakURL,
    token,
    `/users?username=${encodeURIComponent(user.username)}&exact=true`,
  )
  let userID = matches[0]?.id
  if (!userID) {
    const response = await keycloakAdminRaw(keycloakURL, token, '/users', {
      method: 'POST',
      body: JSON.stringify({
        username: user.username,
        firstName: user.firstName,
        lastName: user.lastName,
        email: user.email,
        enabled: true,
        emailVerified: true,
      }),
    })
    if (response.status !== 201) {
      throw new Error(`Create Keycloak user ${user.username} failed: ${response.status} ${await response.text()}`)
    }
    const location = response.headers.get('location') || ''
    userID = location.split('/').pop() || ''
  } else {
    await keycloakAdminFetch(keycloakURL, token, `/users/${userID}`, {
      method: 'PUT',
      body: JSON.stringify({
        username: user.username,
        firstName: user.firstName,
        lastName: user.lastName,
        email: user.email,
        enabled: true,
        emailVerified: true,
      }),
    })
  }

  await keycloakAdminFetch(keycloakURL, token, `/users/${userID}/reset-password`, {
    method: 'PUT',
    body: JSON.stringify({
      type: 'password',
      value: password,
      temporary: false,
    }),
  })

  const role = await keycloakAdminFetch<{ id: string; name: string }>(
    keycloakURL,
    token,
    `/roles/${user.role}`,
  )
  await keycloakAdminFetch(keycloakURL, token, `/users/${userID}/role-mappings/realm`, {
    method: 'POST',
    body: JSON.stringify([role]),
  })
}

async function keycloakAdminFetch<T = unknown>(
  keycloakURL: string,
  token: string,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await keycloakAdminRaw(keycloakURL, token, path, init)
  if (!response.ok) {
    throw new Error(`Keycloak admin ${path} failed: ${response.status} ${await response.text()}`)
  }
  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}

function keycloakAdminRaw(
  keycloakURL: string,
  token: string,
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  return fetch(`${keycloakURL}/admin/realms/hq${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      ...init.headers,
    },
  })
}
