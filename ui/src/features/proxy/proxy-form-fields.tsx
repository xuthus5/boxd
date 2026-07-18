import { useQuery } from "@tanstack/react-query"
import { CircleHelpIcon } from "lucide-react"
import { useCallback, useEffect, useId, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { InboundUsersField } from "@/features/proxy/inbound-users-field"
import {
  getPath, groupFieldsBySection, type FieldSpec, type FieldTransform, type FormFieldContext, type JsonObject, setPath, visibleFields,
} from "@/features/proxy/proxy-form-model"
import { api } from "@/lib/api/endpoints"
import type { JsonValue, NetworkInterfaceInfo } from "@/lib/api/types"

interface ProxyFormFieldsProps {
  fields: FieldSpec[]
  object: JsonObject
  namespace: "proxy.inbound" | "proxy.outbound"
  revision?: number
  context?: FormFieldContext
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

function FieldHelp({ namespace, labelKey }: { namespace: ProxyFormFieldsProps["namespace"]; labelKey: string }) {
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

function FieldHeading({ id, label, namespace, labelKey }: { id: string; label: string; namespace: ProxyFormFieldsProps["namespace"]; labelKey: string }) {
  return <div className="flex items-center gap-1.5">
    <FieldLabel htmlFor={id}>{label}</FieldLabel>
    <FieldHelp namespace={namespace} labelKey={labelKey} />
  </div>
}

function BooleanField({ label, namespace, labelKey, checked, onChange }: { label: string; namespace: ProxyFormFieldsProps["namespace"]; labelKey: string; checked: boolean; onChange: (value: boolean) => void }) {
  const id = useId()
  return <Field orientation="horizontal"><FieldHeading id={id} label={label} namespace={namespace} labelKey={labelKey} /><Switch id={id} aria-label={label} checked={checked} onCheckedChange={onChange} /></Field>
}

function SelectField({ field, label, namespace, value, onChange }: { field: FieldSpec; label: string; namespace: ProxyFormFieldsProps["namespace"]; value: string; onChange: (value: string) => void }) {
  const { t } = useTranslation()
  const id = useId()
  const options = useMemo(() => field.options?.filter(Boolean) ?? [], [field.options])
  const unset = "__unset__"
  const optionLabel = useCallback((option: string) => option === "true" ? t(`${namespace}.enabled`) : option === "false" ? t(`${namespace}.disabled`) : option, [namespace, t])
  const items = useMemo(() => [{ value: unset, label: t(`${namespace}.notSet`) }, ...options.map((option) => ({ value: option, label: optionLabel(option) }))], [namespace, optionLabel, options, t])
  return <Field>
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Select items={items} value={value || unset} onValueChange={(next) => onChange(next === unset ? "" : String(next))}>
      <SelectTrigger id={id} aria-label={label} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>
        <SelectItem value={unset}>{t(`${namespace}.notSet`)}</SelectItem>
        {options.map((option) => <SelectItem key={option} value={option}>{optionLabel(option)}</SelectItem>)}
      </SelectGroup></SelectContent>
    </Select>
  </Field>
}

function ListenAddressField({ label, namespace, labelKey, revision = 0, value, onChange }: { label: string; namespace: ProxyFormFieldsProps["namespace"]; labelKey: string; revision?: number; value: string; onChange: (value: string) => void }) {
  const { t } = useTranslation()
  const selectId = useId()
  const inputId = useId()
  const isPreset = (candidate: string) => candidate === "0.0.0.0" || candidate === "::"
  const deriveMode = (candidate: string) => isPreset(candidate) ? candidate : candidate ? "manual" : "unset"
  const sourceKey = String(revision)
  const [source, setSource] = useState(sourceKey)
  const [mode, setMode] = useState(() => deriveMode(value))
  if (source !== sourceKey) {
    setSource(sourceKey)
    setMode(deriveMode(value))
  }
  const items = useMemo(() => [
    { value: "unset", label: t(`${namespace}.notSet`) },
    { value: "0.0.0.0", label: t(`${namespace}.listenIPv4All`) },
    { value: "::", label: t(`${namespace}.listenIPv6All`) },
    { value: "manual", label: t(`${namespace}.listenManual`) },
  ], [namespace, t])
  return <Field className="sm:col-span-2">
    <FieldHeading id={selectId} label={label} namespace={namespace} labelKey={labelKey} />
    <div className="grid gap-2">
      <Select items={items} value={mode} onValueChange={(next) => {
        const selected = String(next)
        if (selected === "unset") {
          setMode("unset")
          onChange("")
          return
        }
        if (selected === "manual") {
          setMode("manual")
          if (isPreset(value)) onChange("")
          return
        }
        setMode(selected)
        onChange(selected)
      }}>
        <SelectTrigger id={selectId} aria-label={label} className="w-full"><SelectValue /></SelectTrigger>
        <SelectContent><SelectGroup>
          <SelectItem value="unset">{t(`${namespace}.notSet`)}</SelectItem>
          <SelectItem value="0.0.0.0">{t(`${namespace}.listenIPv4All`)}</SelectItem>
          <SelectItem value="::">{t(`${namespace}.listenIPv6All`)}</SelectItem>
          <SelectItem value="manual">{t(`${namespace}.listenManual`)}</SelectItem>
        </SelectGroup></SelectContent>
      </Select>
      {mode === "manual" ? <Input id={inputId} aria-label={t(`${namespace}.listenManualInput`)} value={isPreset(value) ? "" : value} placeholder={t(`${namespace}.listenManualPlaceholder`)} onChange={(event) => onChange(event.target.value)} /> : null}
    </div>
  </Field>
}

function interfaceLabel(item: NetworkInterfaceInfo) {
  const ips = item.ips?.filter(Boolean) ?? []
  return ips.length ? `${item.name} (${ips.join(", ")})` : item.name
}

function NetworkInterfaceField({ label, namespace, labelKey, revision = 0, value, onChange }: { label: string; namespace: ProxyFormFieldsProps["namespace"]; labelKey: string; revision?: number; value: string; onChange: (value: string) => void }) {
  const { t } = useTranslation()
  const selectId = useId()
  const inputId = useId()
  const query = useQuery({ queryKey: ["network", "interfaces"], queryFn: api.network.interfaces })
  const interfaces = useMemo(() => query.data?.interfaces ?? [], [query.data?.interfaces])
  const known = interfaces.some((item) => item.name === value)
  const deriveMode = useCallback((candidate: string) => !candidate ? "unset" : interfaces.some((item) => item.name === candidate) ? candidate : "manual", [interfaces])
  const sourceKey = `${revision}:${interfaces.map((item) => item.name).join("|")}`
  const [source, setSource] = useState(sourceKey)
  const [mode, setMode] = useState(() => deriveMode(value))
  if (source !== sourceKey) {
    setSource(sourceKey)
    setMode(deriveMode(value))
  }
  const items = useMemo(() => [
    { value: "unset", label: t(`${namespace}.notSet`) },
    ...interfaces.map((item) => ({ value: item.name, label: interfaceLabel(item) })),
    { value: "manual", label: t(`${namespace}.interfaceManual`) },
  ], [interfaces, namespace, t])
  return <Field className="sm:col-span-2">
    <FieldHeading id={selectId} label={label} namespace={namespace} labelKey={labelKey} />
    <div className="grid gap-2">
      <Select items={items} value={mode} onValueChange={(next) => {
        const selected = String(next)
        if (selected === "unset") {
          setMode("unset")
          onChange("")
          return
        }
        if (selected === "manual") {
          setMode("manual")
          if (known) onChange("")
          return
        }
        setMode(selected)
        onChange(selected)
      }}>
        <SelectTrigger id={selectId} aria-label={label} className="w-full"><SelectValue /></SelectTrigger>
        <SelectContent><SelectGroup>
          <SelectItem value="unset">{t(`${namespace}.notSet`)}</SelectItem>
          {interfaces.map((item) => <SelectItem key={item.name} value={item.name}>{interfaceLabel(item)}</SelectItem>)}
          <SelectItem value="manual">{t(`${namespace}.interfaceManual`)}</SelectItem>
        </SelectGroup></SelectContent>
      </Select>
      {mode === "manual" ? <Input id={inputId} aria-label={t(`${namespace}.interfaceManualInput`)} value={known ? "" : value} placeholder={t(`${namespace}.interfaceManualPlaceholder`)} onChange={(event) => onChange(event.target.value)} /> : null}
    </div>
  </Field>
}

function TextField({ field, label, namespace, value, onChange }: { field: FieldSpec; label: string; namespace: ProxyFormFieldsProps["namespace"]; value: string; onChange: (value: string) => void }) {
  const id = useId()
  const area = field.kind === "textarea" || field.kind === "list" || field.kind === "number-list"
  const control = area
    ? <Textarea id={id} aria-label={label} value={value} onChange={(event) => onChange(event.target.value)} />
    : <Input id={id} aria-label={label} type={field.kind === "number" ? "number" : "text"} value={value} onChange={(event) => onChange(event.target.value)} />
  return <Field><FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />{control}</Field>
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
      setInvalid(!valid)
      onFieldValidityChange?.(field.path, valid)
      if (valid) { setSource(`${revision}:${JSON.stringify(parsed, null, 2)}`); onChange(parsed as JsonValue) }
    } catch {
      setInvalid(true)
      onFieldValidityChange?.(field.path, false)
    }
  }
  return <Field data-invalid={invalid} className="sm:col-span-2">
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Textarea id={id} aria-label={label} aria-invalid={invalid} value={raw} onChange={(event) => update(event.target.value)} />
    <FieldDescription>{invalid ? t(`${namespace}.invalidStructuredJSON`) : t(array ? `${namespace}.usersJSONHint` : `${namespace}.jsonObjectHint`)}</FieldDescription>
  </Field>
}

function defaultUpdate(object: JsonObject, field: FieldSpec, raw: string) {
  if (field.kind === "list") return setPath(object, field.path, parseList(raw))
  if (field.kind === "number-list") {
    const values = parseList(raw)
    if (!values) return setPath(object, field.path, undefined)
    const numbers = values.map(Number)
    return numbers.every(Number.isFinite) ? setPath(object, field.path, numbers) : null
  }
  if (field.kind === "number") return setPath(object, field.path, raw === "" ? undefined : Number(raw))
  return setPath(object, field.path, raw || undefined)
}

function listValue(value: JsonValue | undefined) {
  return Array.isArray(value) ? value.map(String) : []
}

function RefSelectField({ field, label, namespace, value, context, onChange }: { field: FieldSpec; label: string; namespace: ProxyFormFieldsProps["namespace"]; value: string; context?: FormFieldContext; onChange: (value: string) => void }) {
  const { t } = useTranslation()
  const id = useId()
  const unset = "__unset__"
  const options = useMemo(() => {
    if (field.ref === "inbound") return context?.inboundTags ?? []
    if (field.ref === "outbound") return context?.outboundTags ?? []
    if (field.ref === "dns-server") return context?.dnsServerTags ?? []
    return field.options ?? []
  }, [context?.dnsServerTags, context?.inboundTags, context?.outboundTags, field.options, field.ref])
  const items = useMemo(() => {
    const list = [...options]
    if (value && !list.includes(value)) list.unshift(value)
    return [{ value: unset, label: t(`${namespace}.notSet`) }, ...list.map((option) => ({ value: option, label: option }))]
  }, [namespace, options, t, value])
  return <Field>
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Select items={items} value={value || unset} onValueChange={(next) => onChange(next === unset ? "" : String(next))}>
      <SelectTrigger id={id} aria-label={label} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>
        <SelectItem value={unset}>{t(`${namespace}.notSet`)}</SelectItem>
        {items.filter((item) => item.value !== unset).map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}
      </SelectGroup></SelectContent>
    </Select>
  </Field>
}

function NetworkMultiField({ label, namespace, labelKey, value, onChange }: { label: string; namespace: ProxyFormFieldsProps["namespace"]; labelKey: string; value: string[]; onChange: (value: string[] | undefined) => void }) {
  const id = useId()
  const options = ["tcp", "udp"]
  const toggle = (option: string, checked: boolean) => {
    const next = checked ? [...new Set([...value, option])] : value.filter((item) => item !== option)
    onChange(next.length ? next : undefined)
  }
  return <Field className="sm:col-span-2">
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={labelKey} />
    <div className="flex flex-wrap gap-6" role="group" aria-label={label}>
      {options.map((option) => <Field key={option} orientation="horizontal" className="w-auto">
        <FieldLabel>{option}</FieldLabel>
        <Switch aria-label={`${label} ${option}`} checked={value.includes(option)} onCheckedChange={(checked) => toggle(option, checked)} />
      </Field>)}
    </div>
  </Field>
}

function InterfaceMultiField({ label, namespace, labelKey, value, onChange }: { label: string; namespace: ProxyFormFieldsProps["namespace"]; labelKey: string; value: string[]; onChange: (value: string[] | undefined) => void }) {
  const id = useId()
  const query = useQuery({ queryKey: ["network", "interfaces"], queryFn: api.network.interfaces })
  const interfaces = useMemo(() => query.data?.interfaces ?? [], [query.data?.interfaces])
  const options = useMemo(() => [...new Set([...interfaces.map((item) => item.name), ...value])], [interfaces, value])
  const toggle = (option: string, checked: boolean) => {
    const next = checked ? [...new Set([...value, option])] : value.filter((item) => item !== option)
    onChange(next.length ? next : undefined)
  }
  return <Field className="sm:col-span-2">
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={labelKey} />
    <div className="flex flex-col gap-2" role="group" aria-label={label}>
      {options.length === 0 ? <p className="text-sm text-muted-foreground">—</p> : options.map((option) => {
        const meta = interfaces.find((item) => item.name === option)
        return <label key={option} className="flex items-center gap-2 text-sm">
          <Checkbox checked={value.includes(option)} onCheckedChange={(next) => toggle(option, next === true)} aria-label={`${label} ${option}`} />
          <span>{meta ? interfaceLabel(meta) : option}</span>
        </label>
      })}
    </div>
  </Field>
}

function ProxyField({ field, object, namespace, revision, context, onChange, onFieldValidityChange, transformField }: Omit<ProxyFormFieldsProps, "fields"> & { field: FieldSpec }) {
  const { t } = useTranslation()
  const value = getPath(object, field.path)
  const label = t(`${namespace}.${field.label}`)
  const update = (raw: string) => {
    const transformed = transformField?.(object, field, raw)
    const next = transformed === undefined ? defaultUpdate(object, field, raw) : transformed
    if (next) onChange(next)
  }
  if (field.kind === "boolean") return <BooleanField label={label} namespace={namespace} labelKey={field.label} checked={value === true} onChange={(checked) => onChange(setPath(object, field.path, checked || undefined))} />
  if (field.kind === "select") return <SelectField field={field} label={label} namespace={namespace} value={textValue(value)} onChange={update} />
  if (field.kind === "listen-address") return <ListenAddressField label={label} namespace={namespace} labelKey={field.label} revision={revision} value={textValue(value)} onChange={update} />
  if (field.kind === "network-interface") return <NetworkInterfaceField label={label} namespace={namespace} labelKey={field.label} revision={revision} value={textValue(value)} onChange={update} />
  if (field.kind === "ref" && field.ref === "network-interface-multi") return <InterfaceMultiField label={label} namespace={namespace} labelKey={field.label} value={listValue(value)} onChange={(next) => onChange(setPath(object, field.path, next))} />
  if (field.kind === "ref") return <RefSelectField field={field} label={label} namespace={namespace} value={textValue(value)} context={context} onChange={update} />
  if (field.kind === "network-multi") return <NetworkMultiField label={label} namespace={namespace} labelKey={field.label} value={listValue(value)} onChange={(next) => onChange(setPath(object, field.path, next))} />
  if (field.kind === "users" && namespace === "proxy.inbound") {
    return <InboundUsersField label={label} type={context?.inboundType ?? ""} value={value} onChange={(next) => onChange(setPath(object, field.path, next))} onFieldValidityChange={onFieldValidityChange} path={field.path} />
  }
  if (field.kind === "users" || field.kind === "json-object") return <JSONField field={field} label={label} namespace={namespace} revision={revision} value={value} array={field.kind === "users"} onChange={(next) => onChange(setPath(object, field.path, next))} onFieldValidityChange={onFieldValidityChange} />
  return <TextField field={field} label={label} namespace={namespace} value={textValue(value)} onChange={update} />
}

export function ProxyFormFields(props: ProxyFormFieldsProps) {
  const { fields, object, ...rest } = props
  const { t, i18n } = useTranslation()
  const shown = visibleFields(fields, object)
  const groups = groupFieldsBySection(shown)
  return <div className="flex flex-col gap-6">
    {groups.map((group, index) => {
      const hasTitle = Boolean(group.section && i18n.exists(`${props.namespace}.section.${group.section}`))
      return <section key={`${group.section ?? "default"}-${index}`} className="flex flex-col gap-3">
        {hasTitle ? <h3 className="text-sm font-medium text-muted-foreground">{t(`${props.namespace}.section.${group.section}`)}</h3> : null}
        <FieldGroup className="grid gap-4 sm:grid-cols-2">
          {group.fields.map((field) => <ProxyField key={field.path} field={field} object={object} {...rest} />)}
        </FieldGroup>
      </section>
    })}
  </div>
}
