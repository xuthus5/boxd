import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("SettingsPage", () => {
  it("shows appearance, account, and runtime settings", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/password") ? { defaultPassword: false }
        : path.endsWith("/jwt-secret") ? { masked: "ab********cd", present: true, length: 32 }
          : path.endsWith("/urltest-defaults") ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
          : path.endsWith("/url-test") ? { url: "https://cp.cloudflare.com/" }
            : { enabled: true }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/settings")
    expect(await screen.findByRole("heading", { name: "应用设置" })).toBeInTheDocument()
    expect(screen.getByText("登录用户名由 BOXUI_USERNAME 或启动参数管理，前端不可轮换。")).toBeInTheDocument()
    expect(screen.getByRole("combobox", { name: "测速地址" })).toBeInTheDocument()
    expect(screen.queryByLabelText("自定义测速地址")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存测速地址" })).toBeInTheDocument()
    expect(screen.getByText("订阅 URLTest 默认值")).toBeInTheDocument()
  })
})
