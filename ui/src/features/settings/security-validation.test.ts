import { describe, expect, it } from "vitest"

import {
  isJWTSecretReady,
  isPasswordFormReady,
  MIN_ADMIN_PASSWORD_LENGTH,
  MIN_JWT_SECRET_LENGTH,
  validateAdminPassword,
  validateJWTSecret,
  validatePasswordConfirmation,
} from "@/features/settings/security-validation"

describe("security-validation", () => {
  it("enforces admin password policy aligned with backend", () => {
    expect(MIN_ADMIN_PASSWORD_LENGTH).toBe(12)
    expect(validateAdminPassword("short")).toBe("too_short")
    expect(validateAdminPassword("admin", "admin")).toBe("too_short")
    expect(validateAdminPassword("AdminAccount", "adminaccount")).toBe("matches_username")
    // Backend weak list entries are shorter than the minimum length, so length wins first.
    expect(validateAdminPassword("password")).toBe("too_short")
    expect(validateAdminPassword("admin123")).toBe("too_short")
    expect(validateAdminPassword("replacement-password-456")).toBeNull()
  })

  it("requires matching password confirmation", () => {
    expect(validatePasswordConfirmation("a", "b")).toBe("mismatch")
    expect(validatePasswordConfirmation("same-password-1", "same-password-1")).toBeNull()
    expect(isPasswordFormReady("current", "replacement-password-456", "replacement-password-456")).toBe(true)
    expect(isPasswordFormReady("", "replacement-password-456", "replacement-password-456")).toBe(false)
    expect(isPasswordFormReady("current", "short", "short")).toBe(false)
  })

  it("enforces JWT secret minimum length", () => {
    expect(MIN_JWT_SECRET_LENGTH).toBe(16)
    expect(validateJWTSecret("")).toBe("empty")
    expect(validateJWTSecret("short")).toBe("too_short")
    expect(validateJWTSecret("replacement-secret")).toBeNull()
    expect(isJWTSecretReady("replacement-secret")).toBe(true)
  })
})
