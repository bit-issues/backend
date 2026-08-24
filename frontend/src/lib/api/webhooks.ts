import { apiRequest } from './client'
import type { ProjectWebhookStatus } from '$lib/types/api'

export function getProjectWebhookStatus(slug: string): Promise<ProjectWebhookStatus> {
  return apiRequest<ProjectWebhookStatus>(
    'GET',
    `/projects/${encodeURIComponent(slug)}/webhook`,
  )
}

export function registerProjectWebhook(slug: string): Promise<ProjectWebhookStatus> {
  return apiRequest<ProjectWebhookStatus>(
    'POST',
    `/projects/${encodeURIComponent(slug)}/webhook/register`,
  )
}

export function unregisterProjectWebhook(slug: string): Promise<ProjectWebhookStatus> {
  return apiRequest<ProjectWebhookStatus>(
    'POST',
    `/projects/${encodeURIComponent(slug)}/webhook/unregister`,
  )
}
