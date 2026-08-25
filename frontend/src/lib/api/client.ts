import { clear, getAccessToken, getRefreshToken, setTokens } from '$lib/stores/auth.svelte'
import type { ApiError, RefreshResponse } from '$lib/types/api'

let onUnauthorized: (() => void) | null = null
let refreshInFlight: Promise<boolean> | null = null

export function setOnUnauthorized(cb: () => void) {
  onUnauthorized = cb
}

async function refreshTokens(): Promise<boolean> {
  const refreshTokenSnapshot = getRefreshToken()
  if (!refreshTokenSnapshot) return false
  if (refreshInFlight) return refreshInFlight

  refreshInFlight = (async () => {
    try {
      const res = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshTokenSnapshot }),
      })
      if (!res.ok) return false
      const data: RefreshResponse = await res.json()
      if (getRefreshToken() !== refreshTokenSnapshot) return false
      setTokens(data.access_token, data.refresh_token)
      return true
    } catch {
      return false
    }
  })()

  try {
    return await refreshInFlight
  } finally {
    refreshInFlight = null
  }
}

export class ApiErrorResponse extends Error {
  code: number
  errors?: { field: string; message: string }[]

  constructor(code: number, message: string, errors?: { field: string; message: string }[]) {
    super(message)
    this.code = code
    this.errors = errors
  }
}

export async function apiRequest<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  if (getAccessToken()) {
    headers['Authorization'] = `Bearer ${getAccessToken()}`
  }

  const requestBody = body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined

  let res = await fetch(`/api/v1${path}`, {
    method,
    headers,
    body: requestBody,
  })

  if (res.status === 401 && getRefreshToken()) {
    const refreshed = await refreshTokens()
    if (refreshed) {
      headers['Authorization'] = `Bearer ${getAccessToken()}`
      res = await fetch(`/api/v1${path}`, {
        method,
        headers,
        body: requestBody,
      })
      if (res.status === 401) {
        // A 401 after a successful refresh means the session is valid and the
        // endpoint rejected the request for business reasons (e.g. revoked
        // Bitbucket OAuth credentials), so surface the error to the caller
        // instead of logging the user out.
        const errBody = await res.json().catch(() => ({ message: res.statusText }))
        throw new ApiErrorResponse(res.status, errBody.message, errBody.errors)
      }
    } else {
      clear()
      onUnauthorized?.()
      throw new ApiErrorResponse(401, 'Unauthorized')
    }
  }

  if (res.status === 401) {
    clear()
    onUnauthorized?.()
  }

  if (!res.ok) {
    const errBody = await res.json().catch(() => ({ message: res.statusText }))
    throw new ApiErrorResponse(res.status, errBody.message, errBody.errors)
  }

  if (res.status === 204) return undefined as T
  // Some endpoints (e.g. POST /oauth/bitbucket/disconnect) return 200 with a
  // plain-text status body instead of JSON; treat them as empty responses.
  if (!res.headers.get('content-type')?.includes('json')) return undefined as T
  return res.json()
}
