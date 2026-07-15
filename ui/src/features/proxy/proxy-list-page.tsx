import { PlusIcon, Trash2Icon, WandSparklesIcon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { ProxyEditorDialog } from "@/features/proxy/proxy-editor-dialog"
import { InboundCard } from "@/features/proxy/inbound-card"
import { api } from "@/lib/api/endpoints"
import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>
interface Editing { index: number; item: JsonObject }

function objects(value: JsonValue | undefined) {
  return Array.isArray(value) ? value.filter((item): item is JsonObject => Boolean(item && typeof item === "object" && !Array.isArray(item))) : []
}

function InboundCards({ items, onEdit, onDelete }: { items: JsonObject[]; onEdit: (index: number) => void; onDelete: (index: number) => void }) {
  return <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{items.map((item, index) => (
    <InboundCard key={`${String(item.tag)}-${index}`} item={item} onEdit={() => onEdit(index)} onDelete={() => onDelete(index)} />
  ))}</div>
}

export function ProxyListPage({ configKey, title, addLabel }: { configKey: "inbounds" | "outbounds"; title: string; addLabel: string }) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const [editing, setEditing] = useState<Editing | null>(null)
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert>
  const items = objects(query.data?.[configKey])
  const persist = (nextItems: JsonObject[]) => save.mutate({ ...query.data!, [configKey]: nextItems }, {
    onSuccess: (response) => response.status === "rolled_back" ? toast.error(t("proxy.rolledBack")) : toast.success(t("proxy.saved")),
    onError: (error) => toast.error(error.message),
  })
  const saveItem = (item: JsonObject) => {
    const next = [...items]
    if (editing!.index < 0) next.push(item); else next[editing!.index] = item
    persist(next); setEditing(null)
  }
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between"><h1 className="text-2xl font-semibold">{title}</h1><div className="flex gap-2">
        {configKey === "outbounds" ? <Button variant="outline" onClick={() => api.config.installOutbounds().then(() => query.refetch()).catch((error: Error) => toast.error(error.message))}><WandSparklesIcon data-icon="inline-start" />{t("proxy.installDefaults")}</Button> : null}
        <Button onClick={() => setEditing({ index: -1, item: {} })}><PlusIcon data-icon="inline-start" />{addLabel}</Button>
      </div></div>
      {configKey === "inbounds" && items.length > 0 ? <InboundCards items={items} onEdit={(index) => setEditing({ index, item: items[index] })} onDelete={(index) => persist(items.filter((_, itemIndex) => itemIndex !== index))} /> : <Card><CardHeader><CardTitle>{title}{t("proxy.listSuffix")}</CardTitle><CardDescription>{t("proxy.description")}</CardDescription></CardHeader><CardContent>
        {items.length === 0 ? <Empty><EmptyHeader><EmptyTitle>{t("proxy.empty")}</EmptyTitle><EmptyDescription>{t("proxy.emptyDescription")}</EmptyDescription></EmptyHeader><EmptyContent><Button onClick={() => setEditing({ index: -1, item: {} })}>{addLabel}</Button></EmptyContent></Empty> : (
          <Table><TableHeader><TableRow><TableHead>{t("proxy.tag")}</TableHead><TableHead>{t("common.type")}</TableHead><TableHead>{t("common.address")}</TableHead><TableHead>{t("common.actions")}</TableHead></TableRow></TableHeader><TableBody>
            {items.map((item, index) => <TableRow key={`${String(item.tag)}-${index}`}><TableCell>{String(item.tag ?? "—")}</TableCell><TableCell>{String(item.type ?? "—")}</TableCell><TableCell>{String(item.server ?? item.listen ?? "—")}</TableCell><TableCell className="flex gap-2"><Button variant="outline" size="sm" onClick={() => setEditing({ index, item })}>{t("common.edit")}</Button><ConfirmAction trigger={<Button variant="destructive" size="sm"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>} title={t("proxy.deleteTitle")} description={t("proxy.deleteDescription", { tag: String(item.tag ?? "") })} confirmLabel={t("proxy.confirmDelete")} confirmVariant="destructive" onConfirm={() => persist(items.filter((_, itemIndex) => itemIndex !== index))} /></TableCell></TableRow>)}
          </TableBody></Table>
        )}
      </CardContent></Card>}
      {editing ? <ProxyEditorDialog key={`${editing.index}-${String(editing.item.tag)}`} title={editing.index < 0 ? addLabel : `${t("proxy.editPrefix")} ${String(editing.item.tag ?? "")}`} kind={configKey} item={editing.item} onClose={() => setEditing(null)} onSave={saveItem} /> : null}
    </div>
  )
}
