import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("dashboard interactions", () => {
  it("runs service and maintenance actions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installMockAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/dashboard")
    await screen.findByText("运行中")

    for (const name of ["启动", "停止", "重启", "GC", "清理 DNS", "清理 FakeIP"]) {
      await user.click(screen.getByRole("button", { name }))
      if (name === "停止" || name === "重启") {
        await user.click(screen.getByRole("button", { name: "确认操作" }))
        await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument())
      }
    }

    await vi.waitFor(() => expect(fetchMock.mock.calls.some(([path]) => path === "/api/runtime/fakeip/flush")).toBe(true))
  })
})
