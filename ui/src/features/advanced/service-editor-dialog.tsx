import { CircleAlertIcon } from "lucide-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  changeServiceType,
  inferServiceType,
  isServiceReady,
  prepareServiceObject,
  serviceFields,
  serviceTypes,
} from "@/features/advanced/services-form-model"
import { useConfigQuery } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import {
  isJsonObject,
  policyConfigTags,
  type JsonObject,
} from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

export interface ServiceEditorDialogProps {
  open: boolean
  item: JsonObject
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

const editorFields = serviceFields.filter((field) => field.path !== "type")

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
  const current = typeof object.type === "string" ? object.type : ""
  const values = useMemo(() => current && !serviceTypes.includes(current as (typeof serviceTypes)[number])
    ? [current, ...serviceTypes] : [...serviceTypes], [current])
  const items = useMemo(() => values.map((value) => ({ value, label: value })), [values])
  return (
    <Field data-invalid={!inferServiceType(object) || undefined}>
      <FieldLabel htmlFor="service-type">{t("advanced.services.type")}</FieldLabel>
      <Select items={items} value={current || null} onValueChange={(value) => onChange(changeServiceType(object, String(value)))}>
        <SelectTrigger id="service-type" aria-label={t("advanced.services.type")} aria-invalid={!inferServiceType(object)} className="h-8 w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {values.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  )
}

function RequirementAlert() {
  const { t } = useTranslation()
  return (
    <Alert variant="destructive">
      <CircleAlertIcon />
      <AlertTitle>{t("advanced.services.requiredTitle")}</AlertTitle>
      <AlertDescription>{t("advanced.services.requiredDescription")}</AlertDescription>
    </Alert>
  )
}

export function ServiceEditorDialog({ open, item, title, onOpenChange, onSave }: ServiceEditorDialogProps) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const initial = prepareServiceObject(item)
  const [object, setObject] = useState(initial)
  const [revision, setRevision] = useState(0)
  const [json, setJSON] = useState(() => JSON.stringify(initial, null, 2))
  const [activeTab, setActiveTab] = useState("visual")
  const [jsonInvalid, setJSONInvalid] = useState(false)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const parsed = parseObject(json)
  const visualReady = isServiceReady(object) && invalidFields.size === 0
  const jsonReady = Boolean(parsed && isServiceReady(parsed))
  const canSave = !jsonInvalid && (activeTab === "json" ? jsonReady : visualReady)
  const context = useMemo(() => ({
    inboundTags: policyConfigTags(config.data?.inbounds),
    outboundTags: policyConfigTags(config.data?.outbounds),
  }), [config.data?.inbounds, config.data?.outbounds])

  const updateVisual = (next: JsonObject) => {
    const prepared = prepareServiceObject(next)
    setObject(prepared)
    setJSON(JSON.stringify(prepared, null, 2))
    setJSONInvalid(false)
  }
  const updateJSON = (next: string) => {
    setJSON(next)
    setRevision((current) => current + 1)
    setInvalidFields(new Set())
    const parsedNext = parseObject(next)
    setJSONInvalid(!parsedNext)
    if (parsedNext) setObject(prepareServiceObject(parsedNext))
  }
  const updateValidity = (path: string, valid: boolean) => setInvalidFields((current) => {
    if (valid === !current.has(path)) return current
    const next = new Set(current)
    if (valid) next.delete(path)
    else next.add(path)
    return next
  })
  const persist = () => {
    const candidate = activeTab === "json" && parsed && jsonReady
      ? prepareServiceObject(parsed)
      : visualReady ? prepareServiceObject(object) : null
    if (!candidate) return
    onSave(candidate)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-1rem)] min-w-0 gap-3 overflow-y-auto p-3 sm:max-h-[calc(100dvh-2rem)] sm:max-w-3xl sm:gap-4 sm:p-4">
        <DialogHeader>
          <DialogTitle className="truncate">{title}</DialogTitle>
          <DialogDescription>{t("advanced.services.editorDescription")}</DialogDescription>
        </DialogHeader>
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(String(value || "visual"))} className="min-h-0 min-w-0">
          <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
            <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger>
            <TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
          </TabsList>
          <TabsContent value="visual" className="pt-3 sm:pt-4">
            <div className="flex flex-col gap-3">
              <TypeField object={object} onChange={updateVisual} />
              <PolicyFormFields
                fields={editorFields}
                object={object}
                namespace="advanced.services"
                revision={revision}
                context={context}
                onChange={updateVisual}
                onFieldValidityChange={updateValidity}
              />
              {!visualReady ? <RequirementAlert /> : null}
            </div>
          </TabsContent>
          <TabsContent value="json" className="pt-3 sm:pt-4">
            <FieldGroup>
              <Field data-invalid={!jsonReady || undefined}>
                <FieldLabel className="sr-only">{t("advanced.services.itemJSON")}</FieldLabel>
                <JsonEditor value={json} onChange={updateJSON} ariaLabel={t("advanced.services.itemJSON")} />
              </Field>
            </FieldGroup>
          </TabsContent>
        </Tabs>
        <DialogFooter className="-mx-3 -mb-3 gap-2 p-3 sm:-mx-4 sm:-mb-4 sm:p-4">
          <Button variant="outline" size="sm" className="h-8" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
          <Button size="sm" className="h-8" disabled={!canSave} onClick={persist}>{t("common.save")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
