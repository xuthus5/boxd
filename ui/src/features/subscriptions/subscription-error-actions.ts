/** 订阅失败诊断的可操作辅助。 */

import { toast } from "sonner"

import { copyText } from "@/features/proxy/copy-tag-button"
import {
  classifySubscriptionRequestError,
  formatSubscriptionRequestErrorToast,
  subscriptionErrorHintKey,
  subscriptionRefreshBatchClipboardText,
  subscriptionRequestErrorClipboardText,
  type SubscriptionRefreshBatchSummary,
} from "@/features/subscriptions/subscription-error"

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

export function reportSubscriptionRequestError(
  error: unknown,
  t: (key: string, values?: Record<string, string | number>) => string,
  options: { scope?: string; id?: string; name?: string; fallback?: string } = {},
) {
  const fallback = options.fallback ?? t("subscriptions.refreshFailed")
  const code = classifySubscriptionRequestError(error)
  const payload = subscriptionRequestErrorClipboardText(error, options)
  toast.error(formatSubscriptionRequestErrorToast(error, fallback), {
    description: t(subscriptionErrorHintKey(code)),
    action: payload ? {
      label: t("subscriptions.copyError"),
      onClick: () => {
        void copyText(payload).then(
          () => toast.success(t("subscriptions.errorCopied")),
          () => toast.error(t("subscriptions.errorCopyFailed")),
        )
      },
    } : undefined,
  })
}

export function reportSubscriptionRefreshBatch(
  summary: SubscriptionRefreshBatchSummary,
  message: string,
  t: (key: string, values?: Record<string, string | number>) => string,
) {
  const payload = subscriptionRefreshBatchClipboardText(summary)
  const first = summary.failedSamples[0]
  toast.error(message, {
    description: first ? t(subscriptionErrorHintKey(first.code)) : t(subscriptionErrorHintKey("unknown")),
    action: payload ? {
      label: t("subscriptions.copyError"),
      onClick: () => {
        void copyText(payload).then(
          () => toast.success(t("subscriptions.errorCopied")),
          () => toast.error(t("subscriptions.errorCopyFailed")),
        )
      },
    } : undefined,
  })
}
