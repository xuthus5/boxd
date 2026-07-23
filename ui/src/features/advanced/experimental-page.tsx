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
  isExperimentalStructureValid,
  normalizeExperimentalObject,
  prepareExperimentalObject,
} from "@/features/advanced/experimental-form-model"
import { ExperimentalVisualEditor } from "@/features/advanced/experimental-visual-editor"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import { isJsonObject, type JsonObject } from "@/features/policy/policy-form-model"
import { rolledBackMessage } from "@/lib/api/status"
import type { JsonValue } from "@/lib/api/types"

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

function ExperimentalEditor({ initial, onSave }: {
  initial: JsonValue | undefined
  onSave: (object: JsonObject) => void
}) {
  const { t } = useTranslation()
  const editor = useExperimentalEditorState(initial)
  const structureValid = isExperimentalStructureValid(editor.object)
  const canSave = Boolean(editor.object && structureValid && editor.invalidFields.size === 0)

  return (
    <Card>
      <CardHeader>
        <CardTitle role="heading" aria-level={1}>{t("pages.experimental")}</CardTitle>
        <CardDescription>{t("advanced.experimentalDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="visual" className="min-w-0">
          <TabsList activateOnFocus className="max-w-full">
            <TabsTrigger value="visual">{t("advanced.visualTab")}</TabsTrigger>
            <TabsTrigger value="json">{t("advanced.advancedTab")}</TabsTrigger>
          </TabsList>
          <TabsContent value="visual">
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
          <TabsContent value="json">
            <FieldGroup>
              <Field>
                <FieldLabel className="sr-only">{t("advanced.experimentalJSON")}</FieldLabel>
                <JsonEditor value={editor.value} onChange={editor.updateJSON} ariaLabel={t("advanced.experimentalJSON")} />
              </Field>
            </FieldGroup>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className="flex-wrap justify-end gap-2">
        <Button
          disabled={!canSave}
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
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return <Alert variant="destructive">
      <AlertTitle>{t("common.loadFailed")}</AlertTitle>
      <AlertDescription>{query.error.message}</AlertDescription>
    </Alert>
  }
  const initial = query.data?.experimental
  return (
    <ExperimentalEditor
      key={JSON.stringify(initial ?? {})}
      initial={initial}
      onSave={(object) => save.mutate(
        { ...query.data!, experimental: object },
        {
          onSuccess: (response) => response.status === "rolled_back"
            ? toast.error(rolledBackMessage(response, t("advanced.rolledBack")))
            : toast.success(t("advanced.saved")),
          onError: (error) => toast.error(error.message),
        },
      )}
    />
  )
}
