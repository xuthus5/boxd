import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ErrorBoundary } from "@/app/error-boundary"
import { renderApp } from "@/test/render"

function BrokenPage(): never { throw new Error("render failed") }

describe("ErrorBoundary", () => {
  afterEach(() => vi.restoreAllMocks())

  it("shows a recoverable error surface when a page throws", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    renderApp(<ErrorBoundary><BrokenPage /></ErrorBoundary>)
    expect(screen.getByText("页面出现异常")).toBeInTheDocument()
    expect(screen.getByText("render failed")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
  })
})
