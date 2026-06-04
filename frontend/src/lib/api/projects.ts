import { apiRequest } from './client'
import type { Project, PaginatedResponse } from '$lib/types/api'

export function listProjects(limit = 20, offset = 0): Promise<PaginatedResponse<Project>> {
  return apiRequest<PaginatedResponse<Project>>('GET', `/projects?limit=${limit}&offset=${offset}`)
}

export function getProject(slug: string): Promise<Project> {
  return apiRequest<Project>('GET', `/projects/${slug}`)
}

export function createProject(data: { name: string; repo_url: string }): Promise<Project> {
  return apiRequest<Project>('POST', '/projects', data)
}

export function updateProject(slug: string, data: { name?: string; repo_url?: string }): Promise<Project> {
  return apiRequest<Project>('PATCH', `/projects/${slug}`, data)
}

export function deleteProject(slug: string): Promise<void> {
  return apiRequest<void>('DELETE', `/projects/${slug}`)
}
