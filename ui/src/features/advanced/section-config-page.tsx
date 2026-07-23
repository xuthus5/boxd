import { useCallback, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

function SectionEditor({
  initial,
  section,
  fullConfig,
}: {
  initial: JsonValue
  section: string
  fullConfig: SingBoxConfig
}) {
  const { t } = useTranslation()
  const editorRef = useRef<JsonEditorHandle>(null)
  const [value, setValue] = useState(() => JSON.stringify(initial, null, 2))
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
  const save = useSaveConfigMutation()

  const reveal = useCallback((path: string) => {
    const candidates = [path]
    if (path.startsWith(`${section}.`)) candidates.push(path.slice(section.length + 1))
    if (path.startsWith(`${section}[`)) candidates.push(path.slice(section.length))
    if (!path.startsWith(section)) candidates.push(`${section}.${path.replace(/^\./, "")}`)
    const ok = candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (!ok) toast.message(t("config.pathNotFound", { path }))
    return ok
  }, [section, t])
  useConfigPathReveal(reveal, { section })

  const persist = () => {
    if (!isValidJSON(value)) return
    clearSaveError()
    const next = JSON.parse(value) as JsonValue
    save.mutate({ ...fullConfig, [section]: next }, {
      onSuccess: (response) => {
        if (response.status === "rolled_back") {
          const err = reportRollback(response, t("advanced.rolledBack"))
          if (err.path) reveal(err.path)
          return
        }
        toast.success(t("advanced.saved"))
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
        <FieldLabel className="sr-only">{t("advanced.advancedJSON")}</FieldLabel>
        <JsonEditor ref={editorRef} value={value} onChange={setValue} ariaLabel={t("advanced.advancedJSON")} />
      </Field>
      <ConfigSaveErrorAlert
        error={saveError}
        onDismiss={clearSaveError}
        onJumpToPath={reveal}
      />
      <Field>
        <Button
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={!isValidJSON(value) || save.isPending}
          onClick={persist}
        >
          {t("advanced.save")}
        </Button>
      </Field>
    </FieldGroup>
  )
}

export function SectionConfigPage({ section, title, description }: { section: string; title: string; description: string }) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>{t("common.loadFailed")}</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
      </Alert>
    )
  }
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle role="heading" aria-level={1} className="truncate">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <SectionEditor
          key={JSON.stringify(query.data?.[section] ?? (section === "endpoints" ? [] : {}))}
          section={section}
          fullConfig={query.data!}
          initial={query.data?.[section] ?? (section === "endpoints" ? [] : {})}
        />
      </CardContent>
    </Card>
  )
}
