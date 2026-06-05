import { apiRequest } from './client'
import type { UserBrief } from '$lib/types/api'

export interface UserSearchResponse {
  items: UserBrief[]
  total: number
}

export function searchUsers(query: string, limit = 20): Promise<UserSearchResponse> {
  return apiRequest<UserSearchResponse>('GET', `/users/search?query=${encodeURIComponent(query)}&limit=${limit}`)
}
