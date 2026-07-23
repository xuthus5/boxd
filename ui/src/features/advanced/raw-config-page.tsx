import { useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { ConfigDiffPanel } from "@/features/config/config-diff-panel"
import { useRawConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { diffConfig, formatConfigDiffSummary } from "@/features/config/config-diff"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import type { SingBoxConfig } from "@/lib/api/types"
import { rolledBackMessage, saveErrorMessage } from "@/lib/api/status"
import { parseConfigError } from "@/lib/api/config-error"

function RawEditor({ initial }: { initial: SingBoxConfig }) {
  const { t } = useTranslation()
  const editorRef = useRef<JsonEditorHandle>(null)
  const [value, setValue] = useState(() => JSON.stringify(initial, null, 2))
  const [errorPath, setErrorPath] = useState<string | null>(null)
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
  const reveal = (path: string) => {
    const ok = editorRef.current?.revealPath(path) ?? false
    if (!ok) toast.message(t("config.pathNotFound", { path }))
  }
  const persist = () => {
    if (!nextConfig) return
    setErrorPath(null)
    save.mutate(nextConfig, {
      onSuccess: (response) => response.status === "rolled_back"
        ? toast.error(rolledBackMessage(response, t("advanced.rolledBack")))
        : toast.success(t("advanced.rawSaved")),
      onError: (error) => {
        const message = saveErrorMessage(error)
        toast.error(message)
        const parsed = parseConfigError(error.message)
        if (parsed.path) {
          setErrorPath(parsed.path)
          reveal(parsed.path)
        }
      },
    })
  }
  return (
    <FieldGroup>
      <Field>
        <FieldLabel className="sr-only">{t("advanced.rawJSON")}</FieldLabel>
        <JsonEditor ref={editorRef} value={value} onChange={setValue} ariaLabel={t("advanced.rawJSON")} />
      </Field>
      {errorPath ? (
        <Alert variant="destructive" data-testid="raw-config-error-path">
          <AlertTitle>{t("config.errorPathTitle")}</AlertTitle>
          <AlertDescription className="flex flex-wrap items-center gap-2">
            <span>{t("config.errorPathDescription", { path: errorPath })}</span>
            <Button type="button" size="xs" variant="outline" onClick={() => reveal(errorPath)}>
              {t("config.jumpToPath")}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}
      <ConfigDiffPanel items={diffItems} onSelectPath={reveal} />
      <Field orientation="horizontal">
        <Button variant="outline" onClick={() => { setValue(JSON.stringify(initial, null, 2)); setErrorPath(null) }}>
          {t("advanced.reset")}
        </Button>
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
