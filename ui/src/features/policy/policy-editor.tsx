import { useState } from "react"
import { useTranslation } from "react-i18next"

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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ConfigDiffPanel } from "@/features/config/config-diff-panel"
import { diffConfig } from "@/features/config/config-diff"
import { JsonEditor } from "@/features/config/json-editor"
import {
  isJsonObject,
  isPolicySectionStructureValid,
  type JsonObject,
  type PolicySection,
} from "@/features/policy/policy-form-model"
import type { JsonValue, RouteRuleMetadata } from "@/lib/api/types"

export interface PolicyVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
  onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
  onInstall?: () => void
  onGlobalSave?: (object: JsonObject) => void
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
  onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
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
  onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
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

export function PolicyEditor({
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
  const initialObject = isJsonObject(initialSection) ? initialSection : {}
  const diffItems = editor.object ? diffConfig(initialObject, editor.object) : []
  const savePolicy = () => {
    if (editor.object) onSave(editor.object)
  }
  /* c8 ignore next 3 */
  const saveGlobal = (object: JsonObject) => {
    const initial = isJsonObject(initialSection) ? initialSection : {}
    const preserved: JsonObject = section === "route"
      ? { ...(initial.rules === undefined ? {} : { rules: initial.rules }), ...(initial.rule_set === undefined ? {} : { rule_set: initial.rule_set }) }
      : { ...(initial.servers === undefined ? {} : { servers: initial.servers }), ...(initial.rules === undefined ? {} : { rules: initial.rules }) }
    onSave({ ...object, ...preserved })
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
          onGlobalSave={section === "route" || section === "dns" ? saveGlobal : undefined}
        />
      </CardContent>
      <CardFooter className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center sm:justify-between">
        {!installInVisual ? <div data-testid="policy-diff-summary"><ConfigDiffPanel items={diffItems} /></div> : <span />}
        <div className="flex flex-wrap justify-end gap-2">
          {!installInVisual ? <Button variant="outline" onClick={onInstall}>{installLabel}</Button> : null}
          {!installInVisual ? <Button disabled={!editor.object || !structureValid || editor.invalidFields.size > 0} onClick={savePolicy}>
            {t("policy.save")}
          </Button> : null}
        </div>
      </CardFooter>
    </Card>
  )
}

