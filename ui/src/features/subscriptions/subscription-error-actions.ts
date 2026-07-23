/** 订阅失败诊断的可操作辅助。 */

export function subscriptionErrorClipboardText(item: {
  name?: string
  url?: string
  error?: string
  error_code?: string
  error_at?: string
}): string {
  const lines = [
    item.name?.trim() ? `name: ${item.name.trim()}` : "",
    item.url?.trim() ? `url: ${item.url.trim()}` : "",
    item.error_code?.trim() ? `code: ${item.error_code.trim()}` : "",
    item.error?.trim() ? `error: ${item.error.trim()}` : "",
    item.error_at?.trim() ? `at: ${item.error_at.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}

export function isOpenableSubscriptionURL(raw?: string): boolean {
  const value = raw?.trim()
  if (!value) return false
  try {
    const parsed = new URL(value)
    return parsed.protocol === "http:" || parsed.protocol === "https:"
  } catch {
    return false
  }
}

export function subscriptionSourceURL(raw?: string): string {
  return isOpenableSubscriptionURL(raw) ? raw!.trim() : ""
}
