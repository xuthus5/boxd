import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { JsonEditor } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

interface ProxyEditorDialogProps {
  title: string
  kind: "inbounds" | "outbounds"
  item: JsonObject
  onClose: () => void
  onSave: (item: JsonObject) => void
}

function parseObject(value: string) {
  const parsed: unknown = JSON.parse(value)
  return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed as JsonObject : null
}

function editorObject(value: string) {
  return isValidJSON(value) ? parseObject(value) : null
}

function fieldValue(object: JsonObject | null, key: string) {
  return String(object?.[key] ?? "")
}

const fieldKeys = {
  inbounds: { address: "listen", port: "listen_port" },
  outbounds: { address: "server", port: "server_port" },
} as const

export function ProxyEditorDialog({ title, kind, item, onClose, onSave }: ProxyEditorDialogProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(item, null, 2))
  const object = editorObject(value)
  const keys = fieldKeys[kind]
  const update = (key: string, fieldValue: string | number) => {
    setValue(JSON.stringify({ ...(object ?? {}), [key]: fieldValue }, null, 2))
  }
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-3xl">
        <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("proxy.editorDescription")}</DialogDescription></DialogHeader>
        <FieldGroup className="grid gap-4 sm:grid-cols-2">
            <Field><FieldLabel htmlFor="proxy-tag">Tag</FieldLabel><Input id="proxy-tag" value={fieldValue(object, "tag")} onChange={(event) => update("tag", event.target.value)} /></Field>
            <Field><FieldLabel htmlFor="proxy-type">{t("common.type")}</FieldLabel><Input id="proxy-type" value={fieldValue(object, "type")} onChange={(event) => update("type", event.target.value)} /></Field>
            <Field><FieldLabel htmlFor="proxy-address">{t("common.address")}</FieldLabel><Input id="proxy-address" value={fieldValue(object, keys.address)} onChange={(event) => update(keys.address, event.target.value)} /></Field>
            <Field><FieldLabel htmlFor="proxy-port">{t("common.port")}</FieldLabel><Input id="proxy-port" type="number" value={fieldValue(object, keys.port)} onChange={(event) => update(keys.port, Number(event.target.value))} /></Field>
        </FieldGroup>
        <FieldGroup><Field><FieldLabel className="sr-only">{t("proxy.advancedJSON")}</FieldLabel><JsonEditor value={value} onChange={setValue} ariaLabel={`${title} JSON`} /></Field></FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>{t("common.cancel")}</Button>
          <Button disabled={!object} onClick={() => { if (object) onSave(object) }}>{t("common.save")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
