/** 设置页请求失败的可操作 toast。 */

import { toast } from "sonner"

import { copyText } from "@/lib/clipboard"
import {
  classifySettingsRequestError,
  formatSettingsRequestErrorToast,
  settingsRequestErrorClipboardText,
  settingsRequestErrorHintKey,
} from "@/features/settings/settings-request-error"

export function reportSettingsRequestError(
  error: unknown,
  t: (key: string, values?: Record<string, string | number>) => string,
  options: { scope?: string; field?: string; fallback?: string } = {},
) {
  const fallback = options.fallback ?? t("settings.requestFailed")
  const code = classifySettingsRequestError(error)
  const payload = settingsRequestErrorClipboardText(error, options)
  toast.error(formatSettingsRequestErrorToast(error, fallback), {
    description: t(settingsRequestErrorHintKey(code)),
    action: payload ? {
      label: t("settings.copyRequestError"),
      onClick: () => {
        void copyText(payload).then(
          () => toast.success(t("settings.requestErrorCopied")),
          () => toast.error(t("settings.requestErrorCopyFailed")),
        )
      },
    } : undefined,
  })
}
