import { describe, expect, it } from "vitest"

import {
  classifyExportError,
  exportErrorClipboardText,
  exportErrorHintKey,
  formatExportErrorToast,
} from "@/features/observability/export-error"

describe("export error diagnostics", () => {
  it("maps codes to hints", () => {
    expect(exportErrorHintKey("clipboard_unavailable")).toBe("observability.errorHintClipboardUnavailable")
    expect(exportErrorHintKey("nope")).toBe("observability.errorHintExportUnknown")
  })

  it("classifies clipboard and download failures", () => {
    expect(classifyExportError(new Error("clipboard unavailable"))).toBe("clipboard_unavailable")
    expect(classifyExportError(new Error("NotAllowedError: Write permission denied"))).toBe("clipboard_denied")
    expect(classifyExportError(new Error("Failed to execute 'createObjectURL'"))).toBe("download_failed")
    expect(classifyExportError(new Error("mystery"))).toBe("unknown")
  })

  it("formats toast and clipboard payloads", () => {
    expect(formatExportErrorToast(new Error("clipboard unavailable"), "fallback")).toBe(
      "clipboard_unavailable: clipboard unavailable",
    )
    expect(formatExportErrorToast(new Error("mystery"), "fallback")).toBe("mystery")
    const payload = exportErrorClipboardText(new Error("clipboard unavailable"), {
      scope: "logs",
      kind: "copy",
      count: 3,
    })
    expect(payload).toContain("scope: logs")
    expect(payload).toContain("kind: copy")
    expect(payload).toContain("count: 3")
    expect(payload).toContain("code: clipboard_unavailable")
  })
})
