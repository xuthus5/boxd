/** Keep client checks aligned with internal/core credential + JWT rules. */
export const MIN_ADMIN_PASSWORD_LENGTH = 12
export const MIN_JWT_SECRET_LENGTH = 16

const WEAK_PASSWORDS = new Set(["admin123", "password", "12345678", "qwerty123"])

export type PasswordIssue =
  | "too_short"
  | "matches_username"
  | "weak_common"
  | "mismatch"
  | null

export type JWTSecretIssue = "empty" | "too_short" | null

export function normalizeSecret(value: string) {
  return value.trim()
}

export function validateAdminPassword(
  password: string,
  username = "admin",
): PasswordIssue {
  const value = password
  if (value.length < MIN_ADMIN_PASSWORD_LENGTH) return "too_short"
  if (value.toLowerCase() === username.toLowerCase()) {
    return "matches_username"
  }
  if (WEAK_PASSWORDS.has(value.toLowerCase())) return "weak_common"
  return null
}

export function validatePasswordConfirmation(
  password: string,
  confirmation: string,
): PasswordIssue {
  if (password !== confirmation) return "mismatch"
  return null
}

export function isPasswordFormReady(
  currentPassword: string,
  newPassword: string,
  confirmation: string,
  username = "admin",
) {
  if (!currentPassword) return false
  if (validateAdminPassword(newPassword, username)) return false
  if (validatePasswordConfirmation(newPassword, confirmation)) return false
  return true
}

export function validateJWTSecret(secret: string): JWTSecretIssue {
  const value = secret
  if (!value) return "empty"
  if (value.length < MIN_JWT_SECRET_LENGTH) return "too_short"
  return null
}

export function isJWTSecretReady(secret: string) {
  return validateJWTSecret(secret) === null
}
