/** 配置保存失败的可操作 toast（与 alert 共用结构化错误态）。 */

import { toast } from "sonner"

import {
  configSaveErrorClipboardText,
  configSaveErrorHintKey,
  type ConfigSaveErrorState,
} from "@/features/config/config-save-error"
import { copyText } from "@/lib/clipboard"
import { i18n } from "@/i18n"

export function formatConfigSaveErrorToast(state: ConfigSaveErrorState): string {
  if (state.code && state.code !== "unknown") {
    return `${state.code}: ${state.message}`
  }
  return state.message
}

export function reportConfigSaveErrorToast(state: ConfigSaveErrorState) {
  const payload = configSaveErrorClipboardText(state)
  toast.error(formatConfigSaveErrorToast(state), {
    description: i18n.t(configSaveErrorHintKey(state.code)),
    action: payload
      ? {
          label: i18n.t("config.copySaveError"),
          onClick: () => {
            void copyText(payload).then(
              () => toast.success(i18n.t("config.saveErrorCopied")),
              () => toast.error(i18n.t("config.saveErrorCopyFailed")),
            )
          },
        }
      : undefined,
  })
}
