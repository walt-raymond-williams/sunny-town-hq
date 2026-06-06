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
  method: 'S256'
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
  const error = params.get('error')
  const errorDescription = params.get('error_description')
  const code = params.get('code')
  const state = params.get('state')
  const hashParams = new URLSearchParams(window.location.hash.replace(/^#/, ''))
  const hashError = hashParams.get('error')
  const hashErrorDescription = hashParams.get('error_description')
  const accessToken = hashParams.get('access_token')
  const hashState = hashParams.get('state')
  if (error || errorDescription) {
    finishLoginError(errorDescription || error || 'login failed', state)
  } else if (hashError || hashErrorDescription) {
    finishLoginError(hashErrorDescription || hashError || 'login failed', hashState)
  } else if (accessToken || hashState) {
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

function finishLoginError(message: string, state: string | null): void {
  const storedState = readStoredState()
  if (!state || !storedState || state !== storedState.state) {
    clearStoredState()
    clearTokens()
    cleanLoginCallbackUrl()
    return
  }

  clearStoredState()
  clearTokens()
  sessionStorage.setItem('hq.auth.error', message)
  cleanLoginCallbackUrl()
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
  const data = new TextEncoder().encode(verifier)
  const digest = window.crypto.subtle
    ? new Uint8Array(await window.crypto.subtle.digest('SHA-256', data))
    : sha256(data)
  return {
    value: bytesToBase64Url(digest),
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

function cleanLoginCallbackUrl(): void {
  window.history.replaceState({}, document.title, window.location.pathname)
}

function sha256(message: Uint8Array): Uint8Array {
  const constants = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4,
    0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe,
    0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f,
    0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc,
    0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
    0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116,
    0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7,
    0xc67178f2,
  ]
  const hash = [
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab,
    0x5be0cd19,
  ]
  const bitLength = message.length * 8
  const paddedLength = (((message.length + 9 + 63) >> 6) << 6)
  const padded = new Uint8Array(paddedLength)
  padded.set(message)
  padded[message.length] = 0x80

  const view = new DataView(padded.buffer)
  view.setUint32(paddedLength - 8, Math.floor(bitLength / 0x100000000))
  view.setUint32(paddedLength - 4, bitLength)

  const words = new Uint32Array(64)
  for (let offset = 0; offset < paddedLength; offset += 64) {
    for (let index = 0; index < 16; index += 1) {
      words[index] = view.getUint32(offset + index * 4)
    }
    for (let index = 16; index < 64; index += 1) {
      const word15 = words[index - 15]!
      const word2 = words[index - 2]!
      const s0 = rotateRight(word15, 7) ^ rotateRight(word15, 18) ^ (word15 >>> 3)
      const s1 = rotateRight(word2, 17) ^ rotateRight(word2, 19) ^ (word2 >>> 10)
      words[index] = (words[index - 16]! + s0 + words[index - 7]! + s1) >>> 0
    }

    let a = hash[0]!
    let b = hash[1]!
    let c = hash[2]!
    let d = hash[3]!
    let e = hash[4]!
    let f = hash[5]!
    let g = hash[6]!
    let h = hash[7]!
    for (let index = 0; index < 64; index += 1) {
      const s1 = rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25)
      const ch = (e & f) ^ (~e & g)
      const temp1 = (h + s1 + ch + constants[index]! + words[index]!) >>> 0
      const s0 = rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22)
      const maj = (a & b) ^ (a & c) ^ (b & c)
      const temp2 = (s0 + maj) >>> 0
      h = g
      g = f
      f = e
      e = (d + temp1) >>> 0
      d = c
      c = b
      b = a
      a = (temp1 + temp2) >>> 0
    }

    hash[0] = (hash[0]! + a) >>> 0
    hash[1] = (hash[1]! + b) >>> 0
    hash[2] = (hash[2]! + c) >>> 0
    hash[3] = (hash[3]! + d) >>> 0
    hash[4] = (hash[4]! + e) >>> 0
    hash[5] = (hash[5]! + f) >>> 0
    hash[6] = (hash[6]! + g) >>> 0
    hash[7] = (hash[7]! + h) >>> 0
  }

  const digest = new Uint8Array(32)
  const digestView = new DataView(digest.buffer)
  hash.forEach((value, index) => {
    digestView.setUint32(index * 4, value)
  })
  return digest
}

function rotateRight(value: number, bits: number): number {
  return (value >>> bits) | (value << (32 - bits))
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
