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

  it("falls back to a temporary textarea when the clipboard is unavailable", async () => {
    vi.stubGlobal("navigator", {})
    const fixture = createClipboardDocument(true)

    await copyText("tag-c", undefined, fixture.document)

    expect(fixture.textarea.value).toBe("tag-c")
    expect(fixture.appendChild).toHaveBeenCalledWith(fixture.textarea)
    expect(fixture.select).toHaveBeenCalled()
    expect(fixture.setSelectionRange).toHaveBeenCalledWith(0, 5)
    expect(fixture.execCommand).toHaveBeenCalledWith("copy")
    expect(fixture.remove).toHaveBeenCalled()
  })

  it("uses the textarea fallback when clipboard permission is denied", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("permission denied"))
    const fixture = createClipboardDocument(true)

    await copyText("tag-d", { writeText }, fixture.document)

    expect(writeText).toHaveBeenCalledWith("tag-d")
    expect(fixture.execCommand).toHaveBeenCalledWith("copy")
  })

  it("cleans up and rejects when no copy strategy succeeds", async () => {
    vi.stubGlobal("navigator", {})
    const fixture = createClipboardDocument(false)

    await expect(copyText("tag-e", undefined, fixture.document)).rejects.toThrow(/clipboard/i)
    expect(fixture.remove).toHaveBeenCalled()
  })
})

function createClipboardDocument(copied: boolean) {
  const appendChild = vi.fn()
  const execCommand = vi.fn().mockReturnValue(copied)
  const remove = vi.fn()
  const select = vi.fn()
  const setSelectionRange = vi.fn()
  const textarea = {
    value: "",
    style: {},
    tabIndex: 0,
    setAttribute: vi.fn(),
    remove,
    select,
    setSelectionRange,
  }
  const document = {
    body: { appendChild },
    createElement: vi.fn().mockReturnValue(textarea),
    execCommand,
  } as unknown as Document
  return { appendChild, document, execCommand, remove, select, setSelectionRange, textarea }
}
