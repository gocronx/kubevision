import { useMutation } from "@tanstack/react-query"
import api from "@/lib/api"

// ---- Types ----

export interface Setup2FAResponse {
  secret: string
  otpauthUrl: string
  recoveryCodes: string[]
}

export interface LoginTokens {
  accessToken: string
  refreshToken: string
  user: {
    id: number
    username: string
    role: string
    totpEnabled: boolean
  }
}

// ---- Hooks ----

/**
 * Initiates TOTP setup for the currently authenticated user.
 * Returns the QR code URL, raw secret, and one-time recovery codes.
 * The secret is NOT active until useEnable2FA succeeds.
 */
export function useSetup2FA() {
  return useMutation<Setup2FAResponse, Error>({
    mutationFn: () => api.post("/auth/2fa/setup", {}),
  })
}

/**
 * Activates TOTP for the current user after they have scanned the QR code
 * and confirmed with a valid 6-digit code.
 */
export function useEnable2FA() {
  return useMutation<{ enabled: boolean }, Error, { code: string }>({
    mutationFn: (body) => api.post("/auth/2fa/enable", body),
  })
}

/**
 * Disables TOTP for the current user. Requires a valid 6-digit code to
 * prevent accidental or malicious removal.
 */
export function useDisable2FA() {
  return useMutation<{ disabled: boolean }, Error, { code: string }>({
    mutationFn: (body) => api.post("/auth/2fa/disable", body),
  })
}

/**
 * Completes the 2FA login step by exchanging a tempToken + TOTP code
 * for full JWT tokens.
 */
export function useVerify2FA() {
  return useMutation<LoginTokens, Error, { tempToken: string; code: string }>({
    mutationFn: (body) => api.post("/auth/2fa/verify", body),
  })
}

/**
 * Completes the 2FA login step using a one-time recovery code instead of
 * a TOTP code. The used recovery code is invalidated server-side.
 */
export function useRecoveryCode() {
  return useMutation<LoginTokens, Error, { tempToken: string; recoveryCode: string }>({
    mutationFn: (body) => api.post("/auth/2fa/recovery", body),
  })
}
