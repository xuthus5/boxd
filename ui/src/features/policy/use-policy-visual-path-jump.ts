import { useEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { JsonObject } from "@/features/policy/policy-form-model"
import {
  policyDialogSelectionFromPath,
  type PolicyDialogSelection,
  type PolicySectionPath,
} from "@/features/policy/policy-path"
import type { RouteRuleMetadata } from "@/lib/api/types"

interface PolicyPathLists {
  rules?: readonly JsonObject[]
  ruleSets?: readonly JsonObject[]
  servers?: readonly JsonObject[]
  metadata?: readonly RouteRuleMetadata[]
}

interface UsePolicyVisualPathJumpOptions {
  section: PolicySectionPath
  jumpPath?: string | null
  onJumpPathHandled?: () => void
  lists: PolicyPathLists
  onSelect: (selection: PolicyDialogSelection) => void
}

/** Open the matching policy item dialog when a list-item config path is provided. */
export function usePolicyVisualPathJump({
  section,
  jumpPath,
  onJumpPathHandled,
  lists,
  onSelect,
}: UsePolicyVisualPathJumpOptions) {
  const { t } = useTranslation()
  const listsRef = useRef(lists)
  const onSelectRef = useRef(onSelect)
  const handledRef = useRef(onJumpPathHandled)
  listsRef.current = lists
  onSelectRef.current = onSelect
  handledRef.current = onJumpPathHandled

  useEffect(() => {
    if (!jumpPath) return
    const next = policyDialogSelectionFromPath(jumpPath, section, listsRef.current)
    if (!next) {
      toast.message(t("config.pathNotFound", { path: jumpPath }))
      handledRef.current?.()
      return
    }
    onSelectRef.current(next)
    handledRef.current?.()
  }, [jumpPath, section, t])
}
