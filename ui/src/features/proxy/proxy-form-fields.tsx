import { useCallback, useEffect, useId, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { getPath, type FieldSpec, type FieldTransform, type JsonObject, setPath } from "@/features/proxy/proxy-form-model"
import type { JsonValue } from "@/lib/api/types"

interface ProxyFormFieldsProps {
  fields: FieldSpec[]
  object: JsonObject
  namespace: "proxy.inbound" | "proxy.outbound"
  revision?: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange?: (path: string, valid: boolean) => void
  transformField?: FieldTransform
}

function textValue(value: JsonValue | undefined) {
  if (Array.isArray(value)) return value.map(String).join("\n")
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean" ? String(value) : ""
}

function parseList(value: string) {
  const values = value.split(/[\n,]/).map((item) => item.trim()).filter(Boolean)
  return values.length ? values : undefined
}

function BooleanField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  const id = useId()
  return <Field orientation="horizontal"><FieldLabel htmlFor={id}>{label}</FieldLabel><Switch id={id} aria-label={label} checked={checked} onCheckedChange={onChange} /></Field>
}

function SelectField({ field, label, namespace, value, onChange }: { field: FieldSpec; label: string; namespace: ProxyFormFieldsProps["namespace"]; value: string; onChange: (value: string) => void }) {
  const { t } = useTranslation()
  const id = useId()
  const options = useMemo(() => field.options?.filter(Boolean) ?? [], [field.options])
  const unset = "__unset__"
  const optionLabel = useCallback((option: string) => option === "true" ? t(`${namespace}.enabled`) : option === "false" ? t(`${namespace}.disabled`) : option, [namespace, t])
  const items = useMemo(() => [{ value: unset, label: t(`${namespace}.notSet`) }, ...options.map((option) => ({ value: option, label: optionLabel(option) }))], [namespace, optionLabel, options, t])
  return <Field><FieldLabel htmlFor={id}>{label}</FieldLabel><Select items={items} value={value || unset} onValueChange={(next) => onChange(next === unset ? "" : String(next))}><SelectTrigger id={id} aria-label={label} className="w-full"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem value={unset}>{t(`${namespace}.notSet`)}</SelectItem>{options.map((option) => <SelectItem key={option} value={option}>{optionLabel(option)}</SelectItem>)}</SelectGroup></SelectContent></Select></Field>
}

function TextField({ field, label, value, onChange }: { field: FieldSpec; label: string; value: string; onChange: (value: string) => void }) {
  const id = useId()
  const area = field.kind === "textarea" || field.kind === "list" || field.kind === "number-list" || field.kind === "users"
  const control = area
    ? <Textarea id={id} aria-label={label} value={value} onChange={(event) => onChange(event.target.value)} />
    : <Input id={id} aria-label={label} type={field.kind === "number" ? "number" : "text"} value={value} onChange={(event) => onChange(event.target.value)} />
  return <Field><FieldLabel htmlFor={id}>{label}</FieldLabel>{control}</Field>
}

function JSONField({ field, label, namespace, revision = 0, value, array, onChange, onFieldValidityChange }: { field: FieldSpec; label: string; namespace: ProxyFormFieldsProps["namespace"]; revision?: number; value: JsonValue | undefined; array: boolean; onChange: (value: JsonValue | undefined) => void; onFieldValidityChange?: (path: string, valid: boolean) => void }) {
  const { t } = useTranslation()
  const id = useId()
  const serialized = value === undefined ? "" : JSON.stringify(value, null, 2)
  const sourceKey = `${revision}:${serialized}`
  const [raw, setRaw] = useState(() => serialized)
  const [source, setSource] = useState(() => sourceKey)
  const [invalid, setInvalid] = useState(false)
  if (source !== sourceKey) { setSource(sourceKey); setRaw(serialized); setInvalid(false) }
  useEffect(() => { onFieldValidityChange?.(field.path, true); return () => onFieldValidityChange?.(field.path, true) }, [field.path, onFieldValidityChange, sourceKey])
  const update = (next: string) => {
    setRaw(next)
    if (!next.trim()) { setSource(`${revision}:`); setInvalid(false); onFieldValidityChange?.(field.path, true); onChange(undefined); return }
    try {
      const parsed: unknown = JSON.parse(next)
      const valid = array ? Array.isArray(parsed) && parsed.every((item) => item && typeof item === "object" && !Array.isArray(item)) : Boolean(parsed && typeof parsed === "object" && !Array.isArray(parsed))
      setInvalid(!valid); onFieldValidityChange?.(field.path, valid)
      if (valid) { setSource(`${revision}:${JSON.stringify(parsed, null, 2)}`); onChange(parsed as JsonValue) }
    } catch { setInvalid(true); onFieldValidityChange?.(field.path, false) }
  }
  return <Field data-invalid={invalid}><FieldLabel htmlFor={id}>{label}</FieldLabel><Textarea id={id} aria-label={label} aria-invalid={invalid} value={raw} onChange={(event) => update(event.target.value)} /><FieldDescription>{invalid ? t(`${namespace}.invalidStructuredJSON`) : t(array ? `${namespace}.usersJSONHint` : `${namespace}.jsonObjectHint`)}</FieldDescription></Field>
}

function defaultUpdate(object: JsonObject, field: FieldSpec, raw: string): JsonObject | null {
  if (field.kind === "list") return setPath(object, field.path, parseList(raw))
  if (field.kind === "number-list") {
    const values = parseList(raw)
    if (!values) return setPath(object, field.path, undefined)
    const numbers = values.map(Number)
    return numbers.every(Number.isFinite) ? setPath(object, field.path, numbers) : null
  }
  if (field.kind === "number") return setPath(object, field.path, raw === "" ? undefined : Number(raw))
  if (field.path === "udp_fragment") return setPath(object, field.path, raw === "" ? undefined : raw === "true")
  return setPath(object, field.path, raw || undefined)
}

function ProxyField({ field, object, namespace, revision, onChange, onFieldValidityChange, transformField }: Omit<ProxyFormFieldsProps, "fields"> & { field: FieldSpec }) {
  const { t } = useTranslation()
  const value = getPath(object, field.path)
  const label = t(`${namespace}.${field.label}`)
  const update = (raw: string) => { const transformed = transformField?.(object, field, raw); const next = transformed === undefined ? defaultUpdate(object, field, raw) : transformed; if (next) onChange(next) }
  if (field.kind === "boolean") return <BooleanField label={label} checked={value === true} onChange={(checked) => onChange(setPath(object, field.path, checked || undefined))} />
  if (field.kind === "select") return <SelectField field={field} label={label} namespace={namespace} value={textValue(value)} onChange={update} />
  if (field.kind === "users" || field.kind === "json-object") return <JSONField field={field} label={label} namespace={namespace} revision={revision} value={value} array={field.kind === "users"} onChange={(next) => onChange(setPath(object, field.path, next))} onFieldValidityChange={onFieldValidityChange} />
  return <TextField field={field} label={label} value={textValue(value)} onChange={update} />
}

export function ProxyFormFields(props: ProxyFormFieldsProps) {
  const { fields, ...rest } = props
  return <FieldGroup className="grid gap-4 sm:grid-cols-2">{fields.map((field) => <ProxyField key={field.path} field={field} {...rest} />)}</FieldGroup>
}
