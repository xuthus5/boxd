import { useEffect, useId, useState } from "react"
import { useTranslation } from "react-i18next"

import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  isNumericFieldKind,
  isNumericFieldRawValid,
} from "@/features/proxy/proxy-field-validation"
import type { FieldSpec } from "@/features/proxy/proxy-form-model"

import { CircleHelpIcon } from "lucide-react"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

type Namespace = "proxy.inbound" | "proxy.outbound"

function FieldHelp({ namespace, labelKey }: { namespace: Namespace; labelKey: string }) {
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

function FieldHeading({ id, label, namespace, labelKey }: { id: string; label: string; namespace: Namespace; labelKey: string }) {
  return <div className="flex items-center gap-1.5">
    <FieldLabel htmlFor={id}>{label}</FieldLabel>
    <FieldHelp namespace={namespace} labelKey={labelKey} />
  </div>
}

export function ProxyTextField({
  field,
  label,
  namespace,
  value,
  onChange,
  onFieldValidityChange,
}: {
  field: FieldSpec
  label: string
  namespace: Namespace
  value: string
  onChange: (value: string) => void
  onFieldValidityChange?: (path: string, valid: boolean) => void
}) {
  const { t } = useTranslation()
  const id = useId()
  const numeric = isNumericFieldKind(field.kind)
  const [draft, setDraft] = useState(value)
  const [source, setSource] = useState(value)
  if (source !== value) {
    setSource(value)
    setDraft(value)
  }
  const invalid = numeric && !isNumericFieldRawValid(field.kind, draft)
  useEffect(() => {
    if (!numeric) {
      onFieldValidityChange?.(field.path, true)
      return
    }
    onFieldValidityChange?.(field.path, !invalid)
    return () => onFieldValidityChange?.(field.path, true)
  }, [field.path, invalid, numeric, onFieldValidityChange])
  const area = field.kind === "textarea" || field.kind === "list" || field.kind === "number-list"
  const handleChange = (next: string) => {
    setDraft(next)
    if (!numeric || isNumericFieldRawValid(field.kind, next)) onChange(next)
  }
  const control = area
    ? <Textarea id={id} aria-label={label} aria-invalid={invalid || undefined} value={draft} onChange={(event) => handleChange(event.target.value)} />
    : <Input id={id} aria-label={label} aria-invalid={invalid || undefined} type="text" inputMode={field.kind === "number" ? "decimal" : undefined} value={draft} onChange={(event) => handleChange(event.target.value)} />
  const errorKey = field.kind === "number-list" ? "invalidNumberList" : "invalidNumber"
  return <Field data-invalid={invalid || undefined}>
    <FieldHeading id={id} label={label} namespace={namespace} labelKey={field.label} />
    {control}
    {invalid ? <FieldDescription>{t(`${namespace}.${errorKey}`)}</FieldDescription> : null}
  </Field>
}
