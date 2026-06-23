import { apiRequest } from './client'
import type { LoginResponse, PasskeyCredential, RenamePasskeyRequest } from '$lib/types/api'

export async function passkeyRegisterBegin(): Promise<PublicKeyCredentialCreationOptions> {
  const raw = await apiRequest<Record<string, unknown>>('POST', '/auth/passkey/register/begin')
  return prepareCreationOptions(raw)
}

export async function passkeyRegisterComplete(credential: PublicKeyCredential): Promise<{ id: number; name: string; created_at: string }> {
  const data = credentialToJSON(credential)
  return apiRequest('POST', '/auth/passkey/register/complete', data)
}

export async function passkeyLoginBegin(): Promise<PublicKeyCredentialRequestOptions> {
  const raw = await apiRequest<Record<string, unknown>>('POST', '/auth/passkey/login/begin')
  return prepareRequestOptions(raw)
}

export async function passkeyLoginComplete(credential: PublicKeyCredential): Promise<LoginResponse> {
  const data = credentialToJSON(credential)
  return apiRequest<LoginResponse>('POST', '/auth/passkey/login/complete', data)
}

function prepareCreationOptions(json: Record<string, unknown>): PublicKeyCredentialCreationOptions {
  const pk = (json.publicKey ?? json) as Record<string, unknown>
  const user = pk.user as Record<string, unknown>
  const exclude = pk.excludeCredentials as Array<Record<string, unknown>> | undefined

  return {
    ...pk,
    challenge: base64URLDecode(pk.challenge as string),
    user: { ...user, id: base64URLDecode(user.id as string) },
    excludeCredentials: exclude?.map((cred) => ({
      ...cred,
      id: base64URLDecode(cred.id as string),
    })) as PublicKeyCredentialDescriptor[] ?? [],
  } as unknown as PublicKeyCredentialCreationOptions
}

function prepareRequestOptions(json: Record<string, unknown>): PublicKeyCredentialRequestOptions {
  const pk = (json.publicKey ?? json) as Record<string, unknown>
  const allow = pk.allowCredentials as Array<Record<string, unknown>> | undefined

  return {
    ...pk,
    challenge: base64URLDecode(pk.challenge as string),
    allowCredentials: allow?.map((cred) => ({
      ...cred,
      id: base64URLDecode(cred.id as string),
    })) as PublicKeyCredentialDescriptor[] ?? [],
  } as unknown as PublicKeyCredentialRequestOptions
}

export async function listPasskeys(): Promise<PasskeyCredential[]> {
  return apiRequest<PasskeyCredential[]>('GET', '/auth/passkey/credentials')
}

export async function renamePasskey(id: number, data: RenamePasskeyRequest): Promise<void> {
  return apiRequest<void>('PATCH', `/auth/passkey/credentials/${id}`, data)
}

export async function deletePasskey(id: number): Promise<void> {
  return apiRequest<void>('DELETE', `/auth/passkey/credentials/${id}`)
}

function credentialToJSON(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response as AuthenticatorAttestationResponse | AuthenticatorAssertionResponse

  const base: Record<string, unknown> = {
    id: credential.id,
    rawId: base64URLEncode(new Uint8Array(credential.rawId)),
    type: credential.type,
    clientExtensionResults: credential.getClientExtensionResults(),
  }

  if ('attestationObject' in response) {
    const attResp = response as AuthenticatorAttestationResponse
    base.response = {
      clientDataJSON: base64URLEncode(new Uint8Array(attResp.clientDataJSON)),
      attestationObject: base64URLEncode(new Uint8Array(attResp.attestationObject)),
      transports: attResp.getTransports?.() ?? [],
    }
  } else {
    const assertResp = response as AuthenticatorAssertionResponse
    base.response = {
      clientDataJSON: base64URLEncode(new Uint8Array(assertResp.clientDataJSON)),
      authenticatorData: base64URLEncode(new Uint8Array(assertResp.authenticatorData)),
      signature: base64URLEncode(new Uint8Array(assertResp.signature)),
      userHandle: assertResp.userHandle ? base64URLEncode(new Uint8Array(assertResp.userHandle)) : null,
    }
  }

  return base
}

function base64URLDecode(str: string): Uint8Array {
  const base64 = str.replace(/-/g, '+').replace(/_/g, '/')
  const padded = base64.padEnd(base64.length + (4 - (base64.length % 4)) % 4, '=')
  const binaryStr = atob(padded)
  const bytes = new Uint8Array(binaryStr.length)
  for (let i = 0; i < binaryStr.length; i++) {
    bytes[i] = binaryStr.charCodeAt(i)
  }
  return bytes
}

function base64URLEncode(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}
