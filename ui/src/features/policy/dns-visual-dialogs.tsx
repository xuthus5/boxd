import { useTranslation } from "react-i18next"

import type { JsonObject } from "@/features/policy/policy-form-model"
import { DNSRuleDialog } from "@/features/policy/dns-rule-dialog"
import { DNSServerDialog } from "@/features/policy/dns-server-dialog"

export interface DNSEditorSelection {
  kind: "server" | "rule"
  index: number | null
  item: JsonObject
  jumpPath?: string
}

interface DNSVisualDialogsProps {
  selection: DNSEditorSelection | null
  serverTags: readonly string[]
  onClose: () => void
  onClearJumpPath: () => void
  onSave: (item: JsonObject) => void
}

export function DNSVisualDialogs({
  selection, serverTags, onClose, onClearJumpPath, onSave,
}: DNSVisualDialogsProps) {
  const { t } = useTranslation()
  if (!selection) return null
  if (selection.kind === "server") {
    return (
      <DNSServerDialog
        key={`${selection.index}:${JSON.stringify(selection.item)}`}
        open
        item={selection.item}
        title={selection.index === null
          ? t("policy.dns.addServerTitle")
          : t("policy.dns.editServerTitle")}
        jumpPath={selection.jumpPath}
        onJumpPathHandled={onClearJumpPath}
        onOpenChange={(open) => { if (!open) onClose() }}
        onSave={onSave}
      />
    )
  }
  return (
    <DNSRuleDialog
      key={`${selection.index}:${JSON.stringify(selection.item)}`}
      open
      item={selection.item}
      title={selection.index === null
        ? t("policy.dns.addRuleTitle")
        : t("policy.dns.editRuleTitle", { index: (selection.index ?? 0) + 1 })}
      serverTags={serverTags}
      jumpPath={selection.jumpPath}
      onJumpPathHandled={onClearJumpPath}
      onOpenChange={(open) => { if (!open) onClose() }}
      onSave={onSave}
    />
  )
}
