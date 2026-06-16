import { apiRequest } from './client'
import type { User, UserBrief, PaginatedResponse } from '$lib/types/api'

export interface UserSearchResponse {
  items: UserBrief[]
  total: number
}

export function searchUsers(query: string, limit = 20): Promise<UserSearchResponse> {
  return apiRequest<UserSearchResponse>('GET', `/users/search?query=${encodeURIComponent(query)}&limit=${limit}`)
}

export function listUsers(filters: {
  status?: string
  role?: string
  limit?: number
  offset?: number
} = {}): Promise<PaginatedResponse<User>> {
  const params = new URLSearchParams()
  if (filters.status) params.set('status', filters.status)
  if (filters.role) params.set('role', filters.role)
  if (filters.limit) params.set('limit', String(filters.limit))
  if (filters.offset) params.set('offset', String(filters.offset))
  const qs = params.toString()
  return apiRequest<PaginatedResponse<User>>('GET', `/users${qs ? '?' + qs : ''}`)
}

export function updateUser(id: number, data: {
  status?: string
  role?: string
}): Promise<User> {
  return apiRequest<User>('PATCH', `/users/${id}`, data)
}
