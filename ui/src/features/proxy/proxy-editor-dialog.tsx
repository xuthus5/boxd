import { InboundEditorDialog } from "@/features/proxy/inbound-editor-dialog"
import { OutboundEditorDialog } from "@/features/proxy/outbound-editor-dialog"
import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

interface ProxyEditorDialogProps {
  title: string
  kind: "inbounds" | "outbounds"
  item: JsonObject
  onClose: () => void
  onSave: (item: JsonObject) => void
}

export function ProxyEditorDialog({ title, kind, item, onClose, onSave }: ProxyEditorDialogProps) {
  if (kind === "inbounds") return <InboundEditorDialog title={title} item={item} onClose={onClose} onSave={onSave} />
  return <OutboundEditorDialog title={title} item={item} onClose={onClose} onSave={onSave} />
}
