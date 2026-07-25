import { useQuery } from "@tanstack/react-query"
import { CircleHelpIcon } from "lucide-react"
import { useCallback, useId, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { InboundUsersField } from "@/features/proxy/inbound-users-field"
import {
  parseFiniteNumber,
  parseFiniteNumberList,
} from "@/features/proxy/proxy-field-validation"
import { ProxyJSONField } from "@/features/proxy/proxy-json-field"
import { InterfaceMultiField, NetworkMultiField } from "@/features/proxy/proxy-multi-fields"
import { ProxyTextField } from "@/features/proxy/proxy-text-field"
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
  const options = useMemo(() => {
    const base = field.options?.filter(Boolean) ?? []
    return value && !base.includes(value) ? [value, ...base] : base
  }, [field.options, value])
  const unset = "__unset__"
  const optionLabel = useCallback((option: string) => option === "true" ? t(`${namespace}.enabled`) : option === "false" ? t(`${namespace}.disabled`) : option, [namespace, t])
  const items = useMemo(() => [{ value: unset, label: t(`${namespace}.notSet`) }, ...options.map((option) => ({ value: option, label: optionLabel(option) }))], [namespace, optionLabel, options, t])
  return <Field>
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    <Select items={items} value={value || unset} onValueChange={(next) => onChange(next === unset ? "" : String(next))}>
      <SelectTrigger id={id} aria-label={label} className="h-8 w-full"><SelectValue /></SelectTrigger>
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
        <SelectTrigger id={selectId} aria-label={label} className="h-8 w-full"><SelectValue /></SelectTrigger>
        <SelectContent><SelectGroup>
          <SelectItem value="unset">{t(`${namespace}.notSet`)}</SelectItem>
          <SelectItem value="0.0.0.0">{t(`${namespace}.listenIPv4All`)}</SelectItem>
          <SelectItem value="::">{t(`${namespace}.listenIPv6All`)}</SelectItem>
          <SelectItem value="manual">{t(`${namespace}.listenManual`)}</SelectItem>
        </SelectGroup></SelectContent>
      </Select>
      {mode === "manual" ? <Input id={inputId} className="h-8" aria-label={t(`${namespace}.listenManualInput`)} value={isPreset(value) ? "" : value} placeholder={t(`${namespace}.listenManualPlaceholder`)} onChange={(event) => onChange(event.target.value)} /> : null}
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
        <SelectTrigger id={selectId} aria-label={label} className="h-8 w-full"><SelectValue /></SelectTrigger>
        <SelectContent><SelectGroup>
          <SelectItem value="unset">{t(`${namespace}.notSet`)}</SelectItem>
          {interfaces.map((item) => <SelectItem key={item.name} value={item.name}>{interfaceLabel(item)}</SelectItem>)}
          <SelectItem value="manual">{t(`${namespace}.interfaceManual`)}</SelectItem>
        </SelectGroup></SelectContent>
      </Select>
      {mode === "manual" ? <Input id={inputId} className="h-8" aria-label={t(`${namespace}.interfaceManualInput`)} value={known ? "" : value} placeholder={t(`${namespace}.interfaceManualPlaceholder`)} onChange={(event) => onChange(event.target.value)} /> : null}
    </div>
  </Field>
}

function defaultUpdate(object: JsonObject, field: FieldSpec, raw: string) {
  if (field.kind === "list") return setPath(object, field.path, parseList(raw))
  if (field.kind === "number-list") {
    const numbers = parseFiniteNumberList(raw)
    if (numbers === null) return null
    return setPath(object, field.path, numbers)
  }
  if (field.kind === "number") {
    const value = parseFiniteNumber(raw)
    if (value === null) return null
    return setPath(object, field.path, value)
  }
  if (field.kind === "boolean-select") {
    if (raw === "") return setPath(object, field.path, undefined)
    if (raw === "true") return setPath(object, field.path, true)
    if (raw === "false") return setPath(object, field.path, false)
    return null
  }
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
      <SelectTrigger id={id} aria-label={label} className="h-8 w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>
        <SelectItem value={unset}>{t(`${namespace}.notSet`)}</SelectItem>
        {items.filter((item) => item.value !== unset).map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}
      </SelectGroup></SelectContent>
    </Select>
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
  if (field.kind === "select" || field.kind === "boolean-select") return <SelectField field={field} label={label} namespace={namespace} value={textValue(value)} onChange={update} />
  if (field.kind === "listen-address") return <ListenAddressField label={label} namespace={namespace} labelKey={field.label} revision={revision} value={textValue(value)} onChange={update} />
  if (field.kind === "network-interface") return <NetworkInterfaceField label={label} namespace={namespace} labelKey={field.label} revision={revision} value={textValue(value)} onChange={update} />
  if (field.kind === "ref" && field.ref === "network-interface-multi") return <InterfaceMultiField label={label} namespace={namespace} labelKey={field.label} value={listValue(value)} onChange={(next) => onChange(setPath(object, field.path, next))} />
  if (field.kind === "ref") return <RefSelectField field={field} label={label} namespace={namespace} value={textValue(value)} context={context} onChange={update} />
  if (field.kind === "network-multi") return <NetworkMultiField label={label} namespace={namespace} labelKey={field.label} value={listValue(value)} onChange={(next) => onChange(setPath(object, field.path, next))} />
  if (field.kind === "users" && namespace === "proxy.inbound") {
    return <InboundUsersField label={label} type={context?.inboundType ?? ""} value={value} onChange={(next) => onChange(setPath(object, field.path, next))} onFieldValidityChange={onFieldValidityChange} path={field.path} />
  }
  if (field.kind === "users" || field.kind === "json-object") return <ProxyJSONField field={field} label={label} namespace={namespace} revision={revision} value={value} array={field.kind === "users"} onChange={(next) => onChange(setPath(object, field.path, next))} onFieldValidityChange={onFieldValidityChange} />
  return <ProxyTextField field={field} label={label} namespace={namespace} value={textValue(value)} onChange={update} onFieldValidityChange={onFieldValidityChange} />
}

export function ProxyFormFields(props: ProxyFormFieldsProps) {
  const { fields, object, ...rest } = props
  const { t, i18n } = useTranslation()
  const shown = visibleFields(fields, object)
  const groups = groupFieldsBySection(shown)
  return <div className="flex flex-col gap-3 sm:gap-4">
    {groups.map((group, index) => {
      const hasTitle = Boolean(group.section && i18n.exists(`${props.namespace}.section.${group.section}`))
      return <section key={`${group.section ?? "default"}-${index}`} className="flex flex-col gap-2 sm:gap-3">
        {hasTitle ? <h3 className="text-xs font-medium text-muted-foreground sm:text-sm">{t(`${props.namespace}.section.${group.section}`)}</h3> : null}
        <FieldGroup className="grid gap-2 sm:grid-cols-2 sm:gap-3">
          {group.fields.map((field) => <ProxyField key={field.path} field={field} object={object} {...rest} />)}
        </FieldGroup>
      </section>
    })}
  </div>
}
