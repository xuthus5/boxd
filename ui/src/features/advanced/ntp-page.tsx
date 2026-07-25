import {
  useCallback,
  useRef,
  useState,
  type Dispatch,
  type RefObject,
  type SetStateAction,
} from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  isNTPStructureValid,
  normalizeNTPObject,
  prepareNTPObject,
} from "@/features/advanced/ntp-form-model"
import { NTPVisualEditor } from "@/features/advanced/ntp-visual-editor"
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

const SECTION = "ntp"

function parseNTPObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

function useNTPEditorState(initial: JsonValue | undefined) {
  const [value, setValue] = useState(() => JSON.stringify(normalizeNTPObject(initial), null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const parsed = parseNTPObject(value)
  const object = parsed && isNTPStructureValid(parsed) ? normalizeNTPObject(parsed) : parsed
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

function useNTPReveal(editorRef: RefObject<JsonEditorHandle | null>, setActiveTab: Dispatch<SetStateAction<string>>) {
  const { t } = useTranslation()
  return useCallback((path: string) => {
    setActiveTab("json")
    const candidates = [path]
    if (path.startsWith(`${SECTION}.`)) candidates.push(path.slice(SECTION.length + 1))
    if (!path.startsWith(SECTION)) candidates.push(`${SECTION}.${path.replace(/^\./, "")}`)
    const tryReveal = () => candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => {
      if (!tryReveal()) toast.message(t("config.pathNotFound", { path }))
    }, 50)
    return true
  }, [editorRef, setActiveTab, t])
}

interface NTPEditorProps {
  initial: JsonValue | undefined
  fullConfig: SingBoxConfig
  saving: boolean
  onSave: (object: JsonObject) => void
  saveError: ReturnType<typeof useConfigSaveError>["saveError"]
  onDismissError: () => void
  reportError: (error: unknown) => ConfigSaveErrorState
  clearSaveError: () => void
}

function NTPStructureAlert() {
  const { t } = useTranslation()
  return <Alert variant="destructive">
    <AlertTitle>{t("advanced.ntpInvalidStructureTitle")}</AlertTitle>
    <AlertDescription>{t("advanced.ntpInvalidStructureDescription")}</AlertDescription>
  </Alert>
}

function NTPJSONTab({ editorRef, value, onChange }: {
  editorRef: RefObject<JsonEditorHandle | null>; value: string; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup className="gap-2 sm:gap-3"><Field>
    <FieldLabel className="sr-only">{t("advanced.ntpJSON")}</FieldLabel>
    <JsonEditor ref={editorRef} value={value} onChange={onChange} ariaLabel={t("advanced.ntpJSON")} />
  </Field></FieldGroup>
}

function NTPEditor({ initial, fullConfig, saving, onSave, saveError, onDismissError, reportError, clearSaveError }: NTPEditorProps) {
  const { t } = useTranslation()
  const editor = useNTPEditorState(initial)
  const editorRef = useRef<JsonEditorHandle>(null)
  const [activeTab, setActiveTab] = useState("visual")
  const structureValid = isNTPStructureValid(editor.object)
  const canSave = Boolean(editor.object && structureValid && editor.invalidFields.size === 0)
  const reveal = useNTPReveal(editorRef, setActiveTab)
  useConfigPathReveal((path) => reveal(path), { section: SECTION })
  const { validating, validate } = useConfigValidate({
    buildConfig: () => editor.object && structureValid
      ? { ...fullConfig, ntp: prepareNTPObject(editor.object) }
      : null,
    reportError,
    clearSaveError,
    onReportedError: (error) => { if (error.path) reveal(error.path) },
    source: "validate_ntp",
  })
  const disabled = !canSave || validating || saving
  return <Card size="sm">
    <CardHeader className="gap-1.5"><CardTitle role="heading" aria-level={1} className="truncate">{t("pages.ntp")}</CardTitle>
      <CardDescription>{t("advanced.ntpDescription")}</CardDescription></CardHeader>
    <CardContent className="flex flex-col gap-2 sm:gap-3">
      <ConfigSaveErrorAlert error={saveError} onDismiss={onDismissError} onJumpToPath={reveal} />
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(String(value || "visual"))} className="min-h-0 min-w-0">
        <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
          <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger><TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
        </TabsList>
        <TabsContent value="visual" className="pt-3 sm:pt-4">{editor.object && structureValid
          ? <NTPVisualEditor object={editor.object} revision={editor.revision} onChange={editor.updateObject} onFieldValidityChange={editor.updateFieldValidity} />
          : <NTPStructureAlert />}</TabsContent>
        <TabsContent value="json" className="pt-3 sm:pt-4"><NTPJSONTab editorRef={editorRef} value={editor.value} onChange={editor.updateJSON} /></TabsContent>
      </Tabs>
    </CardContent>
    <CardFooter className="flex-wrap justify-end gap-2">
      <Button variant="outline" size="sm" className="h-8 w-full sm:w-auto" disabled={disabled} onClick={() => { void validate() }}>
        {validating ? t("advanced.validating") : t("advanced.validate")}
      </Button>
      <Button size="sm" className="h-8 w-full sm:w-auto" disabled={disabled} onClick={() => editor.object && onSave(prepareNTPObject(editor.object))}>{t("advanced.save")}</Button>
    </CardFooter>
  </Card>
}

export function NTPPage() {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <PageLoadErrorAlert error={query.error} scope="advanced-ntp" onRetry={() => { void query.refetch() }} />
  const initial = query.data?.ntp
  const saveNTP = (object: JsonObject) => {
    clearSaveError()
    save.mutate({ ...query.data!, ntp: object }, {
      onSuccess: (response) => {
        if (response.status === "rolled_back") {
          reportRollback(response, t("advanced.rolledBack"))
          return
        }
        toast.success(t("advanced.saved"))
      },
      onError: (error) => { reportError(error) },
    })
  }
  return <NTPEditor key={JSON.stringify(initial ?? {})} initial={initial} fullConfig={query.data!}
    saving={save.isPending} saveError={saveError} onDismissError={clearSaveError} reportError={reportError}
    clearSaveError={clearSaveError} onSave={saveNTP} />
}
