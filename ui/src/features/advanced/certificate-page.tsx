import { useCallback, useRef, useState, type Dispatch, type RefObject, type SetStateAction } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  applyCertificateConfig,
  isCertificateStructureValid,
  normalizeCertificateObject,
} from "@/features/advanced/certificate-form-model"
import { CertificateVisualEditor } from "@/features/advanced/certificate-visual-editor"
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

const SECTION = "certificate"

function parseCertificateObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

function useCertificateEditorState(initial: JsonValue | undefined) {
  const [value, setValue] = useState(() => JSON.stringify(normalizeCertificateObject(initial), null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const parsed = parseCertificateObject(value)
  const object = parsed && isCertificateStructureValid(parsed) ? normalizeCertificateObject(parsed) : parsed
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

function useCertificateReveal(editorRef: RefObject<JsonEditorHandle | null>, setActiveTab: Dispatch<SetStateAction<string>>) {
  const { t } = useTranslation()
  return useCallback((path: string) => {
    setActiveTab("json")
    const candidates = [path]
    if (path.startsWith(`${SECTION}.`)) candidates.push(path.slice(SECTION.length + 1))
    if (!path.startsWith(SECTION)) candidates.push(`${SECTION}.${path.replace(/^\./, "")}`)
    const tryReveal = () => candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => { if (!tryReveal()) toast.message(t("config.pathNotFound", { path })) }, 50)
    return true
  }, [editorRef, setActiveTab, t])
}

interface CertificateEditorProps {
  initial: JsonValue | undefined
  fullConfig: SingBoxConfig
  saving: boolean
  onSave: (object: JsonObject) => void
  saveError: ReturnType<typeof useConfigSaveError>["saveError"]
  onDismissError: () => void
  reportError: (error: unknown) => ConfigSaveErrorState
  clearSaveError: () => void
}

function CertificateStructureAlert() {
  const { t } = useTranslation()
  return <Alert variant="destructive"><AlertTitle>{t("advanced.certificateInvalidStructureTitle")}</AlertTitle>
    <AlertDescription>{t("advanced.certificateInvalidStructureDescription")}</AlertDescription></Alert>
}

function CertificateJSONTab({ editorRef, value, onChange }: {
  editorRef: RefObject<JsonEditorHandle | null>; value: string; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup className="gap-2 sm:gap-3"><Field><FieldLabel className="sr-only">{t("advanced.certificateJSON")}</FieldLabel>
    <JsonEditor ref={editorRef} value={value} onChange={onChange} ariaLabel={t("advanced.certificateJSON")} />
  </Field></FieldGroup>
}

function CertificateEditor(props: CertificateEditorProps) {
  const { t } = useTranslation()
  const editor = useCertificateEditorState(props.initial)
  const editorRef = useRef<JsonEditorHandle>(null)
  const [activeTab, setActiveTab] = useState("visual")
  const structureValid = isCertificateStructureValid(editor.object)
  const canSave = Boolean(editor.object && structureValid && editor.invalidFields.size === 0)
  const reveal = useCertificateReveal(editorRef, setActiveTab)
  useConfigPathReveal((path) => reveal(path), { section: SECTION })
  const validation = useConfigValidate({
    buildConfig: () => editor.object && structureValid
      ? applyCertificateConfig(props.fullConfig, editor.object) : null,
    reportError: props.reportError,
    clearSaveError: props.clearSaveError,
    onReportedError: (error) => { if (error.path) reveal(error.path) },
    source: "validate_certificate",
  })
  const disabled = !canSave || validation.validating || props.saving
  return <Card size="sm">
    <CardHeader className="gap-1.5"><CardTitle role="heading" aria-level={1} className="truncate">{t("pages.certificate")}</CardTitle>
      <CardDescription>{t("advanced.certificateDescription")}</CardDescription></CardHeader>
    <CardContent className="flex flex-col gap-2 sm:gap-3">
      <ConfigSaveErrorAlert error={props.saveError} onDismiss={props.onDismissError} onJumpToPath={reveal} />
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(String(value || "visual"))} className="min-h-0 min-w-0">
        <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
          <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger><TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
        </TabsList>
        <TabsContent value="visual" className="pt-3 sm:pt-4">{editor.object && structureValid
          ? <CertificateVisualEditor object={editor.object} revision={editor.revision} onChange={editor.updateObject} onFieldValidityChange={editor.updateFieldValidity} />
          : <CertificateStructureAlert />}</TabsContent>
        <TabsContent value="json" className="pt-3 sm:pt-4"><CertificateJSONTab editorRef={editorRef} value={editor.value} onChange={editor.updateJSON} /></TabsContent>
      </Tabs>
    </CardContent>
    <CardFooter className="flex-wrap justify-end gap-2">
      <Button variant="outline" size="sm" className="h-8 w-full sm:w-auto" disabled={disabled} onClick={() => { void validation.validate() }}>
        {validation.validating ? t("advanced.validating") : t("advanced.validate")}</Button>
      <Button size="sm" className="h-8 w-full sm:w-auto" disabled={disabled}
        onClick={() => editor.object && props.onSave(editor.object)}>{t("advanced.save")}</Button>
    </CardFooter>
  </Card>
}

export function CertificatePage() {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const errors = useConfigSaveError()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <PageLoadErrorAlert error={query.error} scope="advanced-certificate" onRetry={() => { void query.refetch() }} />
  const initial = query.data?.certificate
  const saveCertificate = (object: JsonObject) => {
    errors.clearSaveError()
    save.mutate(applyCertificateConfig(query.data!, object), {
      onSuccess: (response) => {
        if (response.status === "rolled_back") { errors.reportRollback(response, t("advanced.rolledBack")); return }
        toast.success(t("advanced.saved"))
      },
      onError: (error) => { errors.reportError(error) },
    })
  }
  return <CertificateEditor key={JSON.stringify(initial ?? {})} initial={initial} fullConfig={query.data!}
    saving={save.isPending} saveError={errors.saveError} onDismissError={errors.clearSaveError}
    reportError={errors.reportError} clearSaveError={errors.clearSaveError} onSave={saveCertificate} />
}
