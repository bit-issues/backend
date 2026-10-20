import { apiRequest } from './client'
import type { WebhookStatus } from '$lib/types/api'

export function getProjectWebhook(slug: string): Promise<WebhookStatus> {
  return apiRequest<WebhookStatus>(
    'GET',
    `/projects/${encodeURIComponent(slug)}/webhook`,
  )
}

export function registerProjectWebhook(slug: string): Promise<WebhookStatus> {
  return apiRequest<WebhookStatus>(
    'POST',
    `/projects/${encodeURIComponent(slug)}/webhook/register`,
  )
}

export function unregisterProjectWebhook(slug: string): Promise<void> {
  return apiRequest<void>(
    'DELETE',
    `/projects/${encodeURIComponent(slug)}/webhook`,
  )
}
