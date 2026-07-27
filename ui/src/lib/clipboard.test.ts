import { afterEach, describe, expect, it, vi } from "vitest"

import { copyText } from "@/lib/clipboard"

describe("copyText", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("uses the provided clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)

    await copyText("tag-a", { writeText })

    expect(writeText).toHaveBeenCalledWith("tag-a")
  })

  it("falls back to the browser clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })

    await copyText("tag-b")

    expect(writeText).toHaveBeenCalledWith("tag-b")
  })

  it("rejects when the clipboard is unavailable", async () => {
    vi.stubGlobal("navigator", {})

    await expect(copyText("tag-c")).rejects.toThrow(/clipboard/i)
  })
})
