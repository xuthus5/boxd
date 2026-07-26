import { useCallback, useRef, useState, type Dispatch, type RefObject, type SetStateAction } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  applyServicesConfig,
  isServicesStructureValid,
  normalizeServices,
} from "@/features/advanced/services-form-model"
import { ServicesVisualEditor } from "@/features/advanced/services-visual-editor"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { useConfigValidate } from "@/features/config/use-config-validate"
import type { JsonObject } from "@/features/policy/policy-form-model"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

const SECTION = "services"

function parseServices(value: string): JsonObject[] | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isServicesStructureValid(parsed) ? parsed : null
  } catch {
    return null
  }
}

function useServicesEditorState(initial: JsonValue | undefined) {
  const [value, setValue] = useState(() => JSON.stringify(
    isServicesStructureValid(initial) ? normalizeServices(initial) : initial ?? [],
    null,
    2,
  ))
  const parsed = parseServices(value)
  const items = parsed && isServicesStructureValid(parsed) ? normalizeServices(parsed) : parsed
  const updateItems = (next: JsonObject[]) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => setValue(next)
  return { value, items, updateItems, updateJSON }
}

function useServicesReveal(editorRef: RefObject<JsonEditorHandle | null>, setActiveTab: Dispatch<SetStateAction<string>>) {
  const { t } = useTranslation()
  return useCallback((path: string) => {
    setActiveTab("json")
    const candidates = [path]
    if (path.startsWith(`${SECTION}.`)) candidates.push(path.slice(SECTION.length + 1))
    if (path.startsWith(`${SECTION}[`)) candidates.push(path.slice(SECTION.length))
    if (!path.startsWith(SECTION)) candidates.push(`${SECTION}.${path.replace(/^\.?/, "")}`)
    const tryReveal = () => candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => { if (!tryReveal()) toast.message(t("config.pathNotFound", { path })) }, 50)
    return true
  }, [editorRef, setActiveTab, t])
}

function ServicesStructureAlert() {
  const { t } = useTranslation()
  return <Alert variant="destructive"><AlertTitle>{t("advanced.services.invalidStructureTitle")}</AlertTitle>
    <AlertDescription>{t("advanced.services.invalidStructureDescription")}</AlertDescription></Alert>
}

function ServicesJSONTab({ editorRef, value, onChange }: {
  editorRef: React.RefObject<JsonEditorHandle | null>; value: string; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("advanced.services.itemJSON")}</FieldLabel>
    <JsonEditor ref={editorRef} value={value} onChange={onChange} ariaLabel={t("advanced.services.itemJSON")} />
  </Field></FieldGroup>
}

interface ServicesEditorProps {
  initial: JsonValue | undefined
  fullConfig: SingBoxConfig
  saving: boolean
  onSave: (items: JsonObject[]) => void
  saveError: ConfigSaveErrorState | null
  onDismissError: () => void
  reportError: (error: unknown) => ConfigSaveErrorState
  clearSaveError: () => void
}

function ServicesEditor(props: ServicesEditorProps) {
  const { t } = useTranslation()
  const editor = useServicesEditorState(props.initial)
  const editorRef = useRef<JsonEditorHandle>(null)
  const [activeTab, setActiveTab] = useState("visual")
  const structureValid = isServicesStructureValid(editor.items)
  const canSave = Boolean(editor.items && structureValid)
  const reveal = useServicesReveal(editorRef, setActiveTab)
  useConfigPathReveal((path) => reveal(path), { section: SECTION })
  const validation = useConfigValidate({
    buildConfig: () => editor.items && structureValid ? applyServicesConfig(props.fullConfig, editor.items) : null,
    reportError: props.reportError,
    clearSaveError: props.clearSaveError,
    onReportedError: (error) => { if (error.path) reveal(error.path) },
    source: "validate_services",
  })
  const disabled = !canSave || validation.validating || props.saving
  return <section className="flex min-w-0 flex-col gap-3 sm:gap-4">
    <header className="flex min-w-0 flex-col gap-1">
      <h1 className="truncate text-lg font-semibold">{t("pages.services")}</h1>
      <p className="text-sm text-muted-foreground">{t("advanced.servicesDescription")}</p>
    </header>
    <div className="flex flex-col gap-2 sm:gap-3">
      <ConfigSaveErrorAlert error={props.saveError} onDismiss={props.onDismissError} onJumpToPath={reveal} />
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(String(value || "visual"))} className="min-h-0 min-w-0">
        <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
          <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger><TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
        </TabsList>
        <TabsContent value="visual" className="pt-3 sm:pt-4">{editor.items && structureValid
          ? <ServicesVisualEditor items={editor.items} onChange={editor.updateItems} />
          : <ServicesStructureAlert />}</TabsContent>
        <TabsContent value="json" className="pt-3 sm:pt-4"><ServicesJSONTab editorRef={editorRef} value={editor.value} onChange={editor.updateJSON} /></TabsContent>
      </Tabs>
    </div>
    <footer className="flex flex-wrap justify-end gap-2">
      <Button variant="outline" size="sm" className="h-8 w-full sm:w-auto" disabled={disabled} onClick={() => { void validation.validate() }}>
        {validation.validating ? t("advanced.validating") : t("advanced.validate")}</Button>
      <Button size="sm" className="h-8 w-full sm:w-auto" disabled={disabled}
        onClick={() => editor.items && props.onSave(editor.items)}>{t("advanced.save")}</Button>
    </footer>
  </section>
}

export function ServicesPage() {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const errors = useConfigSaveError()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <PageLoadErrorAlert error={query.error} scope="advanced-services" onRetry={() => { void query.refetch() }} />
  const initial = query.data?.services
  const saveServices = (items: JsonObject[]) => {
    errors.clearSaveError()
    save.mutate(applyServicesConfig(query.data!, items), {
      onSuccess: (response) => {
        if (response.status === "rolled_back") { errors.reportRollback(response, t("advanced.rolledBack")); return }
        toast.success(t("advanced.saved"))
      },
      onError: (error) => { errors.reportError(error) },
    })
  }
  return <ServicesEditor key={JSON.stringify(initial ?? [])} initial={initial} fullConfig={query.data!}
    saving={save.isPending} saveError={errors.saveError} onDismissError={errors.clearSaveError}
    reportError={errors.reportError} clearSaveError={errors.clearSaveError} onSave={saveServices} />
}
