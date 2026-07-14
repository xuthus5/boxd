import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import App from "@/App"
import { PagePlaceholder } from "@/app/page-placeholder"
import { renderApp } from "@/test/render"

describe("App", () => {
  it("renders the BoxUI login entry", () => {
    renderApp(<App />)
    expect(screen.getByRole("heading", { name: /boxui/i })).toBeInTheDocument()
  })

  it("renders translated placeholders", () => {
    renderApp(<PagePlaceholder titleKey="pages.dashboard" />)
    expect(screen.getByRole("heading", { name: "仪表盘" })).toBeInTheDocument()
  })
})
