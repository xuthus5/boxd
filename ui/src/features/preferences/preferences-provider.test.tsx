import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); localStorage.clear(); document.documentElement.classList.remove("dark") })

describe("PreferencesProvider", () => {
  it("uses the dark system preference", () => {
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({
      matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn(),
    }))
    renderApp(<App />, "/login")
    expect(screen.getByRole("heading", { name: "BoxUI" })).toBeInTheDocument()
    expect(document.documentElement).toHaveClass("dark")
  })
})
