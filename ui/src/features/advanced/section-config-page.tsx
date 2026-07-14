import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import type { JsonValue } from "@/lib/api/types"

function SectionEditor({ initial, onSave }: { initial: JsonValue; onSave: (value: JsonValue) => void }) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(initial, null, 2))
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("advanced.advancedJSON")}</FieldLabel><JsonEditor value={value} onChange={setValue} ariaLabel={t("advanced.advancedJSON")} /></Field><Field><Button disabled={!isValidJSON(value)} onClick={() => onSave(JSON.parse(value) as JsonValue)}>{t("advanced.save")}</Button></Field></FieldGroup>
}

export function SectionConfigPage({ section, title, description }: { section: string; title: string; description: string }) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert>
  const persist = (value: JsonValue) => save.mutate({ ...query.data!, [section]: value }, { onSuccess: (response) => response.status === "rolled_back" ? toast.error(t("advanced.rolledBack")) : toast.success(t("advanced.saved")), onError: (error) => toast.error(error.message) })
  return <Card><CardHeader><CardTitle role="heading" aria-level={1}>{title}</CardTitle><CardDescription>{description}</CardDescription></CardHeader><CardContent><SectionEditor key={JSON.stringify(query.data?.[section] ?? (section === "endpoints" ? [] : {}))} initial={query.data?.[section] ?? (section === "endpoints" ? [] : {})} onSave={persist} /></CardContent></Card>
}
