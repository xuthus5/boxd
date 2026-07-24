import { useTranslation } from "react-i18next"

import type { JsonObject } from "@/features/policy/policy-form-model"
import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { RouteRuleSetDialog } from "@/features/policy/route-rule-set-dialog"
import type { RouteRuleMetadata } from "@/lib/api/types"

export interface RouteEditorSelection {
  kind: "rule" | "rule-set"
  index: number | null
  item: JsonObject
  metadata?: RouteRuleMetadata
  jumpPath?: string
}

interface RouteVisualDialogsProps {
  selection: RouteEditorSelection | null
  onClose: () => void
  onClearJumpPath: () => void
  onSave: (item: JsonObject, metadata?: RouteRuleMetadata) => void
}

export function RouteVisualDialogs({
  selection, onClose, onClearJumpPath, onSave,
}: RouteVisualDialogsProps) {
  const { t } = useTranslation()
  if (!selection) return null
  if (selection.kind === "rule") {
    return (
      <RouteRuleDialog
        key={`${selection.index}:${JSON.stringify(selection.item)}`}
        open
        item={selection.item}
        metadata={selection.metadata}
        title={selection.index === null
          ? t("policy.route.addRuleTitle")
          : t("policy.route.editRuleTitle", { index: (selection.index ?? 0) + 1 })}
        jumpPath={selection.jumpPath}
        onJumpPathHandled={onClearJumpPath}
        onOpenChange={(open) => { if (!open) onClose() }}
        onSave={onSave}
      />
    )
  }
  return (
    <RouteRuleSetDialog
      key={`${selection.index}:${JSON.stringify(selection.item)}`}
      open
      item={selection.item}
      title={selection.index === null
        ? t("policy.route.addRuleSetTitle")
        : t("policy.route.editRuleSetTitle")}
      jumpPath={selection.jumpPath}
      onJumpPathHandled={onClearJumpPath}
      onOpenChange={(open) => { if (!open) onClose() }}
      onSave={onSave}
    />
  )
}
