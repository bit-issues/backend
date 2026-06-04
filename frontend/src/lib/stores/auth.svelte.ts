import { loginApi, registerApi, logoutApi } from '$lib/api/auth'
import { setTokens } from '$lib/api/client'
import type { User } from '$lib/types/api'

let _accessToken = $state<string | null>(null)
let _refreshToken = $state<string | null>(null)
let _user = $state<User | null>(null)

export function getAccessToken(): string | null {
  return _accessToken
}

export function getRefreshToken(): string | null {
  return _refreshToken
}

export function getUser(): User | null {
  return _user
}

export function isAuthenticated(): boolean {
  return !!_accessToken
}

export function isAdmin(): boolean {
  return _user?.role === 'admin'
}

function persist() {
  if (_accessToken) localStorage.setItem('access_token', _accessToken)
  if (_refreshToken) localStorage.setItem('refresh_token', _refreshToken)
  if (_user) localStorage.setItem('user', JSON.stringify(_user))
}

function restore() {
  _accessToken = localStorage.getItem('access_token')
  _refreshToken = localStorage.getItem('refresh_token')
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      _user = JSON.parse(userStr)
    } catch {
      clear()
    }
  }
  setTokens(_accessToken, _refreshToken)
}

export function clear() {
  _accessToken = null
  _refreshToken = null
  _user = null
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('user')
  setTokens(null, null)
}

export async function login(email: string, password: string): Promise<void> {
  const res = await loginApi({ email, password })
  _accessToken = res.access_token
  _refreshToken = res.refresh_token
  _user = res.user
  persist()
  setTokens(res.access_token, res.refresh_token)
}

export async function register(email: string, password: string): Promise<User> {
  const user = await registerApi({ email, password })
  return user
}

export async function logout(): Promise<void> {
  try {
    if (_refreshToken) {
      await logoutApi({ refresh_token: _refreshToken })
    }
  } catch { /* ignore */ }
  clear()
}

restore()
