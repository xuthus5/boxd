import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { SetupRunAction } from "@/features/setup/setup-run-action"
import { renderApp } from "@/test/render"

afterEach(() => vi.unstubAllGlobals())

function renderAction(running = false, failValidation = false) {
  const requests: string[] = []
  vi.stubGlobal("fetch", vi.fn(async (url: string) => {
    requests.push(url)
    if (url.includes("/validate") && failValidation) return new Response(JSON.stringify({
      status: "error", data: null, error: { code: "config_invalid_runtime", message: "bad route" }, meta: null,
    }), { status: 400 })
    return new Response(JSON.stringify(url === "/api/config/" ? {} : { status: "ok", data: null, error: null, meta: null }))
  }))
  renderApp(<QueryClientProvider client={new QueryClient()}><SetupRunAction running={running} ready /></QueryClientProvider>)
  return requests
}

describe("SetupRunAction", () => {
  it("validates the latest configuration before explicit startup", async () => {
    const requests = renderAction()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "校验并启动" }))
    await waitFor(() => expect(requests).toEqual(["/api/config/", "/api/config/validate?source=setup", "/api/service/start"]))
  })
  it("never starts after validation failure", async () => {
    const requests = renderAction(false, true)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "校验并启动" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("bad route")
    expect(requests).not.toContain("/api/service/start")
    await user.click(screen.getByRole("button", { name: "关闭提示" }))
    expect(screen.queryByTestId("config-save-error")).not.toBeInTheDocument()
  })
  it("only validates an already running kernel", async () => {
    const requests = renderAction(true)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "重新校验" }))
    await waitFor(() => expect(requests).toHaveLength(2))
    expect(requests).not.toContain("/api/service/start")
  })
})
