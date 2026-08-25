import { apiRequest } from './client'
import type {
  BitbucketOAuthAuthorizeResponse,
  BitbucketOAuthStatus,
} from '$lib/types/api'

export function getBitbucketOAuthStatus(): Promise<BitbucketOAuthStatus> {
  return apiRequest<BitbucketOAuthStatus>('GET', '/oauth/bitbucket/status')
}

export function getBitbucketOAuthAuthorizeUrl(): Promise<BitbucketOAuthAuthorizeResponse> {
  return apiRequest<BitbucketOAuthAuthorizeResponse>(
    'GET',
    '/oauth/bitbucket/authorize',
  )
}

export function disconnectBitbucketOAuth(): Promise<void> {
  return apiRequest<void>('POST', '/oauth/bitbucket/disconnect')
}
