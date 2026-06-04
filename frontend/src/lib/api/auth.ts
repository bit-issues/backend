import { apiRequest } from './client'
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RefreshRequest,
  RefreshResponse,
  LogoutRequest,
  ChangePasswordRequest,
  User,
} from '$lib/types/api'

export function loginApi(data: LoginRequest): Promise<LoginResponse> {
  return apiRequest<LoginResponse>('POST', '/auth/login', data)
}

export function registerApi(data: RegisterRequest): Promise<User> {
  return apiRequest<User>('POST', '/auth/register', data)
}

export function refreshApi(data: RefreshRequest): Promise<RefreshResponse> {
  return apiRequest<RefreshResponse>('POST', '/auth/refresh', data)
}

export function logoutApi(data: LogoutRequest): Promise<void> {
  return apiRequest<void>('POST', '/auth/logout', data)
}

export function changePasswordApi(data: ChangePasswordRequest): Promise<void> {
  return apiRequest<void>('POST', '/auth/change-password', data)
}
