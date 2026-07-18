import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  isSpeedTestURLPreset,
  SPEED_TEST_URL_PRESETS,
} from "@/lib/speed-test-urls"

export interface ProbeURLFieldProps {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  description?: string
  placeholder?: string
  disabled?: boolean
  invalid?: boolean
  /** When true, empty values stay empty and select shows an unset option. */
  allowEmpty?: boolean
  emptyLabel?: string
  manualLabel?: string
  className?: string
}

function deriveMode(value: string, allowEmpty: boolean): string {
  if (!value) return allowEmpty ? "unset" : "manual"
  return isSpeedTestURLPreset(value) ? value : "manual"
}

export function ProbeURLField({
  id,
  label,
  value,
  onChange,
  description,
  placeholder,
  disabled = false,
  invalid = false,
  allowEmpty = false,
  emptyLabel,
  manualLabel,
  className,
}: ProbeURLFieldProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState(() => deriveMode(value, allowEmpty))
  const [source, setSource] = useState(value)
  if (source !== value) {
    setSource(value)
    setMode(deriveMode(value, allowEmpty))
  }

  const items = useMemo(() => {
    const next = [] as { value: string; label: string }[]
    if (allowEmpty) next.push({ value: "unset", label: emptyLabel ?? t("common.notSet") })
    for (const preset of SPEED_TEST_URL_PRESETS) next.push({ value: preset, label: preset })
    next.push({ value: "manual", label: t("settings.testURLManual") })
    return next
  }, [allowEmpty, emptyLabel, t])

  const resolvedManualLabel = manualLabel ?? t("settings.testURLManualInput")
  const manualPlaceholder = placeholder ?? t("settings.testURLManualPlaceholder")
  const selectValue = mode

  return (
    <Field data-invalid={invalid || undefined} className={className}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <div className="grid gap-2">
        <Select
          items={items}
          value={selectValue}
          disabled={disabled}
          onValueChange={(next) => {
            const selected = String(next)
            if (selected === "unset") {
              setMode("unset")
              onChange("")
              return
            }
            if (selected === "manual") {
              setMode("manual")
              if (isSpeedTestURLPreset(value) || !value) onChange("")
              return
            }
            setMode(selected)
            onChange(selected)
          }}
        >
          <SelectTrigger
            id={id}
            aria-label={label}
            aria-invalid={invalid || undefined}
            disabled={disabled}
            className="w-full"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {allowEmpty ? <SelectItem value="unset">{emptyLabel ?? t("common.notSet")}</SelectItem> : null}
              {SPEED_TEST_URL_PRESETS.map((preset) => (
                <SelectItem key={preset} value={preset}>{preset}</SelectItem>
              ))}
              <SelectItem value="manual">{t("settings.testURLManual")}</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
        {mode === "manual" ? (
          <Input
            id={`${id}-manual`}
            aria-label={resolvedManualLabel}
            aria-invalid={invalid || undefined}
            disabled={disabled}
            value={isSpeedTestURLPreset(value) ? "" : value}
            placeholder={manualPlaceholder}
            onChange={(event) => onChange(event.target.value)}
          />
        ) : null}
      </div>
      {description ? <FieldDescription>{description}</FieldDescription> : null}
    </Field>
  )
}
