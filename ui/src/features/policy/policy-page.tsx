import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import type { APIEnvelope, JsonValue } from "@/lib/api/types"

interface PolicyPageProps {
  section: "route" | "dns"
  title: string
  installLabel: string
  install: () => Promise<APIEnvelope<JsonValue>>
}

function PolicyEditor({ initial, onSave }: { initial: JsonValue; onSave: (value: JsonValue) => void }) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(initial, null, 2))
  return <FieldGroup><Field><FieldLabel className="sr-only">Policy JSON</FieldLabel><JsonEditor value={value} onChange={setValue} ariaLabel="Policy JSON" /></Field><Field><Button disabled={!isValidJSON(value)} onClick={() => onSave(JSON.parse(value) as JsonValue)}>{t("policy.save")}</Button></Field></FieldGroup>
}

export function PolicyPage({ section, title, installLabel, install }: PolicyPageProps) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert>
  const persist = (value: JsonValue) => save.mutate({ ...query.data!, [section]: value }, {
    onSuccess: (response) => response.status === "rolled_back" ? toast.error(t("policy.rolledBack")) : toast.success(t("proxy.saved")),
    onError: (error) => toast.error(error.message),
  })
  const installDefaults = () => install().then((response) => {
    if (response.status === "rolled_back") throw new Error(t("policy.rolledBack"))
    return query.refetch()
  }).then(() => toast.success(t("policy.installed"))).catch((error: Error) => toast.error(error.message))
  return (
    <Card>
      <CardHeader><CardTitle role="heading" aria-level={1}>{title}</CardTitle><CardDescription>{t("policy.description")}</CardDescription></CardHeader>
      <CardContent><PolicyEditor key={JSON.stringify(query.data?.[section] ?? {})} initial={query.data?.[section] ?? {}} onSave={persist} /></CardContent>
      <CardFooter><Button variant="outline" onClick={installDefaults}>{installLabel}</Button></CardFooter>
    </Card>
  )
}
