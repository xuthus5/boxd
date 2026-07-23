import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { useRawConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import { diffConfig, formatConfigDiffSummary } from "@/features/config/config-diff"
import { isValidJSON } from "@/features/config/json-utils"
import type { SingBoxConfig } from "@/lib/api/types"
import { rolledBackMessage } from "@/lib/api/status"

function RawEditor({ initial }: { initial: SingBoxConfig }) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(initial, null, 2))
  const save = useSaveConfigMutation(true)
  const valid = isValidJSON(value)
  const nextConfig = valid ? JSON.parse(value) as SingBoxConfig : null
  const diffItems = nextConfig ? diffConfig(initial, nextConfig) : []
  const diffSummary = formatConfigDiffSummary(diffItems, {
    added: t("advanced.diffAdded"),
    removed: t("advanced.diffRemoved"),
    changed: t("advanced.diffChanged"),
    none: t("advanced.diffNone"),
    more: t("advanced.diffMore"),
  })
  const persist = () => {
    if (!nextConfig) return
    save.mutate(nextConfig, {
      onSuccess: (response) => response.status === "rolled_back"
        ? toast.error(rolledBackMessage(response, t("advanced.rolledBack")))
        : toast.success(t("advanced.rawSaved")),
      onError: (error) => toast.error(error.message),
    })
  }
  return (
    <FieldGroup>
      <Field>
        <FieldLabel className="sr-only">{t("advanced.rawJSON")}</FieldLabel>
        <JsonEditor value={value} onChange={setValue} ariaLabel={t("advanced.rawJSON")} />
      </Field>
      <p className="text-sm text-muted-foreground" data-testid="raw-config-diff">{diffSummary}</p>
      <Field orientation="horizontal">
        <Button variant="outline" onClick={() => setValue(JSON.stringify(initial, null, 2))}>{t("advanced.reset")}</Button>
        <ConfirmAction
          trigger={<Button disabled={!valid || save.isPending}>{t("advanced.saveRaw")}</Button>}
          title={t("advanced.overwriteTitle")}
          description={`${t("advanced.overwriteDescription")}
${diffSummary}`}
          confirmLabel={t("advanced.confirmOverwrite")}
          onConfirm={persist}
        />
      </Field>
    </FieldGroup>
  )
}

export function RawConfigPage() {
  const { t } = useTranslation()
  const query = useRawConfigQuery()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert>
  return <Card><CardHeader><CardTitle role="heading" aria-level={1}>{t("pages.rawConfig")}</CardTitle><CardDescription>{t("advanced.rawDescription")}</CardDescription></CardHeader><CardContent><RawEditor key={JSON.stringify(query.data)} initial={query.data!} /></CardContent></Card>
}
