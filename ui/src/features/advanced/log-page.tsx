import { useCallback, useRef, useState, type Dispatch, type RefObject, type SetStateAction } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { applyLogConfig, isLogStructureValid, normalizeLogObject } from "@/features/advanced/log-form-model"
import { LogVisualEditor } from "@/features/advanced/log-visual-editor"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { useConfigValidate } from "@/features/config/use-config-validate"
import { isJsonObject, type JsonObject } from "@/features/policy/policy-form-model"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

const SECTION = "log"

function parseLogObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

function useLogEditorState(initial: JsonValue | undefined) {
  const [value, setValue] = useState(() => JSON.stringify(normalizeLogObject(initial), null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const parsed = parseLogObject(value)
  const object = parsed && isLogStructureValid(parsed) ? normalizeLogObject(parsed) : parsed
  const updateObject = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => {
    setValue(next)
    setRevision((current) => current + 1)
    setInvalidFields(new Set())
  }
  const updateFieldValidity = (path: string, valid: boolean) => setInvalidFields((current) => {
    const next = new Set(current)
    if (valid) next.delete(path)
    else next.add(path)
    return next
  })
  return { value, revision, invalidFields, object, updateObject, updateJSON, updateFieldValidity }
}

function logPathCandidates(path: string) {
  const candidates = [path]
  if (path.startsWith(`${SECTION}.`)) candidates.push(path.slice(SECTION.length + 1))
  if (!path.startsWith(SECTION)) candidates.push(`${SECTION}.${path.replace(/^\./, "")}`)
  return candidates
}

function useLogReveal(editorRef: RefObject<JsonEditorHandle | null>, setTab: Dispatch<SetStateAction<string>>) {
  const { t } = useTranslation()
  return useCallback((path: string) => {
    setTab("json")
    const tryReveal = () => logPathCandidates(path).some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => { if (!tryReveal()) toast.message(t("config.pathNotFound", { path })) }, 50)
    return true
  }, [editorRef, setTab, t])
}

interface LogEditorProps {
  initial: JsonValue | undefined
  fullConfig: SingBoxConfig
  saving: boolean
  onSave: (object: JsonObject) => void
  saveError: ReturnType<typeof useConfigSaveError>["saveError"]
  onDismissError: () => void
  reportError: (error: unknown) => ConfigSaveErrorState
  clearSaveError: () => void
}

function LogStructureAlert() {
  const { t } = useTranslation()
  return <Alert variant="destructive">
    <AlertTitle>{t("advanced.logInvalidStructureTitle")}</AlertTitle>
    <AlertDescription>{t("advanced.logInvalidStructureDescription")}</AlertDescription>
  </Alert>
}

function LogJSONTab({ editorRef, value, onChange }: {
  editorRef: RefObject<JsonEditorHandle | null>; value: string; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup className="gap-2 sm:gap-3"><Field>
    <FieldLabel className="sr-only">{t("advanced.logJSON")}</FieldLabel>
    <JsonEditor ref={editorRef} value={value} onChange={onChange} ariaLabel={t("advanced.logJSON")} />
  </Field></FieldGroup>
}

function LogEditorTabs(props: {
  activeTab: string; setActiveTab: (value: string) => void; editor: ReturnType<typeof useLogEditorState>
  editorRef: RefObject<JsonEditorHandle | null>; structureValid: boolean
}) {
  const { t } = useTranslation()
  const { editor } = props
  return <Tabs value={props.activeTab} onValueChange={(value) => props.setActiveTab(String(value || "visual"))} className="min-h-0 min-w-0">
    <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
      <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger>
      <TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
    </TabsList>
    <TabsContent value="visual" className="pt-3 sm:pt-4">{editor.object && props.structureValid
      ? <LogVisualEditor object={editor.object} revision={editor.revision} onChange={editor.updateObject}
        onFieldValidityChange={editor.updateFieldValidity} />
      : <LogStructureAlert />}</TabsContent>
    <TabsContent value="json" className="pt-3 sm:pt-4">
      <LogJSONTab editorRef={props.editorRef} value={editor.value} onChange={editor.updateJSON} />
    </TabsContent>
  </Tabs>
}

function LogEditor(props: LogEditorProps) {
  const { t } = useTranslation()
  const editor = useLogEditorState(props.initial)
  const editorRef = useRef<JsonEditorHandle>(null)
  const [activeTab, setActiveTab] = useState("visual")
  const structureValid = isLogStructureValid(editor.object)
  const canSave = Boolean(editor.object && structureValid && editor.invalidFields.size === 0)
  const reveal = useLogReveal(editorRef, setActiveTab)
  useConfigPathReveal((path) => reveal(path), { section: SECTION })
  const validation = useConfigValidate({
    buildConfig: () => editor.object && structureValid ? applyLogConfig(props.fullConfig, editor.object) : null,
    reportError: props.reportError,
    clearSaveError: props.clearSaveError,
    onReportedError: (error) => { if (error.path) reveal(error.path) },
    source: "validate_log",
  })
  const disabled = !canSave || validation.validating || props.saving
  return <Card size="sm">
    <CardHeader className="gap-1.5"><CardTitle role="heading" aria-level={1} className="truncate">{t("pages.logConfig")}</CardTitle>
      <CardDescription>{t("advanced.logDescription")}</CardDescription></CardHeader>
    <CardContent className="flex flex-col gap-2 sm:gap-3">
      <ConfigSaveErrorAlert error={props.saveError} onDismiss={props.onDismissError} onJumpToPath={reveal} />
      <LogEditorTabs activeTab={activeTab} setActiveTab={setActiveTab} editor={editor}
        editorRef={editorRef} structureValid={structureValid} />
    </CardContent>
    <CardFooter className="flex-wrap justify-end gap-2">
      <Button variant="outline" size="sm" className="h-8 w-full sm:w-auto" disabled={disabled}
        onClick={() => { void validation.validate() }}>{validation.validating ? t("advanced.validating") : t("advanced.validate")}</Button>
      <Button size="sm" className="h-8 w-full sm:w-auto" disabled={disabled}
        onClick={() => editor.object && props.onSave(editor.object)}>{t("advanced.save")}</Button>
    </CardFooter>
  </Card>
}

export function LogPage() {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const errors = useConfigSaveError()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <PageLoadErrorAlert error={query.error} scope="advanced-log" onRetry={() => { void query.refetch() }} />
  const initial = query.data?.log
  const saveLog = (object: JsonObject) => {
    errors.clearSaveError()
    save.mutate(applyLogConfig(query.data!, object), {
      onSuccess: (response) => {
        if (response.status === "rolled_back") { errors.reportRollback(response, t("advanced.rolledBack")); return }
        toast.success(t("advanced.saved"))
      },
      onError: (error) => { errors.reportError(error) },
    })
  }
  return <LogEditor key={JSON.stringify(initial ?? {})} initial={initial} fullConfig={query.data!}
    saving={save.isPending} saveError={errors.saveError} onDismissError={errors.clearSaveError}
    reportError={errors.reportError} clearSaveError={errors.clearSaveError} onSave={saveLog} />
}
