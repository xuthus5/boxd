import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor } from "@/features/config/json-editor"
import { useConfigQuery } from "@/features/config/config-hooks"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { policyDNSServerTags, policyOutboundTags, type JsonObject } from "@/features/policy/policy-form-model"
import { isSharedResourceReady, parseSharedResources, sharedResourceFields, updateSharedResource, type SharedResourceSection } from "@/features/advanced/shared-resource-model"

interface ResourceDialogProps {
  section: SharedResourceSection
  item: JsonObject
  onClose: () => void
  onSave: (item: JsonObject) => void
}

function useResourceState(item: JsonObject) {
  const [raw, setRaw] = useState(JSON.stringify(item, null, 2))
  const [revision, setRevision] = useState(0)
  const [invalid, setInvalid] = useState<Set<string>>(new Set())
  const parsed = parseSharedResources(`[${raw}]`)
  const object = parsed?.length === 1 ? parsed[0] : null
  const update = (next: JsonObject) => setRaw(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => { setRaw(next); setRevision((value) => value + 1); setInvalid(new Set()) }
  const onValidity = (path: string, valid: boolean) => setInvalid((current) => {
    if (valid === !current.has(path)) return current
    const next = new Set(current)
    if (valid) next.delete(path)
    else next.add(path)
    return next
  })
  return { raw, revision, invalid, object, update, updateJSON, onValidity }
}

function ResourceForm({ state, section }: { state: ReturnType<typeof useResourceState>; section: SharedResourceSection }) {
  const config = useConfigQuery()
  if (!state.object) return null
  const object = state.object
  return <PolicyFormFields fields={sharedResourceFields(section, object)} object={object} namespace="kernel114.fields"
    context={{ outboundTags: policyOutboundTags(config.data), dnsServerTags: policyDNSServerTags(config.data?.dns) }}
    revision={state.revision} onFieldValidityChange={state.onValidity}
    onChange={(next) => state.update(updateSharedResource(section, object, next))} />
}

export function SharedResourceDialog({ section, item, onClose, onSave }: ResourceDialogProps) {
  const { t } = useTranslation()
  const state = useResourceState(item)
  const canSave = state.object && isSharedResourceReady(section, state.object) && state.invalid.size === 0
  return <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-3xl">
      <DialogHeader><DialogTitle>{t(`kernel114.${section}.edit`)}</DialogTitle>
        <DialogDescription>{t("kernel114.editorDescription")}</DialogDescription></DialogHeader>
      <Tabs defaultValue="visual">
        <TabsList><TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger><TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger></TabsList>
        <TabsContent value="visual" className="pt-4" keepMounted><ResourceForm state={state} section={section} /></TabsContent>
        <TabsContent value="json" className="pt-4"><JsonEditor value={state.raw} onChange={state.updateJSON} ariaLabel={t("kernel114.itemJSON")} /></TabsContent>
      </Tabs>
      <DialogFooter><Button variant="outline" onClick={onClose}>{t("common.cancel")}</Button>
        <Button disabled={!canSave} onClick={() => { if (state.object) onSave(state.object) }}>{t("common.save")}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
