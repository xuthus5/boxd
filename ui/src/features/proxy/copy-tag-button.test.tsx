import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { CopyTagButton, copyText } from "@/features/proxy/copy-tag-button"
import { i18n } from "@/i18n"

describe("copyText", () => {
  it("uses clipboard writeText when available", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    await copyText("tag-a")
    expect(writeText).toHaveBeenCalledWith("tag-a")
    vi.unstubAllGlobals()
  })

  it("rejects when clipboard is unavailable", async () => {
    vi.stubGlobal("navigator", {})
    await expect(copyText("x")).rejects.toThrow(/clipboard/)
    vi.unstubAllGlobals()
  })
})

describe("CopyTagButton", () => {
  it("renders accessible copy control", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <CopyTagButton tag="mixed-in" />
      </I18nextProvider>,
    )
    expect(screen.getByRole("button", { name: "复制 tag" })).toBeInTheDocument()
  })
})
