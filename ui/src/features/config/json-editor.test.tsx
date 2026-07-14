import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { JsonEditor } from "@/features/config/json-editor"
import { renderApp } from "@/test/render"

describe("JsonEditor", () => {
  it("reports invalid JSON", () => {
    renderApp(<JsonEditor value="{" onChange={vi.fn()} ariaLabel="配置 JSON" />)
    expect(screen.getByText("JSON 格式无效")).toBeInTheDocument()
  })

  it("renders valid JSON without an error", () => {
    renderApp(<JsonEditor value="{}" onChange={vi.fn()} ariaLabel="配置 JSON" />)
    expect(screen.queryByText("JSON 格式无效")).not.toBeInTheDocument()
    expect(screen.getByRole("textbox", { name: "配置 JSON" })).toBeInTheDocument()
  })
})
