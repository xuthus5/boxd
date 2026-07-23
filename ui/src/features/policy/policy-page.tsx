import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { PolicyEditor, type PolicyVisualEditorProps } from "@/features/policy/policy-editor"
import {
  isJsonObject,
  type JsonObject,
  type PolicySection,
} from "@/features/policy/policy-form-model"
import type { APIEnvelope, JsonValue, RouteRuleMetadata } from "@/lib/api/types"
import { api } from "@/lib/api/endpoints"
import { rolledBackMessage } from "@/lib/api/status"

export type { PolicyVisualEditorProps } from "@/features/policy/policy-editor"

interface PolicyPageProps {
  section: PolicySection
  title: string
  installLabel: string
  install: () => Promise<APIEnvelope<JsonValue>>
  renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode
  afterSave?: () => Promise<void>
  afterInstall?: () => Promise<void>
  installInVisual?: boolean
}

export function PolicyPage({
  section,
  title,
  installLabel,
  install,
  renderVisual,
  afterSave,
  afterInstall,
  installInVisual,
}: PolicyPageProps) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>{t("common.loadFailed")}</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
      </Alert>
    )
  }
  const persist = (object: JsonObject) => {
    clearSaveError()
    save.mutate({ ...query.data!, [section]: object }, {
      onSuccess: async (response) => {
        if (response.status === "rolled_back") {
          reportRollback(response, t("policy.rolledBack"))
          return
        }
        try {
          await afterSave?.()
          toast.success(t("proxy.saved"))
        } catch (error) {
          /* c8 ignore next */
          reportError(error)
        }
      },
      onError: (error) => { reportError(error) },
    })
  }
  const installDefaults = () => {
    clearSaveError()
    install()
      .then((response) => {
        if (response.status === "rolled_back") throw new Error(rolledBackMessage(response, t("policy.rolledBack")))
        return query.refetch()
      })
      .then(() => afterInstall?.())
      .then(() => toast.success(t("policy.installed")))
      .catch((error: Error) => { reportError(error) })
  }
  const initialSection = query.data?.[section] ?? {}
  /* c8 ignore next 8 */
  const persistRules = (object: JsonObject, metadata: RouteRuleMetadata[]) => {
    const current = isJsonObject(initialSection) ? initialSection : {}
    const preserved: JsonObject = section === "route"
      ? { ...(object.rules === undefined ? {} : { rules: object.rules }), ...(object.rule_set === undefined ? {} : { rule_set: object.rule_set }) }
      : { ...(object.servers === undefined ? {} : { servers: object.servers }), ...(object.rules === undefined ? {} : { rules: object.rules }) }
    clearSaveError()
    save.mutate({ ...query.data!, [section]: { ...current, ...preserved } }, {
      onSuccess: async (response) => {
        if (response.status === "rolled_back") {
          reportRollback(response, t("policy.rolledBack"))
          return
        }
        try {
          if (metadata.length) await api.config.updateRouteRuleMetadata(metadata)
          toast.success(t("proxy.saved"))
        } catch (error) {
          reportError(error)
        }
      },
      onError: (error) => { reportError(error) },
    })
  }

  return (
    <div className="flex flex-col gap-4">
      <ConfigSaveErrorAlert error={saveError} onDismiss={clearSaveError} />
      <PolicyEditor
        section={section}
        key={JSON.stringify(initialSection)}
        initialSection={initialSection}
        title={title}
        installLabel={installLabel}
        onSave={persist}
        onInstall={installDefaults}
        renderVisual={renderVisual}
        installInVisual={installInVisual}
        onRulesChange={section === "route" || section === "dns" ? persistRules : undefined}
      />
    </div>
  )
}
