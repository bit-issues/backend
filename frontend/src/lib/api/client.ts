import type { ApiError, RefreshResponse } from '$lib/types/api'

let accessToken: string | null = null
let refreshToken: string | null = null
let onUnauthorized: (() => void) | null = null

export function setTokens(access: string | null, refresh: string | null) {
  accessToken = access
  refreshToken = refresh
}

export function getAccessToken() {
  return accessToken
}

export function setOnUnauthorized(cb: () => void) {
  onUnauthorized = cb
}

async function refreshTokens(): Promise<boolean> {
  if (!refreshToken) return false
  try {
    const res = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return false
    const data: RefreshResponse = await res.json()
    accessToken = data.access_token
    refreshToken = data.refresh_token
    return true
  } catch {
    return false
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
  if (accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`
  }

  const requestBody = body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined

  let res = await fetch(`/api/v1${path}`, {
    method,
    headers,
    body: requestBody,
  })

  if (res.status === 401 && refreshToken) {
    const refreshed = await refreshTokens()
    if (refreshed) {
      headers['Authorization'] = `Bearer ${accessToken}`
      res = await fetch(`/api/v1${path}`, {
        method,
        headers,
        body: requestBody,
      })
    } else {
      onUnauthorized?.()
      throw new ApiErrorResponse(401, 'Unauthorized')
    }
  }

  if (!res.ok) {
    const errBody = await res.json().catch(() => ({ message: res.statusText }))
    throw new ApiErrorResponse(res.status, errBody.message, errBody.errors)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}
