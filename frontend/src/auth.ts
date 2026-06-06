import type { Interceptor } from '@connectrpc/connect'

interface RoleClaim {
  roles?: string[]
}

interface TokenPayload {
  exp?: number
  realm_access?: RoleClaim
  resource_access?: Record<string, RoleClaim>
}

interface AuthTokens {
  access_token?: string | null
  refresh_token?: string | null
  id_token?: string | null
  token_type?: string | null
  expires_in?: number
}

interface LoginOptions {
  redirectUri?: string
}

interface StoredAuthState {
  state: string
  redirectUri: string
  codeVerifier: string
}

interface CodeChallenge {
  value: string
  method: 'S256' | 'plain'
}

interface HqKeycloak {
  authenticated: boolean
  token: string
  refreshToken: string
  tokenParsed: TokenPayload | null
  idToken: string
  idTokenParsed: TokenPayload | null
  hasRealmRole(role: string): boolean
  hasResourceRole(role: string, resource?: string): boolean
  login(options?: LoginOptions): Promise<void>
  logout(options?: LoginOptions): Promise<void>
  updateToken(minValidity?: number): Promise<boolean>
}

function defaultKeycloakUrl(): string {
  const configured = import.meta.env.VITE_KEYCLOAK_URL
  if (configured) {
    return configured
  }

  return `${window.location.protocol}//${window.location.hostname}:18081`
}

const keycloakUrl = defaultKeycloakUrl()
const keycloakRealm = import.meta.env.VITE_KEYCLOAK_REALM || 'hq'
export const keycloakClientId = import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'hq-web'
const realmUrl = `${keycloakUrl}/realms/${keycloakRealm}`
const tokenStorageKey = 'hq.auth.tokens'
const stateStorageKey = 'hq.auth.state'
const authRefreshIntervalMs = 30_000
let refreshPromise: Promise<boolean> | null = null
let authRefreshTimer = 0

export const keycloak: HqKeycloak = {
  authenticated: false,
  token: '',
  refreshToken: '',
  tokenParsed: null,
  idToken: '',
  idTokenParsed: null,
  hasRealmRole(role) {
    return Boolean(this.tokenParsed?.realm_access?.roles?.includes(role))
  },
  hasResourceRole(role, resource = keycloakClientId) {
    return Boolean(this.tokenParsed?.resource_access?.[resource]?.roles?.includes(role))
  },
  async login(options = {}) {
    await startLogin(options.redirectUri || window.location.href)
  },
  async logout(options = {}) {
    clearTokens()
    const params = new URLSearchParams({
      client_id: keycloakClientId,
      post_logout_redirect_uri: options.redirectUri || window.location.origin,
    })
    window.location.assign(`${realmUrl}/protocol/openid-connect/logout?${params.toString()}`)
  },
  async updateToken(minValidity = 30) {
    if (!this.authenticated) {
      return false
    }
    if (!isTokenExpiring(this.tokenParsed, minValidity)) {
      return false
    }

    return refreshAccessToken()
  },
}

export async function initAuth(): Promise<boolean> {
  loadStoredTokens()

  const params = new URLSearchParams(window.location.search)
  const code = params.get('code')
  const state = params.get('state')
  const hashParams = new URLSearchParams(window.location.hash.replace(/^#/, ''))
  const accessToken = hashParams.get('access_token')
  const hashState = hashParams.get('state')
  if (accessToken || hashState) {
    finishImplicitLogin(hashParams)
  } else if (code || state) {
    await finishLogin(code, state)
  }

  return keycloak.authenticated
}

export function hasRole(role: string): boolean {
  if (!keycloak.authenticated) {
    return false
  }

  return keycloak.hasRealmRole(role) || keycloak.hasResourceRole(role, keycloakClientId)
}

export async function loginForRole(_role: string, redirectUri = window.location.href): Promise<boolean> {
  await startLogin(redirectUri)
  return false
}

export async function logout(redirectUri = window.location.origin): Promise<void> {
  await keycloak.logout({ redirectUri })
}

export async function authHeaders(extraHeaders: HeadersInit = {}): Promise<Record<string, string>> {
  if (!keycloak.authenticated) {
    await startLogin(window.location.href)
    throw new Error('login required')
  }

  await keycloak.updateToken(30)
  return {
    ...headersToRecord(extraHeaders),
    Authorization: `Bearer ${keycloak.token}`,
  }
}

export async function authFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const headers = await authHeaders(options.headers || {})
  return fetch(url, {
    ...options,
    headers,
  })
}

export const connectAuthInterceptor: Interceptor = (next) => async (req) => {
  const headers = await authHeaders()
  Object.entries(headers).forEach(([key, value]) => {
    req.header.set(key, value)
  })
  return next(req)
}

export function startAuthRefreshLoop(): void {
  stopAuthRefreshLoop()
  window.addEventListener('visibilitychange', refreshAuthOnVisible)
  authRefreshTimer = window.setInterval(() => {
    void refreshAuthenticatedSession()
  }, authRefreshIntervalMs)
  void refreshAuthenticatedSession()
}

export function stopAuthRefreshLoop(): void {
  if (authRefreshTimer) {
    window.clearInterval(authRefreshTimer)
    authRefreshTimer = 0
  }
  window.removeEventListener('visibilitychange', refreshAuthOnVisible)
}

async function startLogin(redirectUri: string): Promise<void> {
  const state = createLocalState()
  const codeVerifier = createCodeVerifier()
  const codeChallenge = await createCodeChallenge(codeVerifier)
  sessionStorage.setItem(
    stateStorageKey,
    JSON.stringify({
      state,
      redirectUri,
      codeVerifier,
    }),
  )

  const params = new URLSearchParams({
    client_id: keycloakClientId,
    redirect_uri: redirectUri,
    response_type: 'code',
    scope: 'openid profile email',
    state,
    code_challenge: codeChallenge.value,
    code_challenge_method: codeChallenge.method,
  })
  window.location.assign(`${realmUrl}/protocol/openid-connect/auth?${params.toString()}`)
}

async function finishLogin(code: string | null, state: string | null): Promise<void> {
  const storedState = readStoredState()
  if (!code || !state || !storedState || state !== storedState.state) {
    clearStoredState()
    clearTokens()
    throw new Error('invalid login callback')
  }

  const response = await fetch(`${realmUrl}/protocol/openid-connect/token`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      client_id: keycloakClientId,
      code,
      redirect_uri: storedState.redirectUri,
      code_verifier: storedState.codeVerifier,
    }),
  })

  if (!response.ok) {
    clearStoredState()
    clearTokens()
    throw new Error(`token exchange failed: ${response.status}`)
  }

  const tokens = (await response.json()) as AuthTokens
  applyTokens(tokens)
  storeTokens(tokens)
  clearStoredState()
  window.history.replaceState({}, document.title, window.location.pathname + window.location.hash)
}

function finishImplicitLogin(hashParams: URLSearchParams): void {
  const storedState = readStoredState()
  const state = hashParams.get('state')
  if (!state || !storedState || state !== storedState.state) {
    clearStoredState()
    clearTokens()
    throw new Error('invalid login callback')
  }

  const tokens = {
    access_token: hashParams.get('access_token'),
    token_type: hashParams.get('token_type'),
    expires_in: Number(hashParams.get('expires_in') || 0),
  }
  applyTokens(tokens)
  storeTokens(tokens)
  clearStoredState()
  window.history.replaceState({}, document.title, window.location.pathname + window.location.search)
}

async function refreshAccessToken(): Promise<boolean> {
  if (refreshPromise) {
    return refreshPromise
  }

  if (!keycloak.refreshToken) {
    clearTokens()
    await startLogin(window.location.href)
    throw new Error('login required')
  }

  refreshPromise = refreshAccessTokenOnce().finally(() => {
    refreshPromise = null
  })
  return refreshPromise
}

async function refreshAccessTokenOnce(): Promise<boolean> {
  const previousRefreshToken = keycloak.refreshToken
  const response = await fetch(`${realmUrl}/protocol/openid-connect/token`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: new URLSearchParams({
      grant_type: 'refresh_token',
      client_id: keycloakClientId,
      refresh_token: previousRefreshToken,
    }),
  })

  if (!response.ok) {
    clearTokens()
    await startLogin(window.location.href)
    throw new Error(`token refresh failed: ${response.status}`)
  }

  const tokens = (await response.json()) as AuthTokens
  if (!tokens.refresh_token) {
    tokens.refresh_token = previousRefreshToken
  }
  applyTokens(tokens)
  storeTokens(tokens)
  return true
}

async function refreshAuthenticatedSession(): Promise<void> {
  if (!keycloak.authenticated) {
    return
  }
  await keycloak.updateToken(90)
}

function refreshAuthOnVisible(): void {
  if (document.visibilityState === 'visible') {
    void refreshAuthenticatedSession()
  }
}

function applyTokens(tokens: AuthTokens): void {
  keycloak.token = tokens.access_token || ''
  keycloak.refreshToken = tokens.refresh_token || ''
  keycloak.idToken = tokens.id_token || ''
  keycloak.tokenParsed = keycloak.token ? parseJWT(keycloak.token) : null
  keycloak.idTokenParsed = keycloak.idToken ? parseJWT(keycloak.idToken) : null
  keycloak.authenticated = Boolean(keycloak.tokenParsed && !isTokenExpiring(keycloak.tokenParsed, 0))
}

function storeTokens(tokens: AuthTokens): void {
  sessionStorage.setItem(tokenStorageKey, JSON.stringify(tokens))
}

function loadStoredTokens(): void {
  const stored = sessionStorage.getItem(tokenStorageKey)
  if (!stored) {
    return
  }

  try {
    applyTokens(JSON.parse(stored))
    if (!keycloak.authenticated) {
      clearTokens()
    }
  } catch {
    clearTokens()
  }
}

function clearTokens(): void {
  keycloak.authenticated = false
  keycloak.token = ''
  keycloak.refreshToken = ''
  keycloak.idToken = ''
  keycloak.tokenParsed = null
  keycloak.idTokenParsed = null
  sessionStorage.removeItem(tokenStorageKey)
}

function readStoredState(): StoredAuthState | null {
  try {
    return JSON.parse(sessionStorage.getItem(stateStorageKey) || 'null')
  } catch {
    return null
  }
}

function clearStoredState(): void {
  sessionStorage.removeItem(stateStorageKey)
}

function parseJWT(token: string): TokenPayload {
  const [, payload] = token.split('.')
  if (!payload) {
    return {}
  }
  return JSON.parse(atob(base64UrlToBase64(payload)))
}

function isTokenExpiring(token: TokenPayload | null, minValiditySeconds: number): boolean {
  if (!token?.exp) {
    return true
  }

  const expiresAt = token.exp * 1000
  return expiresAt - Date.now() <= minValiditySeconds * 1000
}

function base64UrlToBase64(value: string): string {
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/')
  return base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
}

function createLocalState(): string {
  return `${Date.now().toString(36)}-${createRandomBase64Url(16)}`
}

function createCodeVerifier(): string {
  return createRandomBase64Url(32)
}

async function createCodeChallenge(verifier: string): Promise<CodeChallenge> {
  if (!window.crypto.subtle) {
    return {
      value: verifier,
      method: 'plain',
    }
  }

  const data = new TextEncoder().encode(verifier)
  const digest = await window.crypto.subtle.digest('SHA-256', data)
  return {
    value: bytesToBase64Url(new Uint8Array(digest)),
    method: 'S256',
  }
}

function createRandomBase64Url(byteLength: number): string {
  const bytes = new Uint8Array(byteLength)
  window.crypto.getRandomValues(bytes)
  return bytesToBase64Url(bytes)
}

function bytesToBase64Url(bytes: Uint8Array): string {
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function headersToRecord(headers: HeadersInit): Record<string, string> {
  if (headers instanceof Headers) {
    return Object.fromEntries(headers.entries())
  }

  if (Array.isArray(headers)) {
    return Object.fromEntries(headers)
  }

  return headers
}
