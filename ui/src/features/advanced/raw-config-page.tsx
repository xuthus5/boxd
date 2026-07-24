import { useCallback, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { ConfigDiffPanel } from "@/features/config/config-diff-panel"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { useConfigValidate } from "@/features/config/use-config-validate"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useRawConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { diffConfig, formatConfigDiffSummary } from "@/features/config/config-diff"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import type { SingBoxConfig } from "@/lib/api/types"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

function RawEditor({ initial }: { initial: SingBoxConfig }) {
  const { t } = useTranslation()
  const editorRef = useRef<JsonEditorHandle>(null)
  const [value, setValue] = useState(() => JSON.stringify(initial, null, 2))
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
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
  const reveal = useCallback((path: string) => {
    const ok = editorRef.current?.revealPath(path) ?? false
    if (!ok) toast.message(t("config.pathNotFound", { path }))
    return ok
  }, [t])
  useConfigPathReveal(reveal)
  const { validating, validate: runValidate } = useConfigValidate({
    buildConfig: () => nextConfig,
    reportError,
    clearSaveError,
    onReportedError: (err) => { if (err.path) reveal(err.path) },
    source: "validate_raw",
  })
  const persist = () => {
    if (!nextConfig) return
    clearSaveError()
    save.mutate(nextConfig, {
      onSuccess: (response) => {
        if (response.status === "rolled_back") {
          const err = reportRollback(response, t("advanced.rolledBack"))
          if (err.path) reveal(err.path)
          return
        }
        toast.success(t("advanced.rawSaved"))
      },
      onError: (error) => {
        const err = reportError(error)
        if (err.path) reveal(err.path)
      },
    })
  }
  return (
    <FieldGroup className="gap-2 sm:gap-3">
      <Field>
        <FieldLabel className="sr-only">{t("advanced.rawJSON")}</FieldLabel>
        <JsonEditor ref={editorRef} value={value} onChange={setValue} ariaLabel={t("advanced.rawJSON")} />
      </Field>
      <ConfigSaveErrorAlert
        error={saveError}
        onDismiss={clearSaveError}
        onJumpToPath={reveal}
      />
      <ConfigDiffPanel items={diffItems} onSelectPath={reveal} />
      <Field orientation="horizontal" className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <Button variant="outline" size="sm" className="h-8 w-full sm:w-auto" onClick={() => { setValue(JSON.stringify(initial, null, 2)); clearSaveError() }}>
          {t("advanced.reset")}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={!valid || validating || save.isPending}
          onClick={() => { void runValidate() }}
        >
          {validating ? t("advanced.validating") : t("advanced.validate")}
        </Button>
        <ConfirmAction
          trigger={<Button size="sm" className="h-8 w-full sm:w-auto" disabled={!valid || save.isPending || validating}>{t("advanced.saveRaw")}</Button>}
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
  if (query.error) {
    return (
      <PageLoadErrorAlert
        error={query.error}
        scope="advanced-raw"
        onRetry={() => { void query.refetch() }}
      />
    )
  }
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle role="heading" aria-level={1} className="truncate">{t("pages.rawConfig")}</CardTitle>
        <CardDescription>{t("advanced.rawDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <RawEditor key={JSON.stringify(query.data)} initial={query.data!} />
      </CardContent>
    </Card>
  )
}
