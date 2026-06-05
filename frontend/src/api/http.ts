import { authFetch } from '../auth'

export async function readJson<T>(response: Response): Promise<T> {
  const body = (await response.json()) as T & { error?: string }

  if (!response.ok) {
    throw new Error(body.error || `API returned ${response.status}`)
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
