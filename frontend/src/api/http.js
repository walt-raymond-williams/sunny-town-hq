import { authFetch } from '../auth'

export async function readJson(response) {
  const body = await response.json()

  if (!response.ok) {
    throw new Error(body.error || `API returned ${response.status}`)
  }

  return body
}

export async function authJson(url, options = {}) {
  return readJson(await authFetch(url, options))
}

export function jsonOptions(method, body) {
  return {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  }
}
