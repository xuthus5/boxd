import { useState } from "react"
import { fireEvent, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { JsonValueField } from "@/features/config/json-value-field"
import type { JsonValue } from "@/lib/api/types"
import { renderApp } from "@/test/render"

function Harness({ onValidity }: { onValidity?: (path: string, valid: boolean) => void }) {
  const [value, setValue] = useState<JsonValue | undefined>()
  return <><JsonValueField path="provider" label="Provider" value={value} onChange={setValue} onValidity={onValidity} />
    <output aria-label="result">{JSON.stringify(value) ?? "unset"}</output></>
}

describe("JSON union field", () => {
  it("preserves booleans, references, and nested objects without coercion", () => {
    renderApp(<Harness />)
    for (const raw of ['true', '"shared"', '{"type":"acme","domain":["example.org"]}', 'false']) {
      fireEvent.change(screen.getByLabelText("Provider"), { target: { value: raw } })
      expect(screen.getByLabelText("result")).toHaveTextContent(raw)
    }
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "" } })
    expect(screen.getByLabelText("result")).toHaveTextContent("unset")
  })

  it("reports malformed drafts and keeps the last valid value", () => {
    const onValidity = vi.fn()
    const view = renderApp(<Harness onValidity={onValidity} />)
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: '{"type":' } })
    expect(screen.getByLabelText("Provider")).toHaveAttribute("aria-invalid", "true")
    expect(onValidity).toHaveBeenLastCalledWith("provider", false)
    expect(screen.getByLabelText("result")).toHaveTextContent("unset")
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: '"shared"' } })
    expect(onValidity).toHaveBeenLastCalledWith("provider", true)
    view.unmount()
    expect(onValidity).toHaveBeenLastCalledWith("provider", true)
  })

  it("resets an invalid draft when the parent JSON revision changes", () => {
    const onChange = vi.fn()
    const view = renderApp(<JsonValueField path="response" label="Response" value={true} onChange={onChange} />)
    fireEvent.change(screen.getByLabelText("Response"), { target: { value: "{" } })
    view.rerender(<JsonValueField path="response" label="Response" value="remote" revision={1} onChange={onChange} />)
    expect(screen.getByLabelText("Response")).toHaveValue('"remote"')
    expect(screen.getByLabelText("Response")).toHaveAttribute("aria-invalid", "false")
  })
})
