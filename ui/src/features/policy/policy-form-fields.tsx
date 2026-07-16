import { useCallback, useEffect, useId, useMemo, useRef, useState, type ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { defaultPolicyFieldUpdate } from "@/features/policy/policy-field-update"
import {
  getPolicyPath,
  isJsonObject,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
  type PolicyFieldTransform,
} from "@/features/policy/policy-form-model"
import { TransformedPolicyField } from "@/features/policy/policy-transformed-field"
import type { JsonValue } from "@/lib/api/types"

export interface PolicyFormFieldsProps {
  fields: readonly PolicyFieldSpec[]
  object: JsonObject
  namespace: string
  revision?: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange?: (path: string, valid: boolean) => void
  transformField?: PolicyFieldTransform
  leading?: ReactNode
}

type ValidityCallback = PolicyFormFieldsProps["onFieldValidityChange"]

interface BooleanFieldProps {
  field: PolicyFieldSpec
  label: string
  checked: boolean
  onChange: (value: boolean) => void
}

interface SelectFieldProps {
  field: PolicyFieldSpec
  label: string
  namespace: string
  value: string
  onChange: (value: string) => void
  onFieldValidityChange?: ValidityCallback
}

interface TextFieldProps {
  field: PolicyFieldSpec
  label: string
  value: string
  onChange: (value: string) => void
}

interface StructuredFieldProps {
  field: PolicyFieldSpec
  label: string
  namespace: string
  revision?: number
  value: JsonValue | undefined
  array: boolean
  onChange: (value: JsonValue | undefined) => void
  onFieldValidityChange?: ValidityCallback
}

function textValue(value: JsonValue | undefined): string {
  if (Array.isArray(value)) return value.map(String).join("\n")
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value)
  return ""
}

function useFieldValidity(path: string, valid: boolean, onChange?: ValidityCallback, resetKey?: string) {
  const callbackRef = useRef(onChange)
  useEffect(() => {
    callbackRef.current = onChange
  }, [onChange])
  useEffect(() => {
    callbackRef.current?.(path, valid)
  }, [path, resetKey, valid])
  useEffect(() => () => callbackRef.current?.(path, true), [path])
  return useCallback((next: boolean) => callbackRef.current?.(path, next), [path])
}

function BooleanField({ field, label, checked, onChange }: BooleanFieldProps) {
  const id = useId()
  return (
    <Field orientation="horizontal">
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Switch
        id={id}
        aria-label={label}
        required={field.required}
        checked={checked}
        onCheckedChange={onChange}
      />
    </Field>
  )
}

function SelectField({ field, label, namespace, value, onChange, onFieldValidityChange }: SelectFieldProps) {
  const { t } = useTranslation()
  const id = useId()
  const optionsKey = JSON.stringify(field.options?.filter(Boolean) ?? [])
  const options = useMemo(() => JSON.parse(optionsKey) as string[], [optionsKey])
  const values = useMemo(
    () => value && !options.includes(value) ? [value, ...options] : options,
    [options, value],
  )
  const items = useMemo(
    () => [
      { value: null, label: t(`${namespace}.notSet`) },
      ...values.map((option) => ({ value: option, label: option })),
    ],
    [namespace, t, values],
  )
  const invalid = Boolean(field.required && !value)
  useFieldValidity(field.path, !invalid, onFieldValidityChange)
  const update = useCallback(
    (next: string | null) => onChange(next === null ? "" : String(next)),
    [onChange],
  )
  return (
    <Field data-invalid={invalid}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Select items={items} value={value || null} required={field.required} onValueChange={update}>
        <SelectTrigger id={id} aria-label={label} aria-invalid={invalid} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value={null}>{t(`${namespace}.notSet`)}</SelectItem>
            {values.map((option) => <SelectItem key={option} value={option}>{option}</SelectItem>)}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  )
}

function TextField({ field, label, value, onChange }: TextFieldProps) {
  const id = useId()
  const area = field.kind === "textarea" || field.kind === "list" || field.kind === "number-list"
  const control = area ? (
    <Textarea
      id={id}
      aria-label={label}
      required={field.required}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  ) : (
    <Input
      id={id}
      aria-label={label}
      required={field.required}
      type={field.kind === "number" ? "number" : "text"}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  )
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      {control}
    </Field>
  )
}

function useStructuredDraft(props: StructuredFieldProps) {
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
      const valid = array ? Array.isArray(parsed) : isJsonObject(parsed as JsonValue)
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

function StructuredField(props: StructuredFieldProps) {
  const { field, label, namespace, array } = props
  const { t } = useTranslation()
  const id = useId()
  const { invalid, raw, update } = useStructuredDraft(props)
  const hint = invalid ? "invalidStructuredJSON" : array ? "jsonArrayHint" : "jsonObjectHint"
  return (
    <Field data-invalid={invalid}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
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
  )
}

function PolicyField(props: Omit<PolicyFormFieldsProps, "fields"> & { field: PolicyFieldSpec }) {
  const { field, object, namespace, revision, onChange, onFieldValidityChange, transformField } = props
  const { t } = useTranslation()
  const value = getPolicyPath(object, field.path)
  const label = t(`${namespace}.${field.label}`)
  const update = (raw: string) => {
    const transformed = transformField?.(object, field, raw)
    const next = transformed === undefined ? defaultPolicyFieldUpdate(object, field, raw) : transformed
    if (next) onChange(next)
  }
  if (field.kind === "boolean") {
    return <BooleanField field={field} label={label} checked={value === true}
      onChange={(checked) => onChange(setPolicyPath(object, field.path, checked))} />
  }
  if (field.kind === "select") {
    return <SelectField field={field} label={label} namespace={namespace} value={textValue(value)}
      onChange={update} onFieldValidityChange={onFieldValidityChange} />
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
  return <TextField field={field} label={label} value={textValue(value)} onChange={update} />
}

export function PolicyFormFields({ fields, leading, ...props }: PolicyFormFieldsProps) {
  return (
    <FieldGroup className="grid gap-4 sm:grid-cols-2">
      {leading}{fields.map((field) => <PolicyField key={field.path} field={field} {...props} />)}
    </FieldGroup>
  )
}
