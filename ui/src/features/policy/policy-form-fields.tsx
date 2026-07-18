import { useQuery } from "@tanstack/react-query"
import { CircleHelpIcon } from "lucide-react"
import { useCallback, useEffect, useId, useMemo, useRef, useState, type ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { defaultPolicyFieldUpdate } from "@/features/policy/policy-field-update"
import {
  getPolicyPath, groupPolicyFieldsBySection, setPolicyPath, visiblePolicyFields,
  type JsonObject, type PolicyFieldSpec, type PolicyFieldTransform, type PolicyFormContext,
} from "@/features/policy/policy-form-model"
import { TransformedPolicyField } from "@/features/policy/policy-transformed-field"
import { api } from "@/lib/api/endpoints"
import type { JsonValue, NetworkInterfaceInfo } from "@/lib/api/types"

export interface PolicyFormFieldsProps {
  fields: readonly PolicyFieldSpec[]
  object: JsonObject
  namespace: string
  revision?: number
  context?: PolicyFormContext
  onChange: (object: JsonObject) => void
  onFieldValidityChange?: (path: string, valid: boolean) => void
  transformField?: PolicyFieldTransform
  leading?: ReactNode
}

type ValidityCallback = PolicyFormFieldsProps["onFieldValidityChange"]

function textValue(value: JsonValue | undefined): string {
  if (Array.isArray(value)) return value.map(String).join("\n")
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value)
  return ""
}

function listValue(value: JsonValue | undefined) {
  return Array.isArray(value) ? value.map(String) : []
}

function useFieldValidity(path: string, valid: boolean, onChange?: ValidityCallback, resetKey?: string) {
  const callbackRef = useRef(onChange)
  useEffect(() => { callbackRef.current = onChange }, [onChange])
  useEffect(() => { callbackRef.current?.(path, valid) }, [path, resetKey, valid])
  useEffect(() => () => callbackRef.current?.(path, true), [path])
  return useCallback((next: boolean) => callbackRef.current?.(path, next), [path])
}

function FieldHelp({ namespace, labelKey }: { namespace: string; labelKey: string }) {
  const { t, i18n } = useTranslation()
  const helpKey = `${namespace}.${labelKey}Help`
  if (!i18n.exists(helpKey)) return null
  return <Tooltip>
    <TooltipTrigger
      type="button"
      className="inline-flex size-4 items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
      aria-label={t("common.fieldHelp")}
    >
      <CircleHelpIcon className="size-3.5" />
    </TooltipTrigger>
    <TooltipContent className="max-w-xs text-left leading-relaxed">{t(helpKey)}</TooltipContent>
  </Tooltip>
}

function FieldHeading({ id, label, namespace, labelKey }: { id: string; label: string; namespace: string; labelKey: string }) {
  return <div className="flex items-center gap-1.5">
    <FieldLabel htmlFor={id}>{label}</FieldLabel>
    <FieldHelp namespace={namespace} labelKey={labelKey} />
  </div>
}

function BooleanField({ field, label, namespace, checked, onChange }: {
  field: PolicyFieldSpec; label: string; namespace: string; checked: boolean; onChange: (value: boolean) => void
}) {
  const id = useId()
  return <Field orientation="horizontal">
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Switch id={id} aria-label={label} required={field.required} checked={checked} onCheckedChange={onChange} />
  </Field>
}

function SelectField({ field, label, namespace, value, onChange, onFieldValidityChange }: {
  field: PolicyFieldSpec; label: string; namespace: string; value: string
  onChange: (value: string) => void; onFieldValidityChange?: ValidityCallback
}) {
  const { t } = useTranslation()
  const id = useId()
  const optionsKey = JSON.stringify(field.options?.filter(Boolean) ?? [])
  const options = useMemo(() => JSON.parse(optionsKey) as string[], [optionsKey])
  const values = useMemo(() => value && !options.includes(value) ? [value, ...options] : options, [options, value])
  const items = useMemo(
    () => [{ value: null, label: t(`${namespace}.notSet`) }, ...values.map((option) => ({ value: option, label: option }))],
    [namespace, t, values],
  )
  const invalid = Boolean(field.required && !value)
  useFieldValidity(field.path, !invalid, onFieldValidityChange)
  return <Field data-invalid={invalid}>
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Select items={items} value={value || null} required={field.required} onValueChange={(next) => onChange(next === null ? "" : String(next))}>
      <SelectTrigger id={id} aria-label={label} aria-invalid={invalid} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>
        <SelectItem value={null}>{t(`${namespace}.notSet`)}</SelectItem>
        {values.map((option) => <SelectItem key={option} value={option}>{option}</SelectItem>)}
      </SelectGroup></SelectContent>
    </Select>
  </Field>
}

function RefSelectField({ field, label, namespace, value, context, onChange, onFieldValidityChange }: {
  field: PolicyFieldSpec; label: string; namespace: string; value: string
  context?: PolicyFormContext; onChange: (value: string) => void; onFieldValidityChange?: ValidityCallback
}) {
  const { t } = useTranslation()
  const id = useId()
  const options = useMemo(() => {
    if (field.ref === "inbound") return context?.inboundTags ?? []
    if (field.ref === "outbound") return context?.outboundTags ?? []
    if (field.ref === "dns-server") return context?.dnsServerTags ?? []
    if (field.ref === "rule-set") return context?.ruleSetTags ?? []
    return field.options ? [...field.options] : []
  }, [context?.dnsServerTags, context?.inboundTags, context?.outboundTags, context?.ruleSetTags, field.options, field.ref])
  const items = useMemo(() => {
    const list = [...options]
    if (value && !list.includes(value)) list.unshift(value)
    return [{ value: null as string | null, label: t(`${namespace}.notSet`) }, ...list.map((option) => ({ value: option, label: option }))]
  }, [namespace, options, t, value])
  const invalid = Boolean(field.required && !value)
  useFieldValidity(field.path, !invalid, onFieldValidityChange)
  if (options.length === 0) {
    return <Field data-invalid={invalid}>
      <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
      <Input id={id} aria-label={label} aria-invalid={invalid} required={field.required} value={value}
        onChange={(event) => onChange(event.target.value)} />
    </Field>
  }
  return <Field data-invalid={invalid}>
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Select items={items} value={value || null} required={field.required}
      onValueChange={(next) => onChange(next === null ? "" : String(next))}>
      <SelectTrigger id={id} aria-label={label} aria-invalid={invalid} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>
        <SelectItem value={null}>{t(`${namespace}.notSet`)}</SelectItem>
        {items.filter((item) => item.value !== null).map((item) => (
          <SelectItem key={String(item.value)} value={item.value}>{item.label}</SelectItem>
        ))}
      </SelectGroup></SelectContent>
    </Select>
  </Field>
}

function NetworkMultiField({ label, namespace, labelKey, value, onChange }: {
  label: string; namespace: string; labelKey: string; value: string[]
  onChange: (value: string[] | undefined) => void
}) {
  const id = useId()
  const options = ["tcp", "udp"]
  const toggle = (option: string, checked: boolean) => {
    const next = checked ? [...new Set([...value, option])] : value.filter((item) => item !== option)
    onChange(next.length ? next : undefined)
  }
  return <Field className="sm:col-span-2">
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={labelKey} />
    <div className="flex flex-wrap gap-4" role="group" aria-label={label}>
      {options.map((option) => <label key={option} className="flex items-center gap-2 text-sm">
        <Switch aria-label={`${label} ${option}`} checked={value.includes(option)} onCheckedChange={(checked) => toggle(option, checked)} />
        <span>{option}</span>
      </label>)}
    </div>
  </Field>
}

function interfaceLabel(item: NetworkInterfaceInfo) {
  return item.ips?.length ? `${item.name} (${item.ips.join(", ")})` : item.name
}

function RefMultiField({ field, label, namespace, value, context, onChange }: {
  field: PolicyFieldSpec; label: string; namespace: string; value: string[]
  context?: PolicyFormContext; onChange: (value: string[] | undefined) => void
}) {
  const { t } = useTranslation()
  const id = useId()
  const options = useMemo(() => {
    if (field.ref === "inbound") return context?.inboundTags ?? []
    if (field.ref === "outbound") return context?.outboundTags ?? []
    if (field.ref === "dns-server") return context?.dnsServerTags ?? []
    if (field.ref === "rule-set") return context?.ruleSetTags ?? []
    return field.options ? [...field.options] : []
  }, [context?.dnsServerTags, context?.inboundTags, context?.outboundTags, context?.ruleSetTags, field.options, field.ref])
  const items = useMemo(() => {
    const list = [...options]
    for (const item of value) if (item && !list.includes(item)) list.unshift(item)
    return list
  }, [options, value])
  const toggle = (option: string, checked: boolean) => {
    const next = checked ? [...new Set([...value, option])] : value.filter((item) => item !== option)
    onChange(next.length ? next : undefined)
  }
  return <Field className="sm:col-span-2">
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <div className="flex flex-col gap-2" role="group" aria-label={label}>
      {items.length === 0
        ? <p className="text-sm text-muted-foreground">{t(`${namespace}.refMultiEmpty`)}</p>
        : items.map((option) => <div key={option} className="flex items-center gap-2 text-sm">
          <Checkbox
            checked={value.includes(option)}
            onCheckedChange={(next) => toggle(option, next === true)}
            aria-label={`${label} ${option}`}
          />
          <span aria-hidden="true">{option}</span>
        </div>)}
    </div>
  </Field>
}

function NetworkInterfaceField({ label, namespace, labelKey, revision = 0, value, onChange }: {
  label: string; namespace: string; labelKey: string; revision?: number; value: string
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const selectId = useId()
  const inputId = useId()
  const query = useQuery({ queryKey: ["network", "interfaces"], queryFn: api.network.interfaces })
  const interfaces = useMemo(() => query.data?.interfaces ?? [], [query.data?.interfaces])
  const names = useMemo(() => interfaces.map((item) => item.name), [interfaces])
  const sourceKey = String(revision)
  const [source, setSource] = useState(sourceKey)
  const [mode, setMode] = useState(() => value && !names.includes(value) ? "manual" : value || "unset")
  if (source !== sourceKey) {
    setSource(sourceKey)
    setMode(value && !names.includes(value) ? "manual" : value || "unset")
  }
  const options = useMemo(() => {
    const list = [...names]
    if (value && !list.includes(value) && mode !== "manual") list.unshift(value)
    return list
  }, [mode, names, value])
  const items = useMemo(() => [
    { value: "unset", label: t(`${namespace}.notSet`) },
    ...options.map((name) => {
      const meta = interfaces.find((item) => item.name === name)
      return { value: name, label: meta ? interfaceLabel(meta) : name }
    }),
    { value: "manual", label: t(`${namespace}.interfaceManual`) },
  ], [interfaces, namespace, options, t])
  return <Field className="sm:col-span-2">
    <FieldHeading id={selectId} label={label} namespace={namespace} labelKey={labelKey} />
    <Select items={items} value={mode === "manual" ? "manual" : value || "unset"} onValueChange={(next) => {
      const selected = String(next)
      if (selected === "unset") { setMode("unset"); onChange(""); return }
      if (selected === "manual") { setMode("manual"); return }
      setMode(selected)
      onChange(selected)
    }}>
      <SelectTrigger id={selectId} aria-label={label} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>
        {items.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}
      </SelectGroup></SelectContent>
    </Select>
    {mode === "manual" ? <>
      <FieldLabel htmlFor={inputId}>{t(`${namespace}.interfaceManualInput`)}</FieldLabel>
      <Input id={inputId} aria-label={t(`${namespace}.interfaceManualInput`)} value={value} placeholder={t(`${namespace}.interfaceManualPlaceholder`)} onChange={(event) => onChange(event.target.value)} />
    </> : null}
  </Field>
}

function TextField({ field, label, namespace, value, onChange }: {
  field: PolicyFieldSpec; label: string; namespace: string; value: string; onChange: (value: string) => void
}) {
  const id = useId()
  const area = field.kind === "textarea" || field.kind === "list" || field.kind === "number-list"
  return <Field>
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    {area
      ? <Textarea id={id} aria-label={label} required={field.required} value={value} onChange={(event) => onChange(event.target.value)} />
      : <Input id={id} aria-label={label} required={field.required} type={field.kind === "number" ? "number" : "text"} value={value} onChange={(event) => onChange(event.target.value)} />}
  </Field>
}

function useStructuredDraft(props: {
  field: PolicyFieldSpec; revision?: number; value: JsonValue | undefined; array: boolean
  onChange: (value: JsonValue | undefined) => void; onFieldValidityChange?: ValidityCallback
}) {
  const { field, revision = 0, value, array, onChange, onFieldValidityChange } = props
  const serialized = value === undefined ? "" : JSON.stringify(value, null, 2)
  const sourceKey = `${revision}:${serialized}`
  const [raw, setRaw] = useState(serialized)
  const [source, setSource] = useState(sourceKey)
  const [invalid, setInvalid] = useState(false)
  if (source !== sourceKey) {
    setSource(sourceKey)
    setRaw(serialized)
    setInvalid(false)
  }
  const reportValidity = useFieldValidity(field.path, !invalid, onFieldValidityChange, sourceKey)
  const update = (next: string) => {
    setRaw(next)
    if (!next.trim()) {
      setSource(`${revision}:`)
      setInvalid(false)
      reportValidity(true)
      onChange(undefined)
      return
    }
    try {
      const parsed: unknown = JSON.parse(next)
      const valid = array ? Array.isArray(parsed) : Boolean(parsed && typeof parsed === "object" && !Array.isArray(parsed))
      setInvalid(!valid)
      reportValidity(valid)
      if (valid) {
        setSource(`${revision}:${JSON.stringify(parsed, null, 2)}`)
        onChange(parsed as JsonValue)
      }
    } catch (error) {
      void error
      setInvalid(true)
      reportValidity(false)
    }
  }
  return { invalid, raw, update }
}

function StructuredField(props: {
  field: PolicyFieldSpec; label: string; namespace: string; revision?: number
  value: JsonValue | undefined; array: boolean
  onChange: (value: JsonValue | undefined) => void; onFieldValidityChange?: ValidityCallback
}) {
  const { field, label, namespace, array } = props
  const { t } = useTranslation()
  const id = useId()
  const { invalid, raw, update } = useStructuredDraft(props)
  const hint = invalid ? "invalidStructuredJSON" : array ? "jsonArrayHint" : "jsonObjectHint"
  return <Field data-invalid={invalid} className="sm:col-span-2">
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Textarea
      id={id}
      aria-label={label}
      aria-invalid={invalid}
      required={field.required}
      value={raw}
      onChange={(event) => update(event.target.value)}
    />
    <FieldDescription>{t(`${namespace}.${hint}`)}</FieldDescription>
  </Field>
}

function PolicyField(props: Omit<PolicyFormFieldsProps, "fields" | "leading"> & { field: PolicyFieldSpec }) {
  const { field, object, namespace, revision, context, onChange, onFieldValidityChange, transformField } = props
  const { t } = useTranslation()
  const value = getPolicyPath(object, field.path)
  const label = t(`${namespace}.${field.label}`)
  const update = (raw: string) => {
    const transformed = transformField?.(object, field, raw)
    const next = transformed === undefined ? defaultPolicyFieldUpdate(object, field, raw) : transformed
    if (next) onChange(next)
  }
  if (field.kind === "boolean") {
    return <BooleanField field={field} label={label} namespace={namespace} checked={value === true}
      onChange={(checked) => onChange(setPolicyPath(object, field.path, checked))} />
  }
  if (field.kind === "select") {
    return <SelectField field={field} label={label} namespace={namespace} value={textValue(value)}
      onChange={(raw) => {
        if (transformField) {
          const transformed = transformField(object, field, raw)
          if (transformed === null) return
          if (transformed !== undefined) { onChange(transformed); return }
        }
        const options = field.options ?? []
        const numeric = options.length > 0 && options.every((option) => option !== "" && Number.isFinite(Number(option)))
        if (numeric) {
          onChange(setPolicyPath(object, field.path, raw === "" ? undefined : Number(raw)))
          return
        }
        update(raw)
      }} onFieldValidityChange={onFieldValidityChange} />
  }
  if (field.kind === "ref" && field.ref === "network-interface") {
    return <NetworkInterfaceField label={label} namespace={namespace} labelKey={field.label} revision={revision}
      value={textValue(value)} onChange={update} />
  }
  if (field.kind === "ref") {
    return <RefSelectField field={field} label={label} namespace={namespace} value={textValue(value)} context={context}
      onChange={update} onFieldValidityChange={onFieldValidityChange} />
  }
  if (field.kind === "ref-multi") {
    return <RefMultiField field={field} label={label} namespace={namespace} value={listValue(value)} context={context}
      onChange={(next) => onChange(setPolicyPath(object, field.path, next))} />
  }
  if (field.kind === "network-multi") {
    return <NetworkMultiField label={label} namespace={namespace} labelKey={field.label} value={listValue(value)}
      onChange={(next) => onChange(setPolicyPath(object, field.path, next))} />
  }
  if (field.kind === "network-interface") {
    return <NetworkInterfaceField label={label} namespace={namespace} labelKey={field.label} revision={revision}
      value={textValue(value)} onChange={update} />
  }
  if (field.kind === "json-object" || field.kind === "json-array") {
    return <StructuredField field={field} label={label} namespace={namespace} revision={revision}
      value={value} array={field.kind === "json-array"}
      onChange={(next) => onChange(setPolicyPath(object, field.path, next))}
      onFieldValidityChange={onFieldValidityChange} />
  }
  if (transformField) {
    return <TransformedPolicyField {...props} label={label} value={value} transformField={transformField} />
  }
  return <TextField field={field} label={label} namespace={namespace} value={textValue(value)} onChange={update} />
}

export function PolicyFormFields({ fields, leading, object, ...rest }: PolicyFormFieldsProps) {
  const { t, i18n } = useTranslation()
  const shown = visiblePolicyFields(fields, object)
  const groups = groupPolicyFieldsBySection(shown)
  const hasSections = groups.some((group) => group.section)
  if (!hasSections) {
    return <FieldGroup className="grid gap-4 sm:grid-cols-2">
      {leading}
      {shown.map((field) => <PolicyField key={field.path} field={field} object={object} {...rest} />)}
    </FieldGroup>
  }
  return <div className="flex flex-col gap-6">
    {leading ? <FieldGroup className="grid gap-4 sm:grid-cols-2">{leading}</FieldGroup> : null}
    {groups.map((group, index) => {
      const hasTitle = Boolean(group.section && i18n.exists(`${rest.namespace}.section.${group.section}`))
      return <section key={`${group.section ?? "default"}-${index}`} className="flex flex-col gap-3">
        {hasTitle ? <h3 className="text-sm font-medium text-muted-foreground">{t(`${rest.namespace}.section.${group.section}`)}</h3> : null}
        <FieldGroup className="grid gap-4 sm:grid-cols-2">
          {group.fields.map((field) => <PolicyField key={field.path} field={field} object={object} {...rest} />)}
        </FieldGroup>
      </section>
    })}
  </div>
}
