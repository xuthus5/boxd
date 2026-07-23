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
  isEndpointsStructureValid,
  normalizeEndpoints,
  prepareEndpoints,
} from "@/features/advanced/endpoints-form-model"
import { EndpointsVisualEditor } from "@/features/advanced/endpoints-visual-editor"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { type JsonObject } from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

const SECTION = "endpoints"

function parseEndpoints(value: string): JsonObject[] | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isEndpointsStructureValid(parsed) ? parsed : null
  } catch {
    return null
  }
}

function useEndpointsEditorState(initial: JsonValue | undefined) {
  const [value, setValue] = useState(() => JSON.stringify(normalizeEndpoints(initial), null, 2))
  const items = parseEndpoints(value)
  const updateItems = (next: JsonObject[]) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => setValue(next)
  return { value, items, updateItems, updateJSON }
}

function EndpointsEditor({ initial, onSave, saveError, onDismissError }: {
  initial: JsonValue | undefined
  onSave: (items: JsonObject[]) => void
  saveError: ReturnType<typeof useConfigSaveError>["saveError"]
  onDismissError: () => void
}) {
  const { t } = useTranslation()
  const editor = useEndpointsEditorState(initial)
  const editorRef = useRef<JsonEditorHandle>(null)
  const [activeTab, setActiveTab] = useState("visual")
  const structureValid = isEndpointsStructureValid(editor.items)
  const canSave = Boolean(editor.items && structureValid)

  const reveal = useCallback((path: string) => {
    setActiveTab("json")
    const candidates = [path]
    if (path.startsWith(`${SECTION}.`)) candidates.push(path.slice(SECTION.length + 1))
    if (path.startsWith(`${SECTION}[`)) candidates.push(path.slice(SECTION.length))
    if (!path.startsWith(SECTION)) candidates.push(`${SECTION}.${path.replace(/^\./, "")}`)
    const tryReveal = () => candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => {
      if (!tryReveal()) toast.message(t("config.pathNotFound", { path }))
    }, 50)
    return true
  }, [t])
  useConfigPathReveal((path) => reveal(path), { section: SECTION })

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle role="heading" aria-level={1} className="truncate">{t("pages.endpoints")}</CardTitle>
        <CardDescription>{t("advanced.endpointsDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <ConfigSaveErrorAlert
          error={saveError}
          onDismiss={onDismissError}
          onJumpToPath={reveal}
        />
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
            {editor.items && structureValid
              ? <EndpointsVisualEditor items={editor.items} onChange={editor.updateItems} />
              : (
                <Alert variant="destructive">
                  <AlertTitle>{t("advanced.endpointsInvalidStructureTitle")}</AlertTitle>
                  <AlertDescription>{t("advanced.endpointsInvalidStructureDescription")}</AlertDescription>
                </Alert>
              )}
          </TabsContent>
          <TabsContent value="json" className="pt-3 sm:pt-4">
            <FieldGroup className="gap-2 sm:gap-3">
              <Field>
                <FieldLabel className="sr-only">{t("advanced.endpointsJSON")}</FieldLabel>
                <JsonEditor
                  ref={editorRef}
                  value={editor.value}
                  onChange={editor.updateJSON}
                  ariaLabel={t("advanced.endpointsJSON")}
                />
              </Field>
            </FieldGroup>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className="flex-wrap justify-end gap-2">
        <Button
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={!canSave}
          onClick={() => editor.items && onSave(prepareEndpoints(editor.items))}
        >
          {t("advanced.save")}
        </Button>
      </CardFooter>
    </Card>
  )
}

export function EndpointsPage() {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>{t("common.loadFailed")}</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
      </Alert>
    )
  }
  const initial = query.data?.endpoints
  return (
    <EndpointsEditor
      key={JSON.stringify(initial ?? [])}
      initial={initial}
      saveError={saveError}
      onDismissError={clearSaveError}
      onSave={(items) => {
        clearSaveError()
        save.mutate(
          { ...query.data!, endpoints: items },
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
