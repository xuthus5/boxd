import { useId } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  LOG_FILTER_PRESETS,
  summarizeLogLevels,
  type LogFilterPresetId,
  type LogSearchFilters,
} from "@/features/observability/log-filter-presets"
import { LogLevelSummaryBar } from "@/features/observability/log-level-summary"
import type { LogThreshold } from "@/features/observability/log-level"

export function LogFilters({
  filter,
  minimum,
  level,
  levelSummary,
  activePreset,
  onFilterChange,
  onMinimumChange,
  onLevelChange,
  onPreset,
  onClear,
}: {
  filter: string
  minimum: LogThreshold
  level?: string
  levelSummary: ReturnType<typeof summarizeLogLevels>
  activePreset: LogFilterPresetId | null
  onFilterChange: (value: string) => void
  onMinimumChange: (value: LogThreshold) => void
  onLevelChange: (next: Pick<LogSearchFilters, "level">) => void
  onPreset: (id: LogFilterPresetId) => void
  onClear: () => void
}) {
  const { t } = useTranslation()
  const searchId = useId()
  const levelId = useId()
  const levelDescriptionId = useId()
  const levels = [
    { label: t("observability.allLevels"), value: "all" },
    { label: "Debug", value: "debug" },
    { label: "Info", value: "info" },
    { label: "Warn", value: "warn" },
    { label: "Error", value: "error" },
  ]
  const hasFilter = Boolean(filter.trim()) || minimum !== "all" || Boolean(level)
  return <FieldGroup className="gap-2 sm:gap-3">
    <div className="flex flex-col gap-2 @md/field-group:flex-row @md/field-group:gap-3">
      <Field className="flex-1">
        <FieldLabel htmlFor={searchId}>{t("observability.searchLogs")}</FieldLabel>
        <Input id={searchId} className="h-8" aria-label={t("observability.searchLogs")} placeholder={t("observability.searchLogs")} value={filter} onChange={(event) => onFilterChange(event.target.value)} />
      </Field>
      <Field className="sm:w-52">
        <FieldLabel htmlFor={levelId}>{t("observability.minimumLogLevel")}</FieldLabel>
        <Select items={levels} value={minimum} onValueChange={(value) => onMinimumChange(String(value) as LogThreshold)}>
          <SelectTrigger id={levelId} aria-label={t("observability.minimumLogLevel")} aria-describedby={levelDescriptionId} className="h-8 w-full"><SelectValue /></SelectTrigger>
          <SelectContent><SelectGroup>
            {levels.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}
          </SelectGroup></SelectContent>
        </Select>
        <FieldDescription id={levelDescriptionId}>{t("observability.minimumLogLevelDescription")}</FieldDescription>
      </Field>
    </div>
    <LogLevelSummaryBar summary={levelSummary} filters={{ level }} onChange={onLevelChange} />
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="text-xs text-muted-foreground sm:text-sm">{t("observability.logPresets")}</span>
      {LOG_FILTER_PRESETS.map((preset) => (
        <Button
          key={preset.id}
          type="button"
          size="sm"
          className="h-7"
          variant={activePreset === preset.id ? "default" : "outline"}
          aria-pressed={activePreset === preset.id}
          onClick={() => onPreset(preset.id)}
        >
          {t(preset.labelKey)}
        </Button>
      ))}
      {hasFilter ? (
        <Button type="button" size="sm" className="h-7" variant="ghost" onClick={onClear}>
          {t("observability.clearLogFilter")}
        </Button>
      ) : null}
    </div>
  </FieldGroup>
}

