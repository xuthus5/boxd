import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function setup(route: string) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  installMockAPI()
  return { user: userEvent.setup(), view: renderApp(<App />, route) }
}

describe("advanced configuration interactions", () => {
  it("saves endpoints and experimental sections", async () => {
    const endpoints = setup("/advanced/endpoints")
    await endpoints.user.click(await screen.findByRole("button", { name: "保存配置" }))
    endpoints.view.unmount()
    const experimental = setup("/advanced/experimental")
    await experimental.user.click(await screen.findByRole("button", { name: "保存配置" }))
  })

  it("resets and saves the full configuration", async () => {
    const { user } = setup("/advanced/raw")
    await user.click(await screen.findByRole("button", { name: "重置修改" }))
    await user.click(screen.getByRole("button", { name: "保存完整配置" }))
    await user.click(screen.getByRole("button", { name: "确认覆盖" }))
  })
})
