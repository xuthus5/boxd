import { useCallback, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle,
} from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  isExperimentalStructureValid,
  normalizeExperimentalObject,
  prepareExperimentalObject,
} from "@/features/advanced/experimental-form-model"
import { ExperimentalVisualEditor } from "@/features/advanced/experimental-visual-editor"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useConfigValidate } from "@/features/config/use-config-validate"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { isJsonObject, type JsonObject } from "@/features/policy/policy-form-model"
import { api } from "@/lib/api/endpoints"
import type { JsonValue } from "@/lib/api/types"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

function parseExperimentalObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

function useExperimentalEditorState(initial: JsonValue | undefined) {
  const [value, setValue] = useState(() => JSON.stringify(normalizeExperimentalObject(initial), null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const object = parseExperimentalObject(value)
  const updateObject = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => {
    setValue(next)
    setRevision((current) => current + 1)
    setInvalidFields(new Set())
  }
  const updateFieldValidity = (path: string, valid: boolean) => {
    setInvalidFields((current) => {
      const next = new Set(current)
      if (valid) next.delete(path)
      else next.add(path)
      return next
    })
  }
  return { value, revision, invalidFields, object, updateObject, updateJSON, updateFieldValidity }
}

function ExperimentalEditor({ initial, fullConfig, onSave, onInstallClashAPI, installing, saveError, onDismissError, reportError, clearSaveError }: {
  initial: JsonValue | undefined
  fullConfig: import("@/lib/api/types").SingBoxConfig
  onSave: (object: JsonObject) => void
  onInstallClashAPI: () => void
  installing: boolean
  saveError: ReturnType<typeof useConfigSaveError>["saveError"]
  onDismissError: () => void
  reportError: (error: unknown) => ConfigSaveErrorState
  clearSaveError: () => void
}) {
  const { t } = useTranslation()
  const editor = useExperimentalEditorState(initial)
  const editorRef = useRef<JsonEditorHandle>(null)
  const [activeTab, setActiveTab] = useState("visual")
  const structureValid = isExperimentalStructureValid(editor.object)
  const canSave = Boolean(editor.object && structureValid && editor.invalidFields.size === 0)
  const section = "experimental"
  const reveal = useCallback((path: string) => {
    setActiveTab("json")
    const candidates = [path]
    if (path.startsWith(`${section}.`)) candidates.push(path.slice(section.length + 1))
    if (path.startsWith(`${section}[`)) candidates.push(path.slice(section.length))
    if (!path.startsWith(section)) candidates.push(`${section}.${path.replace(/^\./, "")}`)
    const tryReveal = () => candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => {
      if (!tryReveal()) toast.message(t("config.pathNotFound", { path }))
    }, 50)
    return true
  }, [t])
  useConfigPathReveal((path) => reveal(path), { section })
  const { validating, validate } = useConfigValidate({
    buildConfig: () => {
      if (!editor.object) return null
      return { ...fullConfig, experimental: prepareExperimentalObject(editor.object) }
    },
    reportError,
    clearSaveError,
    onReportedError: (err) => { if (err.path) reveal(err.path) },
  })

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle role="heading" aria-level={1} className="truncate">{t("pages.experimental")}</CardTitle>
        <CardDescription>{t("advanced.experimentalDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <ConfigSaveErrorAlert error={saveError} onDismiss={onDismissError} onJumpToPath={reveal} />
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(String(value || "visual"))}
          className="min-h-0 min-w-0"
        >
          <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
            <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger>
            <TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
          </TabsList>
          <TabsContent value="visual" className="pt-3 sm:pt-4">
            {editor.object && structureValid
              ? <ExperimentalVisualEditor
                  object={editor.object}
                  revision={editor.revision}
                  onChange={editor.updateObject}
                  onFieldValidityChange={editor.updateFieldValidity}
                />
              : editor.object
                ? <Alert variant="destructive">
                  <AlertTitle>{t("advanced.invalidStructureTitle")}</AlertTitle>
                  <AlertDescription>{t("advanced.invalidStructureDescription")}</AlertDescription>
                </Alert>
                : null}
          </TabsContent>
          <TabsContent value="json" className="pt-3 sm:pt-4">
            <FieldGroup className="gap-2 sm:gap-3">
              <Field>
                <FieldLabel className="sr-only">{t("advanced.experimentalJSON")}</FieldLabel>
                <JsonEditor
                  ref={editorRef}
                  value={editor.value}
                  onChange={editor.updateJSON}
                  ariaLabel={t("advanced.experimentalJSON")}
                />
              </Field>
            </FieldGroup>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className="flex flex-col-reverse flex-wrap justify-between gap-2 sm:flex-row">
        <Button variant="outline" size="sm" className="h-8 w-full sm:w-auto" disabled={installing} onClick={onInstallClashAPI}>
          {t("advanced.installClashAPI")}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={!canSave || validating || installing}
          onClick={() => { void validate() }}
        >
          {validating ? t("advanced.validating") : t("advanced.validate")}
        </Button>
        <Button
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={!canSave || validating || installing}
          onClick={() => editor.object && onSave(prepareExperimentalObject(editor.object))}
        >
          {t("advanced.save")}
        </Button>
      </CardFooter>
    </Card>
  )
}

export function ExperimentalPage() {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
  const [installing, setInstalling] = useState(false)
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return <PageLoadErrorAlert error={query.error} scope="advanced-experimental" />
  }
  const initial = query.data?.experimental
  const installClashAPI = () => {
    setInstalling(true)
    clearSaveError()
    api.config.installExperimental()
      .then((response) => {
        if (response.status === "rolled_back") {
          reportRollback(response, t("advanced.rolledBack"))
          return
        }
        return query.refetch().then(() => toast.success(t("advanced.clashAPIInstalled")))
      })
      .catch((error: Error) => { reportError(error) })
      .finally(() => setInstalling(false))
  }
  return (
    <ExperimentalEditor
      key={JSON.stringify(initial ?? {})}
      initial={initial}
      fullConfig={query.data!}
      installing={installing}
      saveError={saveError}
      onDismissError={clearSaveError}
      reportError={reportError}
      clearSaveError={clearSaveError}
      onInstallClashAPI={installClashAPI}
      onSave={(object) => {
        clearSaveError()
        save.mutate(
          { ...query.data!, experimental: object },
          {
            onSuccess: (response) => {
              if (response.status === "rolled_back") {
                reportRollback(response, t("advanced.rolledBack"))
                return
              }
              toast.success(t("advanced.saved"))
            },
            onError: (error) => { reportError(error) },
          },
        )
      }}
    />
  )
}
