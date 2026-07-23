import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes, useSearchParams } from "react-router-dom"
import { describe, expect, it, vi } from "vitest"

import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"

function Probe({
  reveal,
  ready = true,
  section,
}: {
  reveal: (path: string) => boolean
  ready?: boolean
  section?: string
}) {
  useConfigPathReveal(reveal, { ready, section })
  const [params] = useSearchParams()
  return <div data-testid="path">{params.get("path") ?? ""}</div>
}

function renderWithPath(path: string, reveal: (path: string) => boolean, section?: string) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/advanced/raw" element={<Probe reveal={reveal} section={section} />} />
          <Route path="/proxy/inbounds" element={<Probe reveal={reveal} section={section ?? "inbounds"} />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe("useConfigPathReveal", () => {
  it("reveals and clears path query after success", async () => {
    const reveal = vi.fn().mockReturnValue(true)
    const view = renderWithPath("/advanced/raw?path=inbounds%5B0%5D.listen_port", reveal)
    await waitFor(() => expect(reveal).toHaveBeenCalledWith("inbounds[0].listen_port"))
    await waitFor(() => expect(view.getByTestId("path")).toHaveTextContent(""))
  })

  it("tries section-relative candidates", async () => {
    const reveal = vi.fn((path: string) => path === "listen_port")
    renderWithPath("/proxy/inbounds?path=inbounds%5B0%5D.listen_port", reveal, "inbounds")
    await waitFor(() => expect(reveal).toHaveBeenCalled())
    expect(reveal.mock.calls.map((call) => call[0])).toEqual(expect.arrayContaining([
      "inbounds[0].listen_port",
      "[0].listen_port",
    ]))
  })
})
