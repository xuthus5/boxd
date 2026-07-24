import { useCallback, useEffect, type RefObject } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { JsonEditorHandle } from "@/features/config/json-editor"
import {
  proxyItemRelativePaths,
  type ProxySection,
} from "@/features/proxy/proxy-path"

/** Reveal a config/error path inside the open proxy item JSON editor. */
export function useProxyItemPathReveal(
  editorRef: RefObject<JsonEditorHandle | null>,
  section: ProxySection,
  index: number,
  setActiveTab: (tab: string) => void,
) {
  const { t } = useTranslation()
  return useCallback((path: string) => {
    setActiveTab("advanced")
    const candidates = proxyItemRelativePaths(path, section, index)
    if (candidates.length === 0) {
      toast.message(t("config.pathNotFound", { path }))
      return false
    }
    const tryReveal = () => candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => {
      if (!tryReveal()) toast.message(t("config.pathNotFound", { path }))
    }, 50)
    return true
  }, [editorRef, index, section, setActiveTab, t])
}

/** Apply list-page jumpPath (relative or full) once into the open dialog. */
export function useProxyJumpPath(
  jumpPath: string | null | undefined,
  onJumpPathHandled: (() => void) | undefined,
  reveal: (path: string) => boolean,
) {
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
}
