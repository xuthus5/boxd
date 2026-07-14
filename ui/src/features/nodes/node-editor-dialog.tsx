import { useMutation, useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { JsonEditor } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import { api } from "@/lib/api/endpoints"
import type { JsonValue, Outbound } from "@/lib/api/types"

interface Props { tag: string; onClose: () => void; onSaved: () => void }

function parseJSON(value: string): JsonValue | null {
  if (!isValidJSON(value)) return null
  return JSON.parse(value) as JsonValue
}

function NodeEditorForm({ node, originalTag, onSaved }: { node: Outbound; originalTag: string; onSaved: () => void }) {
  const { t } = useTranslation()
  const [tag, setTag] = useState(node.tag)
  const [type, setType] = useState(node.type)
  const [server, setServer] = useState(node.server ?? "")
  const [port, setPort] = useState(String(node.port ?? ""))
  const [config, setConfig] = useState(() => JSON.stringify(node.raw ?? {}, null, 2))
  const parsed = parseJSON(config)
  const save = useMutation({
    mutationFn: () => api.nodes.update(originalTag, { tag, type, server, port: Number(port), config: parsed! }),
    onSuccess: () => { toast.success(t("nodes.updated")); onSaved() },
    onError: (error: Error) => toast.error(error.message),
  })
  return <>
    <FieldGroup className="grid gap-4 sm:grid-cols-2">
      <Field><FieldLabel htmlFor="node-tag">Tag</FieldLabel><Input id="node-tag" value={tag} onChange={(event) => setTag(event.target.value)} /></Field>
      <Field><FieldLabel htmlFor="node-type">{t("common.type")}</FieldLabel><Input id="node-type" value={type} onChange={(event) => setType(event.target.value)} /></Field>
      <Field><FieldLabel htmlFor="node-server">{t("nodes.server")}</FieldLabel><Input id="node-server" value={server} onChange={(event) => setServer(event.target.value)} /></Field>
      <Field><FieldLabel htmlFor="node-port">{t("common.port")}</FieldLabel><Input id="node-port" type="number" value={port} onChange={(event) => setPort(event.target.value)} /></Field>
    </FieldGroup>
    <FieldGroup><Field><FieldLabel className="sr-only">{t("nodes.advancedJSON")}</FieldLabel><JsonEditor value={config} onChange={setConfig} ariaLabel={t("nodes.advancedJSON")} /></Field></FieldGroup>
    <DialogFooter><Button disabled={!tag || !type || parsed === null || save.isPending} onClick={() => save.mutate()}>{t("common.save")}</Button></DialogFooter>
  </>
}

export function NodeEditorDialog({ tag, onClose, onSaved }: Props) {
  const { t } = useTranslation()
  const query = useQuery({ queryKey: ["nodes", tag], queryFn: () => api.nodes.get(tag) })
  return <Dialog open onOpenChange={(open) => { if (!open) onClose() }}><DialogContent className="max-w-3xl">
    <DialogHeader><DialogTitle>{t("nodes.edit")}</DialogTitle><DialogDescription>{t("nodes.editDescription")}</DialogDescription></DialogHeader>
    {query.isLoading ? <Skeleton className="h-64 w-full" /> : null}
    {query.error ? <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert> : null}
    {query.data ? <NodeEditorForm node={query.data} originalTag={tag} onSaved={onSaved} /> : null}
  </DialogContent></Dialog>
}
