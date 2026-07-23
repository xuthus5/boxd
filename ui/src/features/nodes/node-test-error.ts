import type { TestResult } from "@/lib/api/types"

/** 节点测速失败时的剪贴板诊断文本。 */
export function nodeTestErrorClipboardText(result: Pick<TestResult, "tag" | "test_type" | "error" | "timestamp" | "success">): string {
  if (result.success) return ""
  const lines = [
    result.tag?.trim() ? `tag: ${result.tag.trim()}` : "",
    result.test_type?.trim() ? `test: ${result.test_type.trim()}` : "",
    result.error?.trim() ? `error: ${result.error.trim()}` : "",
    result.timestamp?.trim() ? `at: ${result.timestamp.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}

export function nodeTestErrorLabel(result: Pick<TestResult, "error">, fallback: string): string {
  const message = result.error?.trim()
  return message || fallback
}
