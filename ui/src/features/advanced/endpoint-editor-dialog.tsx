import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  changeEndpointType,
  endpointDialerFields,
  endpointIdentityFields,
  endpointTypes,
  inferEndpointType,
  isEndpointReady,
  prepareEndpointObject,
  tailscaleFields,
  wireGuardFields,
} from "@/features/advanced/endpoints-form-model"
import { JsonEditor } from "@/features/config/json-editor"
import { useConfigQuery } from "@/features/config/config-hooks"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import {
  isJsonObject,
  policyConfigTags,
  policyDNSServerTags,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

export interface EndpointEditorDialogProps {
  open: boolean
  item: JsonObject
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

function parseObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

function TypeField({ object, onChange }: { object: JsonObject; onChange: (item: JsonObject) => void }) {
  const { t } = useTranslation()
  const current = inferEndpointType(object)
  const items = useMemo(() => endpointTypes.map((value) => ({ value, label: value })), [])
  return (
    <Field>
      <FieldLabel htmlFor="endpoint-type">{t("advanced.endpoints.type")}</FieldLabel>
      <Select items={items} value={current} onValueChange={(value) => onChange(changeEndpointType(object, String(value)))}>
        <SelectTrigger id="endpoint-type" aria-label={t("advanced.endpoints.type")} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {endpointTypes.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  )
}

function sectionFields(section: "basic" | "type" | "dialer"): readonly PolicyFieldSpec[] {
  if (section === "basic") return endpointIdentityFields.filter((field) => field.path !== "type")
  if (section === "type") return [...wireGuardFields, ...tailscaleFields]
  return endpointDialerFields
}

export function EndpointEditorDialog({ open, item, title, onOpenChange, onSave }: EndpointEditorDialogProps) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const [object, setObject] = useState(() => prepareEndpointObject(item))
  const [revision, setRevision] = useState(0)
  const [json, setJSON] = useState(() => JSON.stringify(item, null, 2))
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const parsed = parseObject(json)
  const visualReady = isEndpointReady(object) && invalidFields.size === 0
  const jsonReady = Boolean(parsed && isEndpointReady(parsed))
  const context = useMemo(() => ({
    outboundTags: policyConfigTags(config.data?.outbounds, typeof object.tag === "string" ? object.tag : undefined),
    dnsServerTags: policyDNSServerTags(config.data?.dns),
  }), [config.data?.dns, config.data?.outbounds, object.tag])

  const updateVisual = (next: JsonObject) => {
    const prepared = prepareEndpointObject(next)
    setObject(prepared)
    setJSON(JSON.stringify(prepared, null, 2))
  }
  const updateJSON = (next: string) => {
    setJSON(next)
    setRevision((current) => current + 1)
    setInvalidFields(new Set())
    const parsedNext = parseObject(next)
    if (parsedNext) setObject(prepareEndpointObject(parsedNext))
  }
  const updateValidity = (path: string, valid: boolean) => {
    setInvalidFields((current) => {
      const next = new Set(current)
      if (valid) next.delete(path)
      else next.add(path)
      return next
    })
  }
  const persist = () => {
    const candidate = parsed && isEndpointReady(parsed) ? prepareEndpointObject(parsed)
      : visualReady ? prepareEndpointObject(object) : null
    if (!candidate) return
    onSave(candidate)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{t("advanced.endpoints.editorDescription")}</DialogDescription>
        </DialogHeader>
        <Tabs defaultValue="visual" className="min-w-0">
          <TabsList activateOnFocus className="max-w-full">
            <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger>
            <TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
          </TabsList>
          <TabsContent value="visual" className="pt-4">
            <div className="flex flex-col gap-4">
              <TypeField object={object} onChange={updateVisual} />
              <PolicyFormFields
                fields={sectionFields("basic")}
                object={object}
                namespace="advanced.endpoints"
                revision={revision}
                context={context}
                onChange={updateVisual}
                onFieldValidityChange={updateValidity}
              />
              <PolicyFormFields
                fields={sectionFields("type")}
                object={object}
                namespace="advanced.endpoints"
                revision={revision}
                context={context}
                onChange={updateVisual}
                onFieldValidityChange={updateValidity}
              />
              <PolicyFormFields
                fields={sectionFields("dialer")}
                object={object}
                namespace="advanced.endpoints"
                revision={revision}
                context={context}
                onChange={updateVisual}
                onFieldValidityChange={updateValidity}
              />
            </div>
          </TabsContent>
          <TabsContent value="json" className="pt-4">
            <FieldGroup>
              <Field>
                <FieldLabel className="sr-only">{t("advanced.endpoints.itemJSON")}</FieldLabel>
                <JsonEditor value={json} onChange={updateJSON} ariaLabel={t("advanced.endpoints.itemJSON")} />
              </Field>
            </FieldGroup>
          </TabsContent>
        </Tabs>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
          <Button disabled={!(visualReady || jsonReady)} onClick={persist}>{t("common.save")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

