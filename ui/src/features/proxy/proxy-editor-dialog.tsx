import { InboundEditorDialog } from "@/features/proxy/inbound-editor-dialog"
import { OutboundEditorDialog } from "@/features/proxy/outbound-editor-dialog"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

interface ProxyEditorDialogProps {
  title: string
  kind: "inbounds" | "outbounds"
  item: JsonObject
  index?: number
  onClose: () => void
  onSave: (item: JsonObject) => void
  jumpPath?: string | null
  onJumpPathHandled?: () => void
  reportError?: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
}

export function ProxyEditorDialog({
  title, kind, item, index = -1, onClose, onSave, jumpPath, onJumpPathHandled, reportError, clearSaveError,
}: ProxyEditorDialogProps) {
  if (kind === "inbounds") {
    return (
      <InboundEditorDialog
        title={title}
        item={item}
        index={index}
        onClose={onClose}
        onSave={onSave}
        jumpPath={jumpPath}
        onJumpPathHandled={onJumpPathHandled}
        reportError={reportError}
        clearSaveError={clearSaveError}
      />
    )
  }
  return (
    <OutboundEditorDialog
      title={title}
      item={item}
      index={index}
      onClose={onClose}
      onSave={onSave}
      jumpPath={jumpPath}
      onJumpPathHandled={onJumpPathHandled}
      reportError={reportError}
      clearSaveError={clearSaveError}
    />
  )
}
