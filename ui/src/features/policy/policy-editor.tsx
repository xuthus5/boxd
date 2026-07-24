import { useCallback, useEffect, useRef, useState } from "react"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ConfigDiffPanel } from "@/features/config/config-diff-panel"
import { diffConfig } from "@/features/config/config-diff"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useConfigQuery } from "@/features/config/config-hooks"
import { useConfigValidate } from "@/features/config/use-config-validate"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { reportConfigValidateError } from "@/features/config/report-config-validate-error"
import {
  isJsonObject,
  isPolicySectionStructureValid,
  type JsonObject,
  type PolicySection,
} from "@/features/policy/policy-form-model"
import type { JsonValue, RouteRuleMetadata } from "@/lib/api/types"
import { parsePolicyItemPath } from "@/features/policy/policy-path"

export interface PolicyVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
  onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
  onInstall?: () => void
  onGlobalSave?: (object: JsonObject) => void
  jumpPath?: string | null
  onJumpPathHandled?: () => void
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
  jumpPath?: string | null
  onJumpPathHandled?: () => void
  reportError?: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
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
  editorRef?: React.RefObject<JsonEditorHandle | null>
  activeTab: string
  onTabChange: (value: string | number | null) => void
  visualJumpPath?: string | null
  onVisualJumpPathHandled?: () => void
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
  editorRef,
  activeTab,
  onTabChange,
  visualJumpPath,
  onVisualJumpPathHandled,
}: PolicyEditorTabsProps) {
  const { t } = useTranslation()
  const structureValid = Boolean(object && isPolicySectionStructureValid(section, object))
  return (
    <Tabs value={activeTab} onValueChange={onTabChange} className="min-w-0">
      <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
        <TabsTrigger value="visual">{t("policy.visualTab")}</TabsTrigger>
        <TabsTrigger value="json">{t("policy.advancedTab")}</TabsTrigger>
      </TabsList>
      <TabsContent value="visual">
        {object && structureValid
          ? renderVisual({ object, revision, onChange, onFieldValidityChange, onRulesChange, onInstall, onGlobalSave, jumpPath: visualJumpPath, onJumpPathHandled: onVisualJumpPathHandled })
          : object ? <Alert variant="destructive">
            <AlertTitle>{t("policy.invalidStructureTitle")}</AlertTitle>
            <AlertDescription>{t("policy.invalidStructureDescription")}</AlertDescription>
          </Alert> : null}
      </TabsContent>
      <TabsContent value="json">
        <FieldGroup>
          <Field>
            <FieldLabel className="sr-only">{t("policy.jsonLabel")}</FieldLabel>
            <JsonEditor ref={editorRef} value={value} onChange={onJSONChange} ariaLabel={t("policy.jsonLabel")} />
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
  jumpPath,
  onJumpPathHandled,
  reportError,
  clearSaveError,
}: PolicyEditorProps) {
  const { t } = useTranslation()
  const configQuery = useConfigQuery()
  const editor = usePolicyEditorState(initialSection)
  const editorRef = useRef<JsonEditorHandle>(null)
  const [activeTab, setActiveTab] = useState("visual")
  const [visualJumpPath, setVisualJumpPath] = useState<string | null>(null)
  const structureValid = Boolean(editor.object && isPolicySectionStructureValid(section, editor.object))
  const initialObject = isJsonObject(initialSection) ? initialSection : {}
  const diffItems = editor.object ? diffConfig(initialObject, editor.object) : []
  const revealJSON = useCallback((path: string) => {
    setActiveTab("json")
    const candidates = [path]
    if (path.startsWith(`${section}.`)) candidates.push(path.slice(section.length + 1))
    if (path.startsWith(`${section}[`)) candidates.push(path.slice(section.length))
    if (!path.startsWith(section)) candidates.push(`${section}.${path.replace(/^\./, "")}`)
    const tryReveal = () => candidates.some((candidate) => editorRef.current?.revealPath(candidate) ?? false)
    if (tryReveal()) return true
    window.setTimeout(() => {
      if (!tryReveal()) toast.message(t("config.pathNotFound", { path }))
    }, 50)
    return true
  }, [section, t])
  const reveal = useCallback((path: string) => {
    if ((section === "route" || section === "dns") && parsePolicyItemPath(path, section)) {
      setActiveTab("visual")
      setVisualJumpPath(path)
      return true
    }
    return revealJSON(path)
  }, [revealJSON, section])
  useConfigPathReveal((path) => reveal(path), { section })
  useEffect(() => {
    if (!jumpPath) return
    reveal(jumpPath)
    onJumpPathHandled?.()
  }, [jumpPath, onJumpPathHandled, reveal])
  const buildConfig = useCallback(() => {
    if (!editor.object || !configQuery.data) return null
    return { ...configQuery.data, [section]: editor.object }
  }, [configQuery.data, editor.object, section])
  const { validating, validate } = useConfigValidate({
    buildConfig,
    reportError: reportError ?? reportConfigValidateError,
    clearSaveError,
    onReportedError: (err) => { if (err.path) reveal(err.path) },
  })
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
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle role="heading" aria-level={1} className="truncate">{title}</CardTitle>
        <CardDescription>{t("policy.description")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
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
          editorRef={editorRef}
          activeTab={activeTab}
          onTabChange={(value) => setActiveTab(String(value || "visual"))}
          visualJumpPath={visualJumpPath}
          onVisualJumpPathHandled={() => setVisualJumpPath(null)}
        />
      </CardContent>
      <CardFooter className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center sm:justify-between">
        {!installInVisual ? (
          <div data-testid="policy-diff-summary">
            <ConfigDiffPanel items={diffItems} onSelectPath={reveal} />
          </div>
        ) : <span />}
        <div className="flex flex-wrap justify-end gap-2">
          {!installInVisual ? <Button variant="outline" size="sm" className="h-8" onClick={onInstall}>{installLabel}</Button> : null}
          <Button
            variant="outline"
            size="sm"
            className="h-8"
            disabled={!editor.object || !structureValid || editor.invalidFields.size > 0 || validating || !configQuery.data}
            onClick={() => { void validate() }}
          >
            {validating ? t("advanced.validating") : t("advanced.validate")}
          </Button>
          {!installInVisual ? <Button size="sm" className="h-8" disabled={!editor.object || !structureValid || editor.invalidFields.size > 0 || validating} onClick={savePolicy}>
            {t("policy.save")}
          </Button> : null}
        </div>
      </CardFooter>
    </Card>
  )
}

