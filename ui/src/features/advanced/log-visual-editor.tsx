import { BanIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { logFields } from "@/features/advanced/log-form-model"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import type { JsonObject } from "@/features/policy/policy-form-model"

export interface LogVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

function DisabledLogWarning() {
  const { t } = useTranslation()
  return <Alert variant="destructive">
    <BanIcon />
    <AlertTitle>{t("advanced.log.disabledWarningTitle")}</AlertTitle>
    <AlertDescription>{t("advanced.log.disabledWarningDescription")}</AlertDescription>
  </Alert>
}

export function LogVisualEditor(props: LogVisualEditorProps) {
  const shared = {
    object: props.object,
    revision: props.revision,
    onChange: props.onChange,
    onFieldValidityChange: props.onFieldValidityChange,
  }
  return <div className="flex flex-col gap-3 sm:gap-4">
    {props.object.disabled === true ? <DisabledLogWarning /> : null}
    <PolicyFormFields fields={logFields} namespace="advanced.log" {...shared} />
  </div>
}
