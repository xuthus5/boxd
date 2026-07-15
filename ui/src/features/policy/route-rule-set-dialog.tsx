import { useMemo, useState } from "react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor } from "@/features/config/json-editor"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { isJsonObject, type JsonObject, type PolicyFieldSpec } from "@/features/policy/policy-form-model"
import { changeRuleSetType, ruleSetTypes } from "@/features/policy/route-form-model"
import type { JsonValue } from "@/lib/api/types"

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

function parseObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch (error) {
    void error
    return null
  }
}

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
  const current = String(object.type ?? "inline")
  const options = useMemo(() => optionsWithCurrent(current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <FieldGroup><Field><FieldLabel htmlFor="route-rule-set-type">规则集类型</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeRuleSetType(object, String(value)))}>
      <SelectTrigger id="route-rule-set-type" aria-label="规则集类型" className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select>
  </Field></FieldGroup>
}

function RuleSetFields({ object, revision, onChange }: {
  object: JsonObject; revision: number; onChange: (item: JsonObject) => void
}) {
  const type = String(object.type ?? "inline")
  const fields = type === "remote" ? remoteFields : type === "local" ? localFields : []
  return <>
    <PolicyFormFields fields={tagFields} object={object} namespace="policy.route" revision={revision} onChange={onChange} />
    <TypeSelect object={object} onChange={onChange} />
    <PolicyFormFields fields={formatFields} object={object} namespace="policy.route" revision={revision} onChange={onChange} />
    <PolicyFormFields fields={fields} object={object} namespace="policy.route" revision={revision} onChange={onChange} />
    {type === "inline" ? <Alert><AlertTitle>Inline 规则集</AlertTitle><AlertDescription>复杂 inline 规则内容请在高级 JSON 中维护。</AlertDescription></Alert> : null}
  </>
}

function AdvancedJSONField({ value, title, onChange }: {
  value: string; title: string; onChange: (value: string) => void
}) {
  return <FieldGroup><Field><FieldLabel className="sr-only">高级 JSON</FieldLabel>
    <JsonEditor value={value} onChange={onChange} ariaLabel={`${title} JSON`} />
  </Field></FieldGroup>
}

export function RouteRuleSetDialog({ open, item, title, onOpenChange, onSave }: RouteRuleSetDialogProps) {
  const [value, setValue] = useState(() => JSON.stringify(item, null, 2))
  const [revision, setRevision] = useState(0)
  const object = parseObject(value)
  const update = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => { setValue(next); setRevision((current) => current + 1) }
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-3xl">
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>按类型编辑规则集，完整对象可在高级 JSON 中维护。</DialogDescription></DialogHeader>
      <div className="min-h-0 overflow-y-auto pr-1">
        {object ? <Tabs defaultValue="basic"><TabsList><TabsTrigger value="basic">基础</TabsTrigger><TabsTrigger value="advanced">高级 JSON</TabsTrigger></TabsList>
          <TabsContent value="basic" className="pt-4" keepMounted><RuleSetFields object={object} revision={revision} onChange={update} /></TabsContent>
          <TabsContent value="advanced" className="pt-4"><AdvancedJSONField value={value} title={title} onChange={updateJSON} /></TabsContent>
        </Tabs> : <AdvancedJSONField value={value} title={title} onChange={updateJSON} />}
      </div>
      <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
        <Button disabled={!object || !requiredFieldsPresent(object)} onClick={() => { if (object) onSave(object) }}>保存</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
