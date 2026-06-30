import { apiRequest } from './client'
import type {
  Task,
  TaskDetails,
  TaskListResponse,
  TaskCreateRequest,
  TaskUpdateRequest,
  PaginatedResponse,
} from '$lib/types/api'

export interface TaskFilters {
  project?: string
  author?: number
  assignee?: number
  statuses?: string
  priorities?: string
  search?: string
  sort?: string
  limit?: number
  offset?: number
}

function buildQuery(filters: TaskFilters): string {
  const params = new URLSearchParams()
  if (filters.project) params.set('project', filters.project)
  if (filters.author !== undefined) params.set('author', String(filters.author))
  if (filters.assignee !== undefined) params.set('assignee', String(filters.assignee))
  if (filters.statuses) params.set('statuses', filters.statuses)
  if (filters.priorities) params.set('priorities', filters.priorities)
  if (filters.search) params.set('search', filters.search)
  if (filters.sort) params.set('sort', filters.sort)
  if (filters.limit !== undefined) params.set('limit', String(filters.limit))
  if (filters.offset !== undefined) params.set('offset', String(filters.offset))
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

export function listTasks(filters: TaskFilters = {}): Promise<TaskListResponse> {
  return apiRequest<TaskListResponse>('GET', `/tasks${buildQuery(filters)}`)
}

export function getMyTasks(filters: TaskFilters = {}): Promise<TaskListResponse> {
  return apiRequest<TaskListResponse>('GET', `/tasks/me${buildQuery(filters)}`)
}

export function getTask(id: number): Promise<TaskDetails> {
  return apiRequest<TaskDetails>('GET', `/tasks/${id}`)
}

export function createTask(data: TaskCreateRequest): Promise<TaskDetails> {
  return apiRequest<TaskDetails>('POST', '/tasks', data)
}

export function updateTask(id: number, data: TaskUpdateRequest): Promise<TaskDetails> {
  return apiRequest<TaskDetails>('PATCH', `/tasks/${id}`, data)
}

export function deleteTask(id: number): Promise<void> {
  return apiRequest<void>('DELETE', `/tasks/${id}`)
}
