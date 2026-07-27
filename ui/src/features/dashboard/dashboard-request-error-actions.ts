/** Dashboard 服务/维护请求失败的可操作 toast。 */

import { toast } from "sonner"

import {
  classifyDashboardRequestError,
  dashboardRequestErrorClipboardText,
  dashboardRequestErrorHintKey,
  formatDashboardRequestErrorToast,
} from "@/features/dashboard/dashboard-request-error"
import { copyText } from "@/lib/clipboard"

export function reportDashboardRequestError(
  error: unknown,
  t: (key: string, values?: Record<string, string | number>) => string,
  options: { scope?: string; action?: string; fallback?: string } = {},
) {
  const fallback = options.fallback ?? t("dashboard.requestFailed")
  const code = classifyDashboardRequestError(error)
  const payload = dashboardRequestErrorClipboardText(error, options)
  toast.error(formatDashboardRequestErrorToast(error, fallback), {
    description: t(dashboardRequestErrorHintKey(code)),
    action: payload ? {
      label: t("dashboard.copyRequestError"),
      onClick: () => {
        void copyText(payload).then(
          () => toast.success(t("dashboard.requestErrorCopied")),
          () => toast.error(t("dashboard.requestErrorCopyFailed")),
        )
      },
    } : undefined,
  })
}
