import { render, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { AuthProvider } from "@/features/auth/auth-context"
import { sessionStore } from "@/lib/session"

vi.mock("@/lib/api/desktop", () => ({
  isDesktop: vi.fn(),
  autoLogin: vi.fn(),
}))

import { autoLogin, isDesktop } from "@/lib/api/desktop"

afterEach(() => {
  sessionStore.clear()
  vi.clearAllMocks()
})

describe("AuthProvider embedded auto-login", () => {
  it("skips auto-login in web mode", () => {
    vi.mocked(isDesktop).mockReturnValue(false)
    render(<AuthProvider>{null}</AuthProvider>)
    expect(autoLogin).not.toHaveBeenCalled()
  })

  it("auto-logins and injects session in desktop mode", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(autoLogin).mockResolvedValue({
      token: "embedded-token",
      expires_at: "2099-01-01T00:00:00Z",
    })
    render(<AuthProvider>{null}</AuthProvider>)
    await waitFor(() => {
      expect(sessionStore.get()?.token).toBe("embedded-token")
    })
  })

  it("keeps existing session and does not re-login", () => {
    sessionStore.set({ token: "existing", expiresAt: "2099-01-01T00:00:00Z" })
    vi.mocked(isDesktop).mockReturnValue(true)
    render(<AuthProvider>{null}</AuthProvider>)
    expect(autoLogin).not.toHaveBeenCalled()
    expect(sessionStore.get()?.token).toBe("existing")
  })

  it("stays logged out when auto-login fails", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(autoLogin).mockRejectedValue(new Error("boom"))
    render(<AuthProvider>{null}</AuthProvider>)
    await waitFor(() => {
      expect(sessionStore.get()).toBeNull()
    })
  })
})
