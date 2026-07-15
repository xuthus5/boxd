import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor } from "@/features/config/json-editor"
import { usePolicyDialogState } from "@/features/policy/policy-dialog-state"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { type JsonObject, type PolicyFieldSpec } from "@/features/policy/policy-form-model"
import { changeRuleSetType, ruleSetTypes } from "@/features/policy/route-form-model"

interface RouteRuleSetDialogProps {
  open: boolean
  item: JsonObject
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

const tagFields = [{ path: "tag", label: "ruleSetTag", required: true }] as const satisfies readonly PolicyFieldSpec[]
const formatFields = [{ path: "format", label: "format" }] as const satisfies readonly PolicyFieldSpec[]
const localFields = [{ path: "path", label: "path", required: true }] as const satisfies readonly PolicyFieldSpec[]
const remoteFields = [
  { path: "url", label: "url", required: true },
  { path: "download_detour", label: "downloadDetour" },
  { path: "update_interval", label: "updateInterval" },
] as const satisfies readonly PolicyFieldSpec[]

function optionsWithCurrent(current: string) {
  return current && !ruleSetTypes.includes(current as typeof ruleSetTypes[number])
    ? [current, ...ruleSetTypes]
    : [...ruleSetTypes]
}

function requiredFieldsPresent(object: JsonObject): boolean {
  if (typeof object.tag !== "string" || !object.tag) return false
  if (object.type === "remote") return typeof object.url === "string" && object.url.length > 0
  if (object.type === "local") return typeof object.path === "string" && object.path.length > 0
  return true
}

function TypeSelect({ object, onChange }: { object: JsonObject; onChange: (item: JsonObject) => void }) {
  const { t } = useTranslation()
  const current = String(object.type ?? "inline")
  const options = useMemo(() => optionsWithCurrent(current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <FieldGroup><Field><FieldLabel htmlFor="route-rule-set-type">{t("policy.route.ruleSetType")}</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeRuleSetType(object, String(value)))}>
      <SelectTrigger id="route-rule-set-type" aria-label={t("policy.route.ruleSetType")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select>
  </Field></FieldGroup>
}

function RuleSetFields({ object, revision, onChange }: {
  object: JsonObject; revision: number; onChange: (item: JsonObject) => void
}) {
  const { t } = useTranslation()
  const type = String(object.type ?? "inline")
  const fields = type === "remote" ? remoteFields : type === "local" ? localFields : []
  return <div className="flex flex-col gap-4">
    <PolicyFormFields fields={tagFields} object={object} namespace="policy.route" revision={revision} onChange={onChange} />
    <TypeSelect object={object} onChange={onChange} />
    <PolicyFormFields fields={formatFields} object={object} namespace="policy.route" revision={revision} onChange={onChange} />
    <PolicyFormFields fields={fields} object={object} namespace="policy.route" revision={revision} onChange={onChange} />
    {type === "inline" ? <Alert><AlertTitle>{t("policy.route.inlineTitle")}</AlertTitle><AlertDescription>{t("policy.route.inlineDescription")}</AlertDescription></Alert> : null}
  </div>
}

function AdvancedJSONField({ value, title, revision, onChange }: {
  value: string; title: string; revision: number; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("policy.route.advancedJSON")}</FieldLabel>
    <JsonEditor key={revision} value={value} onChange={onChange} ariaLabel={t("policy.route.advancedJSONLabel", { title })} />
  </Field></FieldGroup>
}

export function RouteRuleSetDialog({ open, item, title, onOpenChange, onSave }: RouteRuleSetDialogProps) {
  const { t } = useTranslation()
  const state = usePolicyDialogState(item)
  const requiredValid = requiredFieldsPresent(state.object)
  const canSave = state.jsonValid && requiredValid && state.invalidFields.size === 0
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-3xl">
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("policy.route.ruleSetDialogDescription")}</DialogDescription></DialogHeader>
      <div className="min-h-0 min-w-0 overflow-y-auto pr-1"><div className="flex min-w-0 flex-col gap-4">
        {!requiredValid ? <Alert variant="destructive"><AlertTitle>{t("policy.route.requiredTitle")}</AlertTitle>
          <AlertDescription>{t("policy.route.ruleSetRequiredDescription")}</AlertDescription></Alert> : null}
        <Tabs defaultValue="basic" className="min-w-0"><TabsList activateOnFocus className="max-w-full overflow-x-auto"><TabsTrigger value="basic">{t("policy.route.ruleSetBasicTab")}</TabsTrigger><TabsTrigger value="advanced">{t("policy.route.advancedJSON")}</TabsTrigger></TabsList>
          <TabsContent value="basic" className="pt-4" keepMounted><RuleSetFields object={state.object} revision={state.revision} onChange={state.update} /></TabsContent>
          <TabsContent value="advanced" className="pt-4" keepMounted><AdvancedJSONField value={state.value} title={title}
            revision={state.editorRevision} onChange={state.updateJSON} /></TabsContent>
        </Tabs>
      </div></div>
      <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>{t("policy.route.cancel")}</Button>
        <Button disabled={!canSave} onClick={() => { if (state.jsonValid) onSave(state.object) }}>{t("policy.route.save")}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
