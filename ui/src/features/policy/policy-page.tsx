import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import {
  isJsonObject,
  isPolicySectionStructureValid,
  type JsonObject,
  type PolicySection,
} from "@/features/policy/policy-form-model"
import type { APIEnvelope, JsonValue } from "@/lib/api/types"

export interface PolicyVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

interface PolicyPageProps {
  section: PolicySection
  title: string
  installLabel: string
  install: () => Promise<APIEnvelope<JsonValue>>
  renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode
}

interface PolicyEditorProps {
  section: PolicySection
  initialSection: JsonValue
  title: string
  installLabel: string
  onSave: (object: JsonObject) => void
  onInstall: () => void
  renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode
}

interface PolicyEditorTabsProps {
  section: PolicySection
  object: JsonObject | null
  revision: number
  value: string
  onChange: (object: JsonObject) => void
  onJSONChange: (value: string) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
  renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode
}

function parsePolicyObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

function usePolicyEditorState(initialSection: JsonValue) {
  const [value, setValue] = useState(() => JSON.stringify(initialSection, null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const object = parsePolicyObject(value)
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

function PolicyEditorTabs({
  section,
  object,
  revision,
  value,
  onChange,
  onJSONChange,
  onFieldValidityChange,
  renderVisual,
}: PolicyEditorTabsProps) {
  const { t } = useTranslation()
  const structureValid = Boolean(object && isPolicySectionStructureValid(section, object))
  return (
    <Tabs defaultValue="visual" className="min-w-0">
      <TabsList activateOnFocus className="max-w-full overflow-x-auto">
        <TabsTrigger value="visual">{t("policy.visualTab")}</TabsTrigger>
        <TabsTrigger value="json">{t("policy.advancedTab")}</TabsTrigger>
      </TabsList>
      <TabsContent value="visual">
        {object && structureValid
          ? renderVisual({ object, revision, onChange, onFieldValidityChange })
          : object ? <Alert variant="destructive">
            <AlertTitle>{t("policy.invalidStructureTitle")}</AlertTitle>
            <AlertDescription>{t("policy.invalidStructureDescription")}</AlertDescription>
          </Alert> : null}
      </TabsContent>
      <TabsContent value="json">
        <FieldGroup>
          <Field>
            <FieldLabel className="sr-only">{t("policy.jsonLabel")}</FieldLabel>
            <JsonEditor value={value} onChange={onJSONChange} ariaLabel={t("policy.jsonLabel")} />
          </Field>
        </FieldGroup>
      </TabsContent>
    </Tabs>
  )
}

function PolicyEditor({
  section,
  initialSection,
  title,
  installLabel,
  onSave,
  onInstall,
  renderVisual,
}: PolicyEditorProps) {
  const { t } = useTranslation()
  const editor = usePolicyEditorState(initialSection)
  const structureValid = Boolean(editor.object && isPolicySectionStructureValid(section, editor.object))
  const savePolicy = () => {
    if (editor.object) onSave(editor.object)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle role="heading" aria-level={1}>{title}</CardTitle>
        <CardDescription>{t("policy.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <PolicyEditorTabs
          section={section}
          object={editor.object}
          revision={editor.revision}
          value={editor.value}
          onChange={editor.updateObject}
          onJSONChange={editor.updateJSON}
          onFieldValidityChange={editor.updateFieldValidity}
          renderVisual={renderVisual}
        />
      </CardContent>
      <CardFooter className="flex-wrap justify-between gap-2">
        <Button variant="outline" onClick={onInstall}>{installLabel}</Button>
        <Button disabled={!editor.object || !structureValid || editor.invalidFields.size > 0} onClick={savePolicy}>
          {t("policy.save")}
        </Button>
      </CardFooter>
    </Card>
  )
}

export function PolicyPage({
  section,
  title,
  installLabel,
  install,
  renderVisual,
}: PolicyPageProps) {
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
  const persist = (object: JsonObject) => save.mutate({ ...query.data!, [section]: object }, {
    onSuccess: (response) => response.status === "rolled_back"
      ? toast.error(t("policy.rolledBack"))
      : toast.success(t("proxy.saved")),
    onError: (error) => toast.error(error.message),
  })
  const installDefaults = () => install()
    .then((response) => {
      if (response.status === "rolled_back") throw new Error(t("policy.rolledBack"))
      return query.refetch()
    })
    .then(() => toast.success(t("policy.installed")))
    .catch((error: Error) => toast.error(error.message))
  const initialSection = query.data?.[section] ?? {}

  return (
    <PolicyEditor
      section={section}
      key={JSON.stringify(initialSection)}
      initialSection={initialSection}
      title={title}
      installLabel={installLabel}
      onSave={persist}
      onInstall={installDefaults}
      renderVisual={renderVisual}
    />
  )
}
