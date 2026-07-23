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
import { api } from "@/lib/api/endpoints"
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

function ExperimentalEditor({ initial, onSave, onInstallClashAPI, installing }: {
  initial: JsonValue | undefined
  onSave: (object: JsonObject) => void
  onInstallClashAPI: () => void
  installing: boolean
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
      <CardFooter className="flex-wrap justify-between gap-2">
        <Button variant="outline" disabled={installing} onClick={onInstallClashAPI}>
          {t("advanced.installClashAPI")}
        </Button>
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
  const [installing, setInstalling] = useState(false)
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return <Alert variant="destructive">
      <AlertTitle>{t("common.loadFailed")}</AlertTitle>
      <AlertDescription>{query.error.message}</AlertDescription>
    </Alert>
  }
  const initial = query.data?.experimental
  const installClashAPI = () => {
    setInstalling(true)
    api.config.installExperimental()
      .then((response) => {
        if (response.status === "rolled_back") {
          toast.error(rolledBackMessage(response, t("advanced.rolledBack")))
          return
        }
        return query.refetch().then(() => toast.success(t("advanced.clashAPIInstalled")))
      })
      .catch((error: Error) => toast.error(error.message))
      .finally(() => setInstalling(false))
  }
  return (
    <ExperimentalEditor
      key={JSON.stringify(initial ?? {})}
      initial={initial}
      installing={installing}
      onInstallClashAPI={installClashAPI}
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
