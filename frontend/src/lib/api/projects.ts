import { apiRequest } from './client'
import type { Project, PaginatedResponse } from '$lib/types/api'

export function listProjects(limit = 20, offset = 0, search?: string): Promise<PaginatedResponse<Project>> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  if (search) params.set('search', search)
  return apiRequest<PaginatedResponse<Project>>('GET', `/projects?${params}`)
}

export function getProject(slug: string): Promise<Project> {
  return apiRequest<Project>('GET', `/projects/${encodeURIComponent(slug)}`)
}

export function createProject(data: { name: string; repo_url: string }): Promise<Project> {
  return apiRequest<Project>('POST', '/projects', data)
}

export function updateProject(slug: string, data: { name?: string; repo_url?: string }): Promise<Project> {
  return apiRequest<Project>('PATCH', `/projects/${encodeURIComponent(slug)}`, data)
}

export function deleteProject(slug: string): Promise<void> {
  return apiRequest<void>('DELETE', `/projects/${encodeURIComponent(slug)}`)
}
