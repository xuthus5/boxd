/** 观测页导出/复制失败的可操作 toast。 */

import { toast } from "sonner"

import {
  classifyExportError,
  exportErrorClipboardText,
  exportErrorHintKey,
  formatExportErrorToast,
} from "@/features/observability/export-error"
import { copyText } from "@/features/proxy/copy-tag-button"

export function reportExportError(
  error: unknown,
  t: (key: string, values?: Record<string, string | number>) => string,
  options: {
    scope?: string
    kind?: string
    count?: number
    filename?: string
    fallback?: string
  } = {},
) {
  const fallback = options.fallback ?? t("observability.exportFailed")
  const code = classifyExportError(error)
  const payload = exportErrorClipboardText(error, options)
  toast.error(formatExportErrorToast(error, fallback), {
    description: t(exportErrorHintKey(code)),
    action: payload ? {
      label: t("observability.copyExportError"),
      onClick: () => {
        void copyText(payload).then(
          () => toast.success(t("observability.exportErrorCopied")),
          () => toast.error(t("observability.exportErrorCopyFailed")),
        )
      },
    } : undefined,
  })
}
