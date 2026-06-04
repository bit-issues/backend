export interface User {
  id: number
  email: string
  name: string
  role: 'admin' | 'user'
  status: 'pending' | 'active' | 'blocked'
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  user: User
}

export interface RegisterRequest {
  email: string
  password: string
}

export interface RefreshRequest {
  refresh_token: string
}

export interface RefreshResponse {
  access_token: string
  refresh_token: string
}

export interface LogoutRequest {
  refresh_token: string
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

export interface ApiError {
  code: number
  message: string
  errors?: { field: string; message: string }[]
}

export interface Project {
  id: string
  name: string
  repo_url: string
  created_at: string
  updated_at: string
}

export interface UserBrief {
  id: number
  name: string
  role: 'admin' | 'user'
  created_at: string
}

export interface Task {
  id: number
  project_slug: string
  number: number
  title: string
  description: string
  priority: Priority
  status: Status
  kind: Kind
  author: UserBrief
  assignee: UserBrief | null
  due_date: string | null
  created_at: string
  updated_at: string
}

export interface TaskDetails extends Task {
  project: Project
  comments: Comment[]
  attachments: Attachment[]
}

export interface TaskCreateRequest {
  project_slug: string
  title: string
  description?: string
  priority?: Priority
  kind?: Kind
  assignee_id?: number
  due_date?: string
}

export interface TaskUpdateRequest {
  title?: string
  description?: string
  priority?: Priority
  kind?: Kind
  status?: Status
  assignee_id?: number | null
  due_date?: string | null
  comment?: string
}

export interface TaskListResponse {
  items: Task[]
  total: number
}

export const PRIORITIES = ['Trivial', 'Minor', 'Major', 'Critical', 'Blocker'] as const
export type Priority = typeof PRIORITIES[number]

export const STATUSES = ['New', 'Open', 'In Progress', 'Resolved', 'Closed', 'Reopened', 'Invalid', 'Duplicate', 'Wontfix', 'On Hold'] as const
export type Status = typeof STATUSES[number]

export const KINDS = ['Bug', 'Enhancement', 'Task', 'Proposal'] as const
export type Kind = typeof KINDS[number]

export interface Comment {
  id: number
  author: UserBrief
  content: string
  created_at: string
  updated_at: string
}

export interface CommentCreateRequest {
  content: string
}

export interface CommentUpdateRequest {
  content: string
}

export interface Attachment {
  id: number
  task_id: number
  author: UserBrief
  file_name: string
  size_bytes: number
  status: 'pending' | 'uploaded'
  created_at: string
  updated_at: string
}

export interface AttachmentInitRequest {
  file_name: string
  size_bytes: number
}

export interface AttachmentInitResponse {
  id: number
  file_name: string
  size_bytes: number
  upload_url: string
}

export interface AttachmentConfirmResponse {
  id: number
  file_name: string
  size_bytes: number
  download_url: string
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
}
