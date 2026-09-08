import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldContent, FieldDescription, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field"
import { setupMessage } from "@/features/setup/setup-messages"
import type { SetupModule, SetupModuleID } from "@/lib/api/setup"

interface SetupModulesProps {
  modules: SetupModule[]
  selected: SetupModuleID[]
  disabled: boolean
  onChange: (selected: SetupModuleID[]) => void
}

export function SetupModules({ modules, selected, disabled, onChange }: SetupModulesProps) {
  const { t } = useTranslation()
  return <FieldSet disabled={disabled}>
    <FieldLegend>{t("setup.selectModules")}</FieldLegend>
    <FieldDescription>{t("setup.modulesHint")}</FieldDescription>
    <FieldGroup className="grid gap-3 sm:grid-cols-2">
      {modules.map((module) => <Field key={module.id} orientation="horizontal" data-disabled={disabled}>
        <Checkbox id={`setup-module-${module.id}`} checked={selected.includes(module.id)} disabled={disabled}
          onCheckedChange={(checked) => onChange(checked ? [...selected, module.id] : selected.filter((id) => id !== module.id))} />
        <FieldContent>
          <FieldLabel htmlFor={`setup-module-${module.id}`}>{t(`setup.${module.id}`)}</FieldLabel>
          <div className="flex flex-wrap gap-1.5">
            <Badge variant={module.state === "invalid" ? "destructive" : "outline"}>{t(`setup.${module.state}`)}</Badge>
            <Badge variant="secondary">{t("setup.count", { count: module.count })}</Badge>
          </div>
          {module.dependencies?.length ? <FieldDescription>{t("setup.dependsOn", { modules: module.dependencies.map((id) => t(`setup.${id}`)).join("、") })}</FieldDescription> : null}
          {module.message ? <FieldDescription>{setupMessage(module.message, t)}</FieldDescription> : null}
        </FieldContent>
      </Field>)}
    </FieldGroup>
  </FieldSet>
}
