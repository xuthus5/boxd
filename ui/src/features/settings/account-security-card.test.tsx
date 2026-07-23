import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { AuthProvider } from "@/features/auth/auth-context"
import { AccountSecurityCard } from "@/features/settings/account-security-card"
import { PreferencesProvider } from "@/features/preferences/preferences-provider"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function renderCard(props?: Partial<{ defaultPassword: boolean; jwt: { masked: string; present: boolean; length: number } }>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return renderApp(
    <QueryClientProvider client={client}>
      <AuthProvider>
        <PreferencesProvider>
          <AccountSecurityCard
            defaultPassword={props?.defaultPassword ?? false}
            jwt={props?.jwt ?? { masked: "abcd****", present: true, length: 32 }}
          />
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
  vi.mocked(toast.error).mockClear()
  vi.mocked(toast.success).mockClear()
})

describe("AccountSecurityCard", () => {
  it("keeps password fields after a failed rotation and supports clear", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = typeof input === "string" ? input : input.toString()
      if (init?.method === "PUT" && path.includes("/api/settings/password")) {
        return Promise.resolve(new Response(JSON.stringify({
          code: "invalid_request",
          message: "bad password",
        }), { status: 400 }))
      }
      return Promise.resolve(new Response(JSON.stringify({ ok: true })))
    }))
    const user = userEvent.setup()
    renderCard()
    expect(screen.getByText("已自定义密码")).toBeInTheDocument()
    expect(screen.getByText("JWT · 32")).toBeInTheDocument()
    await user.type(screen.getByLabelText("当前密码"), "current")
    await user.type(screen.getByLabelText("新密码"), "replacement-password-456")
    await user.type(screen.getByLabelText("确认新密码"), "replacement-password-456")
    await user.click(screen.getByRole("button", { name: "轮换密码" }))
    await user.click(screen.getByRole("button", { name: "确认轮换" }))
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("bad password")
    })
    expect(screen.getByLabelText("当前密码")).toHaveValue("current")
    await user.click(screen.getByRole("button", { name: "取消" }))
    await user.click(screen.getByRole("button", { name: "清空" }))
    expect(screen.getByLabelText("当前密码")).toHaveValue("")
    expect(screen.getByLabelText("新密码")).toHaveValue("")
  })

  it("shows default-password and missing JWT badges", () => {
    renderCard({
      defaultPassword: true,
      jwt: { masked: "", present: false, length: 0 },
    })
    expect(screen.getByText("默认密码")).toBeInTheDocument()
    expect(screen.getByText("JWT 未配置")).toBeInTheDocument()
    expect(screen.getByText("默认密码仍在使用")).toBeInTheDocument()
  })

  it("disables rotate until password form is valid", async () => {
    const user = userEvent.setup()
    renderCard()
    const rotate = screen.getByRole("button", { name: "轮换密码" })
    expect(rotate).toBeDisabled()
    await user.type(screen.getByLabelText("当前密码"), "current")
    await user.type(screen.getByLabelText("新密码"), "short")
    await user.type(screen.getByLabelText("确认新密码"), "short")
    expect(rotate).toBeDisabled()
    await user.clear(screen.getByLabelText("新密码"))
    await user.type(screen.getByLabelText("新密码"), "replacement-password-456")
    await user.clear(screen.getByLabelText("确认新密码"))
    await user.type(screen.getByLabelText("确认新密码"), "replacement-password-456")
    await waitFor(() => expect(rotate).toBeEnabled())
  })
})
