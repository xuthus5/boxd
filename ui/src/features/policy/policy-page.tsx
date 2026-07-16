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
import { api } from "@/lib/api/endpoints"

export interface PolicyVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
  onRulesChange?: (object: JsonObject, metadata: import("@/lib/api/types").RouteRuleMetadata[]) => void
  onInstall?: () => void
  onGlobalSave?: (object: JsonObject) => void
}

interface PolicyPageProps {
  section: PolicySection
  title: string
  installLabel: string
  install: () => Promise<APIEnvelope<JsonValue>>
  renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode
  afterSave?: () => Promise<void>
  afterInstall?: () => Promise<void>
  installInVisual?: boolean
}

interface PolicyEditorProps {
  section: PolicySection
  initialSection: JsonValue
  title: string
  installLabel: string
  onSave: (object: JsonObject) => void
  onInstall: () => void
  renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode
  installInVisual?: boolean
  onRulesChange?: (object: JsonObject, metadata: import("@/lib/api/types").RouteRuleMetadata[]) => void
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
  onRulesChange?: (object: JsonObject, metadata: import("@/lib/api/types").RouteRuleMetadata[]) => void
  onInstall?: () => void
  onGlobalSave?: (object: JsonObject) => void
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
  onRulesChange,
  onInstall,
  onGlobalSave,
}: PolicyEditorTabsProps) {
  const { t } = useTranslation()
  const structureValid = Boolean(object && isPolicySectionStructureValid(section, object))
  return (
    <Tabs defaultValue="visual" className="min-w-0">
      <TabsList activateOnFocus className="max-w-full">
        <TabsTrigger value="visual">{t("policy.visualTab")}</TabsTrigger>
        <TabsTrigger value="json">{t("policy.advancedTab")}</TabsTrigger>
      </TabsList>
      <TabsContent value="visual">
        {object && structureValid
          ? renderVisual({ object, revision, onChange, onFieldValidityChange, onRulesChange, onInstall, onGlobalSave })
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
  installInVisual,
  onRulesChange,
}: PolicyEditorProps) {
  const { t } = useTranslation()
  const editor = usePolicyEditorState(initialSection)
  const structureValid = Boolean(editor.object && isPolicySectionStructureValid(section, editor.object))
  const savePolicy = () => {
    if (editor.object) onSave(editor.object)
  }
  /* c8 ignore next 3 */
  const saveGlobal = (object: JsonObject) => {
    const initial = isJsonObject(initialSection) ? initialSection : {}
    onSave({ ...object, rules: initial.rules, rule_set: initial.rule_set })
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
          onRulesChange={onRulesChange}
          onInstall={onInstall}
          onGlobalSave={section === "route" ? saveGlobal : undefined}
        />
      </CardContent>
      <CardFooter className="flex-wrap justify-end gap-2">
        {!installInVisual ? <Button variant="outline" onClick={onInstall}>{installLabel}</Button> : null}
        {!installInVisual ? <Button disabled={!editor.object || !structureValid || editor.invalidFields.size > 0} onClick={savePolicy}>
          {t("policy.save")}
        </Button> : null}
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
  afterSave,
  afterInstall,
  installInVisual,
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
    onSuccess: async (response) => {
      if (response.status === "rolled_back") {
        toast.error(t("policy.rolledBack"))
        return
      }
      try {
        await afterSave?.()
        toast.success(t("proxy.saved"))
      } catch (error) {
        /* c8 ignore next */
        toast.error(error instanceof Error ? error.message : String(error))
      }
    },
    onError: (error) => toast.error(error.message),
  })
  const installDefaults = () => install()
    .then((response) => {
      if (response.status === "rolled_back") throw new Error(t("policy.rolledBack"))
      return query.refetch()
    })
    .then(() => afterInstall?.())
    .then(() => toast.success(t("policy.installed")))
    .catch((error: Error) => toast.error(error.message))
  const initialSection = query.data?.[section] ?? {}
  /* c8 ignore next 8 */
  const persistRules = (object: JsonObject, metadata: import("@/lib/api/types").RouteRuleMetadata[]) => {
    const current = isJsonObject(initialSection) ? initialSection : {}
    const route = { ...current, rules: object.rules, rule_set: object.rule_set }
    save.mutate({ ...query.data!, [section]: route }, {
      onSuccess: async (response) => {
        if (response.status === "rolled_back") { toast.error(t("policy.rolledBack")); return }
        try {
          if (metadata.length) await api.config.updateRouteRuleMetadata(metadata)
          toast.success(t("proxy.saved"))
        } catch (error) { toast.error(error instanceof Error ? error.message : String(error)) }
      },
      onError: (error) => toast.error(error.message),
    })
  }

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
      installInVisual={installInVisual}
      onRulesChange={section === "route" ? persistRules : undefined}
    />
  )
}
