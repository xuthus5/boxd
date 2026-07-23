import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  Toaster: () => null,
}))

import { toast } from "sonner"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllGlobals()
  sessionStore.clear()
  localStorage.clear()
  document.documentElement.classList.remove("dark")
})

describe("PreferencesProvider", () => {
  it("uses the dark system preference", () => {
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({
      matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn(),
    }))
    renderApp(<App />, "/login")
    expect(screen.getByRole("heading", { name: "boxd" })).toBeInTheDocument()
    expect(document.documentElement).toHaveClass("dark")
  })

  it("loads remote preferences after login", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installMockAPI()
    const base = fetchMock.getMockImplementation()!
    fetchMock.mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname).split("?")[0]
      if (path === "/api/settings/preferences" && !init?.method) {
        return Promise.resolve(new Response(JSON.stringify({
          theme: "dark", language: "en", minimumLogLevel: "warn",
        })))
      }
      return base(input, init)
    })
    renderApp(<App />, "/settings")
    await screen.findByRole("heading", { name: "Application Settings" })
    expect(localStorage.getItem("boxd.preferences.v1")).toContain("dark")
    expect(localStorage.getItem("boxd.preferences.v1")).toContain("warn")
  })
})

describe("PreferencesProvider language failures", () => {
  it("logs when language switching fails", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => undefined)
    const { i18n } = await import("@/i18n")
    const change = vi.spyOn(i18n, "changeLanguage").mockRejectedValueOnce(new Error("lang failed"))
    const { PreferencesProvider } = await import("@/features/preferences/preferences-provider")
    const { AuthProvider } = await import("@/features/auth/auth-context")
    const { render } = await import("@testing-library/react")
    render(<AuthProvider><PreferencesProvider><div>child</div></PreferencesProvider></AuthProvider>)
    await vi.waitFor(() => expect(errorSpy).toHaveBeenCalled())
    change.mockRestore()
    errorSpy.mockRestore()
  })
})

describe("PreferencesProvider densified save errors", () => {
  it("surfaces densified toast when remote preference save fails", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installMockAPI()
    const base = fetchMock.getMockImplementation()!
    fetchMock.mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname).split("?")[0]
      if (path === "/api/settings/preferences" && init?.method === "PUT") {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error",
          data: null,
          error: { code: "internal_error", message: "failed to save preferences" },
          meta: {},
        }), { status: 500 }))
      }
      return base(input, init)
    })
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await screen.findByRole("heading", { name: "应用设置" })
    await user.click(screen.getByText("深色"))
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toMatch(/failed to save preferences|internal/i)
    expect(options).toEqual(expect.objectContaining({
      description: expect.any(String),
      action: expect.objectContaining({ label: expect.any(String) }),
    }))
  })
})
