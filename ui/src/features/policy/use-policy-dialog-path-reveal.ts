import { useCallback, useEffect, type RefObject } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { JsonEditorHandle } from "@/features/config/json-editor"

/** Reveal a relative error path inside a policy item dialog JSON editor. */
export function usePolicyDialogPathReveal(
  editorRef: RefObject<JsonEditorHandle | null>,
  setActiveTab: (tab: string) => void,
  jumpPath: string | null | undefined,
  onJumpPathHandled?: () => void,
) {
  const { t } = useTranslation()
  const reveal = useCallback((path: string) => {
    setActiveTab("advanced")
    const relative = path.trim()
    if (!relative) return false
    const tryReveal = () => editorRef.current?.revealPath(relative) ?? false
    if (tryReveal()) return true
    window.setTimeout(() => {
      if (!tryReveal()) toast.message(t("config.pathNotFound", { path: relative }))
    }, 50)
    return true
  }, [editorRef, setActiveTab, t])

  useEffect(() => {
    if (jumpPath === undefined || jumpPath === null) return
    const text = jumpPath.trim()
    if (!text) {
      onJumpPathHandled?.()
      return
    }
    reveal(text)
    onJumpPathHandled?.()
  }, [jumpPath, onJumpPathHandled, reveal])

  return reveal
}
