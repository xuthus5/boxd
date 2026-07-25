import { CopyIcon, ListPlusIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { cloneJsonObject, type JsonObject } from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"
import {
  createWireGuardPeerDraft,
  isWireGuardPeerList,
  isWireGuardPeerListReady,
  isWireGuardPeerReady,
  prepareWireGuardPeer,
  summarizeWireGuardPeer,
  transformWireGuardPeerField,
  wireGuardPeerFields,
  wireGuardPeers,
} from "@/features/advanced/wireguard-peer-form-model"

interface Selection {
  index: number | null
  item: JsonObject
}

export interface WireGuardPeerEditorProps {
  value: JsonValue | undefined
  onChange: (value: JsonValue | undefined) => void
  onValidityChange?: (valid: boolean) => void
}

function replaceOrAppend(items: readonly JsonObject[], index: number | null, item: JsonObject): JsonObject[] {
  if (index === null) return [...items, item]
  return items.map((current, currentIndex) => currentIndex === index ? item : current)
}

function PeerCard({ index, item, onEdit, onCopy, onDelete }: {
  index: number
  item: JsonObject
  onEdit: () => void
  onCopy: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const summary = summarizeWireGuardPeer(item)
  const title = t("advanced.endpoints.peerTitle", { index: index + 1 })
  return <Card size="sm">
    <CardHeader className="min-w-0 gap-1.5">
      <CardTitle className="truncate">{title}</CardTitle>
      <CardDescription className="truncate" title={summary.endpoint ?? summary.publicKey}>
        {summary.endpoint ?? (summary.publicKey || t("advanced.endpoints.peerMissingKey"))}
      </CardDescription>
      <CardAction className="flex gap-1">
        <Button variant="outline" size="xs" className="h-7" aria-label={t("advanced.endpoints.peerEdit", { index: index + 1 })} onClick={onEdit}>
          <PencilIcon />
        </Button>
        <Button variant="ghost" size="xs" className="h-7" aria-label={t("advanced.endpoints.peerCopy", { index: index + 1 })} onClick={onCopy}>
          <CopyIcon />
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent className="flex flex-wrap gap-1.5">
      <Badge variant="secondary">{t("advanced.endpoints.peerAllowedCount", { count: summary.allowedIPs })}</Badge>
      {summary.keepalive !== undefined ? <Badge variant="outline">{t("advanced.endpoints.peerKeepaliveBadge", { seconds: summary.keepalive })}</Badge> : null}
      {summary.publicKey ? <Badge variant="outline" className="max-w-full truncate">{summary.publicKey}</Badge> : null}
    </CardContent>
    <CardFooter className="justify-end">
      <ConfirmAction
        trigger={<Button variant="destructive" size="xs" className="h-7" aria-label={t("advanced.endpoints.peerDelete", { index: index + 1 })}><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>}
        title={t("advanced.endpoints.peerDeleteTitle")}
        description={t("advanced.endpoints.peerDeleteDescription", { index: index + 1 })}
        confirmLabel={t("advanced.endpoints.confirmDelete")}
        confirmVariant="destructive"
        onConfirm={onDelete}
      />
    </CardFooter>
  </Card>
}

function PeerDialog({ selection, onClose, onSave }: {
  selection: Selection
  onClose: () => void
  onSave: (item: JsonObject) => void
}) {
  const { t } = useTranslation()
  const [object, setObject] = useState(() => prepareWireGuardPeer(selection.item))
  const [invalidFields, setInvalidFields] = useState<Set<string>>(new Set())
  const update = (next: JsonObject) => setObject(prepareWireGuardPeer(next))
  const updateValidity = (path: string, valid: boolean) => setInvalidFields((current) => {
    if (valid === !current.has(path)) return current
    const next = new Set(current)
    if (valid) next.delete(path)
    else next.add(path)
    return next
  })
  const canSave = isWireGuardPeerReady(object) && invalidFields.size === 0
  return <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] overflow-y-auto sm:max-w-3xl">
      <DialogHeader>
        <DialogTitle>{selection.index === null ? t("advanced.endpoints.peerAddTitle") : t("advanced.endpoints.peerEditTitle")}</DialogTitle>
        <DialogDescription>{t("advanced.endpoints.peerDialogDescription")}</DialogDescription>
      </DialogHeader>
      <div className="min-h-0 min-w-0 overflow-y-auto pr-1">
        {!isWireGuardPeerReady(object) ? <Alert variant="destructive">
          <AlertTitle>{t("advanced.endpoints.peerRequiredTitle")}</AlertTitle>
          <AlertDescription>{t("advanced.endpoints.peerRequiredDescription")}</AlertDescription>
        </Alert> : null}
        <PolicyFormFields
          fields={wireGuardPeerFields}
          object={object}
          namespace="advanced.endpoints"
          context={{}}
          onChange={update}
          onFieldValidityChange={updateValidity}
          transformField={transformWireGuardPeerField}
        />
      </div>
      <DialogFooter>
        <Button variant="outline" size="sm" onClick={onClose}>{t("common.cancel")}</Button>
        <Button size="sm" disabled={!canSave} onClick={() => onSave(prepareWireGuardPeer(object))}>{t("common.save")}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
}

export function WireGuardPeerEditor({ value, onChange, onValidityChange }: WireGuardPeerEditorProps) {
  const { t } = useTranslation()
  const validStructure = isWireGuardPeerList(value)
  const peers = useMemo(() => validStructure ? wireGuardPeers(value) : [], [validStructure, value])
  const [selection, setSelection] = useState<Selection | null>(null)
  const validityCallback = useRef(onValidityChange)
  useEffect(() => { validityCallback.current = onValidityChange }, [onValidityChange])
  useEffect(() => {
    validityCallback.current?.(isWireGuardPeerListReady(value))
    return () => validityCallback.current?.(true)
  }, [value])

  const edit = (index: number | null) => setSelection({
    index,
    item: index === null ? createWireGuardPeerDraft() : cloneJsonObject(peers[index]),
  })
  const save = (item: JsonObject) => {
    if (!selection) return
    onChange(replaceOrAppend(peers, selection.index, prepareWireGuardPeer(item)))
    setSelection(null)
  }
  const copy = (index: number) => onChange(replaceOrAppend(peers, null, cloneJsonObject(peers[index])))
  const remove = (index: number) => onChange(peers.filter((_, itemIndex) => itemIndex !== index))

  return <Field className="sm:col-span-2">
    <div className="flex items-center justify-between gap-2">
      <FieldLabel>{t("advanced.endpoints.peers")}</FieldLabel>
      <Button type="button" size="sm" className="h-8" onClick={() => edit(null)}>
        <ListPlusIcon data-icon="inline-start" />{t("advanced.endpoints.peerAdd")}
      </Button>
    </div>
    <FieldDescription>{t("advanced.endpoints.peersHelp")}</FieldDescription>
    {!validStructure ? <Alert variant="destructive">
      <AlertTitle>{t("advanced.endpoints.peerInvalidTitle")}</AlertTitle>
      <AlertDescription>{t("advanced.endpoints.peerInvalidDescription")}</AlertDescription>
    </Alert> : peers.length === 0 ? <Empty>
      <EmptyHeader><EmptyMedia variant="icon"><ListPlusIcon /></EmptyMedia>
        <EmptyTitle>{t("advanced.endpoints.peerEmpty")}</EmptyTitle>
        <EmptyDescription>{t("advanced.endpoints.peerEmptyDescription")}</EmptyDescription>
      </EmptyHeader>
      <EmptyContent><Button type="button" variant="outline" size="sm" onClick={() => edit(null)}>
        <ListPlusIcon data-icon="inline-start" />{t("advanced.endpoints.peerAdd")}
      </Button></EmptyContent>
    </Empty> : <div className="grid gap-2 sm:grid-cols-2">
      {peers.map((item, index) => <PeerCard key={index} index={index} item={item}
        onEdit={() => edit(index)} onCopy={() => copy(index)} onDelete={() => remove(index)} />)}
    </div>}
    {selection ? <PeerDialog key={`${selection.index}:${JSON.stringify(selection.item)}`} selection={selection} onClose={() => setSelection(null)} onSave={save} /> : null}
  </Field>
}
