import { apiRequest } from './client'
import type { AttachmentInitRequest, AttachmentInitResponse, AttachmentConfirmResponse } from '$lib/types/api'

export function initUpload(taskId: number, data: AttachmentInitRequest): Promise<AttachmentInitResponse> {
  return apiRequest<AttachmentInitResponse>('POST', `/tasks/${taskId}/attachments`, data)
}

export function confirmUpload(taskId: number, attachmentId: number): Promise<AttachmentConfirmResponse> {
  return apiRequest<AttachmentConfirmResponse>('PUT', `/tasks/${taskId}/attachments/${attachmentId}/confirm`)
}

export function getDownloadUrl(taskId: number, attachmentId: number): Promise<{ download_url: string }> {
  return apiRequest<{ download_url: string }>('GET', `/tasks/${taskId}/attachments/${attachmentId}/download`)
}

export function deleteAttachment(taskId: number, attachmentId: number): Promise<void> {
  return apiRequest<void>('DELETE', `/tasks/${taskId}/attachments/${attachmentId}`)
}
