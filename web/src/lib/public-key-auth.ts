import api from "@/lib/api"

type Ceremony<T> = { ceremonyId: string; options: T }

function decode(value: string): ArrayBuffer {
  const padded = value.replace(/-/g, "+").replace(/_/g, "/") + "===".slice((value.length + 3) % 4)
  const bytes = Uint8Array.from(atob(padded), (char) => char.charCodeAt(0))
  return bytes.buffer
}

function encode(value: ArrayBuffer): string {
  const bytes = new Uint8Array(value)
  let binary = ""
  bytes.forEach((byte) => { binary += String.fromCharCode(byte) })
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")
}

function creationOptions(options: PublicKeyCredentialCreationOptionsJSON): PublicKeyCredentialCreationOptions {
  return {
    ...options,
    challenge: decode(options.challenge),
    user: { ...options.user, id: decode(options.user.id) },
    excludeCredentials: options.excludeCredentials?.map((item) => ({ ...item, type: "public-key" as const, id: decode(item.id), transports: item.transports as AuthenticatorTransport[] | undefined })),
  } as unknown as PublicKeyCredentialCreationOptions
}

function requestOptions(options: PublicKeyCredentialRequestOptionsJSON): PublicKeyCredentialRequestOptions {
  return {
    ...options,
    challenge: decode(options.challenge),
    allowCredentials: options.allowCredentials?.map((item) => ({ ...item, type: "public-key" as const, id: decode(item.id), transports: item.transports as AuthenticatorTransport[] | undefined })),
  } as unknown as PublicKeyCredentialRequestOptions
}

function serialize(credential: PublicKeyCredential): object {
  const response = credential.response
  const base = { id: credential.id, rawId: encode(credential.rawId), type: credential.type, authenticatorAttachment: credential.authenticatorAttachment, clientExtensionResults: credential.getClientExtensionResults() }
  if (response instanceof AuthenticatorAttestationResponse) {
    return { ...base, response: { clientDataJSON: encode(response.clientDataJSON), attestationObject: encode(response.attestationObject), transports: response.getTransports() } }
  }
  const assertion = response as AuthenticatorAssertionResponse
  return { ...base, response: { clientDataJSON: encode(assertion.clientDataJSON), authenticatorData: encode(assertion.authenticatorData), signature: encode(assertion.signature), userHandle: assertion.userHandle ? encode(assertion.userHandle) : null } }
}

export function publicKeyAvailable(): boolean {
  return typeof window !== "undefined" && window.isSecureContext && "PublicKeyCredential" in window
}

export interface PublicKeyConfig { enabled: boolean }
export async function getPublicKeyConfig() {
  return api.get("/auth/public-key/config") as Promise<PublicKeyConfig>
}

export async function publicKeyEnabled(): Promise<boolean> {
  if (!publicKeyAvailable()) return false
  try {
    return (await getPublicKeyConfig()).enabled
  } catch {
    return false
  }
}

export async function registerPublicKey(label: string, password: string, totpCode: string) {
  const ceremony = await api.post("/auth/public-key/register/begin", { label, password, totpCode }) as Ceremony<PublicKeyCredentialCreationOptionsJSON>
  const credential = await navigator.credentials.create({ publicKey: creationOptions(ceremony.options) }) as PublicKeyCredential | null
  if (!credential) throw new Error("Credential creation was cancelled")
  return api.post("/auth/public-key/register/finish", serialize(credential), { headers: { "X-WebAuthn-Ceremony": ceremony.ceremonyId } })
}

export async function loginWithPublicKey(username = "") {
  const ceremony = await api.post("/auth/public-key/login/begin", { username }) as Ceremony<PublicKeyCredentialRequestOptionsJSON>
  const credential = await navigator.credentials.get({ publicKey: requestOptions(ceremony.options) }) as PublicKeyCredential | null
  if (!credential) throw new Error("Authentication was cancelled")
  return api.post("/auth/public-key/login/finish", serialize(credential), { headers: { "X-WebAuthn-Ceremony": ceremony.ceremonyId } })
}

export interface PublicKeyCredentialInfo { id: number; label: string; transports: string[]; createdAt: string; lastUsedAt?: string; backupEligible: boolean; backupState: boolean }
export async function listPublicKeys() { return api.get("/auth/public-key/credentials") as Promise<PublicKeyCredentialInfo[]> }
export async function renamePublicKey(id: number, label: string) { return api.put(`/auth/public-key/credentials/${id}`, { label }) }
export async function revokePublicKey(id: number) { return api.delete(`/auth/public-key/credentials/${id}`) }
