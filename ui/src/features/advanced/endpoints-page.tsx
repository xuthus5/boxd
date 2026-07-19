import { useState } from "react"
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
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import { type JsonObject } from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

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

function EndpointsEditor({ initial, onSave }: {
  initial: JsonValue | undefined
  onSave: (items: JsonObject[]) => void
}) {
  const { t } = useTranslation()
  const editor = useEndpointsEditorState(initial)
  const structureValid = isEndpointsStructureValid(editor.items)
  const canSave = Boolean(editor.items && structureValid)

  return (
    <Card>
      <CardHeader>
        <CardTitle role="heading" aria-level={1}>{t("pages.endpoints")}</CardTitle>
        <CardDescription>{t("advanced.endpointsDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="visual" className="min-w-0">
          <TabsList activateOnFocus className="max-w-full">
            <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger>
            <TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
          </TabsList>
          <TabsContent value="visual">
            {editor.items && structureValid
              ? <EndpointsVisualEditor items={editor.items} onChange={editor.updateItems} />
              : (
                <Alert variant="destructive">
                  <AlertTitle>{t("advanced.endpointsInvalidStructureTitle")}</AlertTitle>
                  <AlertDescription>{t("advanced.endpointsInvalidStructureDescription")}</AlertDescription>
                </Alert>
              )}
          </TabsContent>
          <TabsContent value="json">
            <FieldGroup>
              <Field>
                <FieldLabel className="sr-only">{t("advanced.endpointsJSON")}</FieldLabel>
                <JsonEditor value={editor.value} onChange={editor.updateJSON} ariaLabel={t("advanced.endpointsJSON")} />
              </Field>
            </FieldGroup>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className="flex-wrap justify-end gap-2">
        <Button
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
      onSave={(items) => save.mutate(
        { ...query.data!, endpoints: items },
        {
          onSuccess: (response) => response.status === "rolled_back"
            ? toast.error(t("advanced.rolledBack"))
            : toast.success(t("advanced.saved")),
          onError: (error) => toast.error(error.message),
        },
      )}
    />
  )
}

