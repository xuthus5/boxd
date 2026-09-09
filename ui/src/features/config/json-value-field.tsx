import { useEffect, useId, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Textarea } from "@/components/ui/textarea"
import type { JsonValue } from "@/lib/api/types"

interface JsonValueFieldProps {
  path: string
  label: string
  value: JsonValue | undefined
  revision?: number
  onChange: (value: JsonValue | undefined) => void
  onValidity?: (path: string, valid: boolean) => void
}

const helpKeys: Record<string, string> = {
  optimistic: "kernel114.fields.optimisticCacheHelp",
  match_response: "kernel114.fields.matchResponseHelp",
  "tls.certificate_provider": "kernel114.fields.certificateProviderHelp",
  http_client: "kernel114.fields.httpClientHelp",
}

export function JsonValueField({ path, label, value, revision = 0, onChange, onValidity }: JsonValueFieldProps) {
  const { t } = useTranslation()
  const id = useId()
  const serialized = value === undefined ? "" : JSON.stringify(value, null, 2)
  const sourceKey = `${revision}:${serialized}`
  const [source, setSource] = useState(sourceKey)
  const [raw, setRaw] = useState(serialized)
  const [invalid, setInvalid] = useState(false)
  const validity = useRef(onValidity)
  useEffect(() => { validity.current = onValidity }, [onValidity])
  useEffect(() => { validity.current?.(path, !invalid) }, [invalid, path, sourceKey])
  useEffect(() => () => validity.current?.(path, true), [path])
  if (source !== sourceKey) { setSource(sourceKey); setRaw(serialized); setInvalid(false) }
  const update = (text: string) => {
    setRaw(text)
    try {
      const next = text.trim() ? JSON.parse(text) as JsonValue : undefined
      setInvalid(false)
      setSource(`${revision}:${next === undefined ? "" : JSON.stringify(next, null, 2)}`)
      onChange(next)
    } catch {
      setInvalid(true)
    }
  }
  return <Field className="sm:col-span-2" data-invalid={invalid}>
    <FieldLabel htmlFor={id}>{label}</FieldLabel>
    <Textarea id={id} aria-label={label} aria-invalid={invalid} value={raw} onChange={(event) => update(event.target.value)} />
    <FieldDescription>{t(invalid ? "kernel114.invalidJSONValue" : helpKeys[path] ?? "kernel114.jsonValueHint")}</FieldDescription>
  </Field>
}
