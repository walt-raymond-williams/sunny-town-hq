import { authFetch } from '../auth'

export async function readJson<T>(response: Response): Promise<T> {
  const text = await response.text()
  let body: (T & { error?: string }) | null = null

  if (text) {
    try {
      body = JSON.parse(text) as T & { error?: string }
    } catch {
      if (!response.ok) {
        throw new Error(text || `API returned ${response.status}`)
      }
      throw new Error('API returned invalid JSON')
    }
  }

  if (!response.ok) {
    throw new Error(body?.error || `API returned ${response.status}`)
  }

  if (!body) {
    throw new Error('API returned an empty response')
  }

  return body
}

export async function authJson<T>(url: string, options: RequestInit = {}): Promise<T> {
  return readJson<T>(await authFetch(url, options))
}

export function jsonOptions<TBody>(method: string, body: TBody): RequestInit {
  return {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  }
}
