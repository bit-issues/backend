import { apiRequest } from './client'
import type { Comment, CommentCreateRequest, CommentUpdateRequest } from '$lib/types/api'

export function createComment(taskId: number, data: CommentCreateRequest): Promise<Comment> {
  return apiRequest<Comment>('POST', `/tasks/${taskId}/comments`, data)
}

export function updateComment(taskId: number, commentId: number, data: CommentUpdateRequest): Promise<Comment> {
  return apiRequest<Comment>('PUT', `/tasks/${taskId}/comments/${commentId}`, data)
}

export function deleteComment(taskId: number, commentId: number): Promise<void> {
  return apiRequest<void>('DELETE', `/tasks/${taskId}/comments/${commentId}`)
}
