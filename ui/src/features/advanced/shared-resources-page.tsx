import { useCallback, useRef, useState, type RefObject } from "react"
import { useTranslation } from "react-i18next"
import { PencilIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { SharedResourceDialog } from "@/features/advanced/shared-resource-dialog"
import { createSharedResource, parseSharedResources, type SharedResourceSection } from "@/features/advanced/shared-resource-model"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { useConfigValidate } from "@/features/config/use-config-validate"
import type { JsonObject } from "@/features/policy/policy-form-model"
import type { SingBoxConfig } from "@/lib/api/types"

function ResourceList({ section, items, onChange }: {
  section: SharedResourceSection; items: JsonObject[]; onChange: (items: JsonObject[]) => void
}) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState<{ index: number; item: JsonObject } | null>(null)
  return <div className="flex flex-col gap-3">
    <Button size="sm" className="self-start" onClick={() => setEditing({ index: -1, item: createSharedResource(section) })}>
      <PlusIcon />{t(`kernel114.${section}.add`)}</Button>
    {items.length === 0 ? <p className="text-sm text-muted-foreground">{t("kernel114.emptyResources")}</p> : null}
    {items.map((item, index) => <div key={index} className="flex items-center gap-2 rounded-lg border p-3">
      <div className="min-w-0 flex-1"><p className="truncate text-sm font-medium">{String(item.tag || `#${index + 1}`)}</p>
        <p className="text-xs text-muted-foreground">{String(item.type || (section === "http_clients" ? "HTTP" : ""))}</p></div>
      <Button size="icon-sm" variant="outline" aria-label={t("kernel114.editResource", { tag: item.tag })} onClick={() => setEditing({ index, item })}><PencilIcon /></Button>
      <ConfirmAction trigger={<Button size="icon-sm" variant="destructive" aria-label={t("kernel114.deleteResource", { tag: item.tag })}><Trash2Icon /></Button>}
        title={t("kernel114.deleteResource", { tag: item.tag })} description={t("kernel114.deleteDescription")}
        confirmLabel={t("common.confirmDelete")} confirmVariant="destructive" onConfirm={() => onChange(items.filter((_, position) => index !== position))} />
    </div>)}
    {editing ? <SharedResourceDialog section={section} item={editing.item} onClose={() => setEditing(null)} onSave={(item) => {
      onChange(editing.index < 0 ? [...items, item] : items.map((current, index) => index === editing.index ? item : current))
      setEditing(null)
    }} /> : null}
  </div>
}

function useResourceEditor(section: SharedResourceSection, config: SingBoxConfig, editorRef: RefObject<JsonEditorHandle | null>) {
  const { t } = useTranslation()
  const [raw, setRaw] = useState(JSON.stringify(config[section] ?? [], null, 2))
  const [tab, setTab] = useState("visual")
  const errors = useConfigSaveError()
  const save = useSaveConfigMutation()
  const items = parseSharedResources(raw)
  const reveal = useCallback((path: string) => {
    setTab("advanced")
    const relative = path.startsWith(section) ? path.slice(section.length) : path
    window.setTimeout(() => {
      if (!editorRef.current?.revealPath(relative)) toast.message(t("config.pathNotFound", { path }))
    }, 0)
    return true
  }, [editorRef, section, t])
  useConfigPathReveal(reveal, { section })
  const validation = useConfigValidate({ buildConfig: () => items ? { ...config, [section]: items } : null,
    reportError: errors.reportError, clearSaveError: errors.clearSaveError,
    onReportedError: (error) => { if (error.path) reveal(error.path) }, source: "validate" })
  const persist = () => {
    if (!items) return
    errors.clearSaveError()
    save.mutate({ ...config, [section]: items }, {
      onSuccess: (response) => {
        if (response.status === "rolled_back") { errors.reportRollback(response, t("advanced.rolledBack")); return }
        toast.success(t("advanced.saved"))
      },
      onError: (error) => { const failure = errors.reportError(error); if (failure.path) reveal(failure.path) },
    })
  }
  return { raw, setRaw, tab, setTab, items, errors, validation, reveal, persist, saving: save.isPending }
}

function ResourceEditor({ section, config }: { section: SharedResourceSection; config: SingBoxConfig }) {
  const { t } = useTranslation()
  const editorRef = useRef<JsonEditorHandle>(null)
  const state = useResourceEditor(section, config, editorRef)
  const busy = state.saving || state.validation.validating
  return <Card size="sm">
    <CardHeader><CardTitle role="heading" aria-level={1}>{t(`kernel114.${section}.title`)}</CardTitle>
      <CardDescription>{t(`kernel114.${section}.description`)}</CardDescription></CardHeader>
    <CardContent className="flex flex-col gap-3">
      <ConfigSaveErrorAlert error={state.errors.saveError} onDismiss={state.errors.clearSaveError} onJumpToPath={state.reveal} />
      <Tabs value={state.tab} onValueChange={(value) => state.setTab(String(value))}>
        <TabsList><TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger><TabsTrigger value="advanced">{t("advanced.advancedTab")}</TabsTrigger></TabsList>
        <TabsContent value="visual" className="pt-4">{state.items ? <ResourceList section={section} items={state.items}
          onChange={(items) => state.setRaw(JSON.stringify(items, null, 2))} /> : <p role="alert">{t("kernel114.invalidResourceList")}</p>}</TabsContent>
        <TabsContent value="advanced" className="pt-4"><JsonEditor ref={editorRef} value={state.raw} onChange={state.setRaw} ariaLabel={t("kernel114.resourceJSON")} /></TabsContent>
      </Tabs>
    </CardContent>
    <CardFooter className="justify-end gap-2"><Button variant="outline" disabled={!state.items || busy} onClick={() => { void state.validation.validate() }}>{t("advanced.validate")}</Button>
      <Button disabled={!state.items || busy} onClick={state.persist}>{t("advanced.save")}</Button></CardFooter>
  </Card>
}

export function SharedResourcesPage({ section }: { section: SharedResourceSection }) {
  const query = useConfigQuery()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <PageLoadErrorAlert error={query.error} scope="advanced-section" onRetry={() => { void query.refetch() }} />
  return <ResourceEditor key={`${section}:${JSON.stringify(query.data?.[section])}`} section={section} config={query.data ?? {}} />
}
