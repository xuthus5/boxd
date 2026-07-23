import { useEffect, useId, useState } from "react"
import { useTranslation } from "react-i18next"

import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Textarea } from "@/components/ui/textarea"
import type { FieldSpec } from "@/features/proxy/proxy-form-model"
import type { JsonValue } from "@/lib/api/types"

function FieldHeading({ id, label }: { id: string; label: string }) {
  return <FieldLabel htmlFor={id}>{label}</FieldLabel>
}

export function ProxyJSONField({ field, label, namespace, revision = 0, value, array, onChange, onFieldValidityChange }: { field: FieldSpec; label: string; namespace: "proxy.inbound" | "proxy.outbound"; revision?: number; value: JsonValue | undefined; array: boolean; onChange: (value: JsonValue | undefined) => void; onFieldValidityChange?: (path: string, valid: boolean) => void }) {
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
    <FieldHeading id={id} label={label} />
    <Textarea id={id} aria-label={label} aria-invalid={invalid} value={raw} onChange={(event) => update(event.target.value)} />
    <FieldDescription>{invalid ? t(`${namespace}.invalidStructuredJSON`) : t(array ? `${namespace}.usersJSONHint` : `${namespace}.jsonObjectHint`)}</FieldDescription>
  </Field>
}

