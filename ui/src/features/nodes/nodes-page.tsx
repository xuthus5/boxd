import { DownloadIcon, GaugeIcon, PencilIcon, RefreshCcwIcon, Trash2Icon } from "lucide-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { NodeEditorDialog } from "@/features/nodes/node-editor-dialog"
import { NodeResultsCard } from "@/features/nodes/node-results-card"
import { RuntimeGroupsCard } from "@/features/nodes/runtime-groups-card"
import { api } from "@/lib/api/endpoints"
import type { ImportResult, Outbound } from "@/lib/api/types"

function ImportDialog({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [link, setLink] = useState("")
  const [preview, setPreview] = useState<ImportResult | null>(null)
  const parse = useMutation({ mutationFn: () => api.import.link(link), onSuccess: setPreview, onError: (error: Error) => toast.error(error.message) })
  const save = useMutation({ mutationFn: () => api.import.save({ tag: preview!.tag, type: preview!.type, server: preview!.server, port: preview!.port, config: preview!.config }), onSuccess: () => { toast.success(t("nodes.saved")); onSaved() }, onError: (error: Error) => toast.error(error.message) })
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}><DialogContent><DialogHeader><DialogTitle>{t("nodes.import")}</DialogTitle><DialogDescription>{t("nodes.importDescription")}</DialogDescription></DialogHeader>
      <FieldGroup><Field><FieldLabel htmlFor="node-link">{t("nodes.link")}</FieldLabel><Input id="node-link" value={link} onChange={(event) => setLink(event.target.value)} /></Field>{preview ? <Field><FieldLabel>{t("nodes.parseResult")}</FieldLabel><Card><CardHeader><CardTitle>{preview.tag}</CardTitle><CardDescription>{preview.type}</CardDescription></CardHeader><CardContent>{preview.server}:{preview.port}</CardContent></Card></Field> : null}</FieldGroup>
      <DialogFooter><Button variant="outline" disabled={!link || parse.isPending} onClick={() => parse.mutate()}>{t("nodes.parse")}</Button><Button disabled={!preview || save.isPending} onClick={() => save.mutate()}>{t("nodes.saveNode")}</Button></DialogFooter>
    </DialogContent></Dialog>
  )
}

type TestType = "tcp" | "http" | "icmp"

interface NodeRowProps { node: Outbound; testType: TestType; onDelete: () => void; onEdit: () => void; onTested: () => void }

function NodeRow({ node, testType, onDelete, onEdit, onTested }: NodeRowProps) {
  const { t } = useTranslation()
  const test = useMutation({ mutationFn: () => api.nodes.test({ tag: node.tag, test_type: testType, server: node.server ?? "", port: node.port ?? 0 }), onSuccess: (result) => { onTested(); if (result.success) toast.success(`${result.latency_ms?.toFixed(0)} ms`); else toast.error(result.error || t("nodes.testFailed")) }, onError: (error: Error) => toast.error(error.message) })
  const subscription = node.source === "subscription"
  return <TableRow><TableCell>{node.tag}</TableCell><TableCell>{node.type}</TableCell><TableCell>{node.server ?? "—"}:{node.port ?? "—"}</TableCell><TableCell><Badge variant="secondary">{subscription ? node.source_name || t("nodes.subscription") : t("nodes.imported")}</Badge></TableCell><TableCell className="flex gap-2"><Button variant="outline" size="sm" onClick={() => test.mutate()}><GaugeIcon data-icon="inline-start" />{t("nodes.test")}</Button><Button variant="outline" size="sm" disabled={subscription} onClick={onEdit}><PencilIcon data-icon="inline-start" />{t("common.edit")}</Button><ConfirmAction trigger={<Button variant="destructive" size="sm" disabled={subscription}><Trash2Icon data-icon="inline-start" />{subscription ? t("nodes.subscriptionNode") : t("common.delete")}</Button>} title={t("common.deleteTitle")} description={t("common.deleteDescription")} confirmLabel={t("common.confirmDelete")} confirmVariant="destructive" onConfirm={onDelete} /></TableCell></TableRow>
}

export function NodesPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const query = useQuery({ queryKey: ["nodes"], queryFn: api.nodes.list })
  const [importing, setImporting] = useState(false)
  const [editing, setEditing] = useState<string | null>(null)
  const [testType, setTestType] = useState<TestType>("http")
  const visibleTags = useMemo(() => new Set(query.data?.map((node) => node.tag) ?? []), [query.data])
  const refresh = () => client.invalidateQueries({ queryKey: ["nodes"] })
  const action = (request: Promise<unknown>, message: string) => request.then(() => refresh()).then(() => toast.success(message)).catch((error: Error) => toast.error(error.message))
  const removeNode = (tag: string) => action(api.nodes.delete(tag).then(() => client.invalidateQueries({ queryKey: ["nodes", "results"] })), t("nodes.deleted"))
  const batch = useMutation({
    mutationFn: () => api.nodes.testBatch((query.data ?? []).filter((node) => node.server && node.port).map((node) => ({ tag: node.tag, test_type: testType, server: node.server!, port: node.port! }))),
    onSuccess: () => { void client.invalidateQueries({ queryKey: ["nodes", "results"] }); toast.success(t("nodes.batchComplete")) },
    onError: (error: Error) => toast.error(error.message),
  })
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert>
  return (
    <div className="flex flex-col gap-4"><div className="flex items-center justify-between"><h1 className="text-2xl font-semibold">{t("nodes.title")}</h1><div className="flex gap-2"><ToggleGroup aria-label={t("nodes.testType")} value={[testType]} onValueChange={(value) => { if (value[0]) setTestType(value[0] as TestType) }} variant="outline"><ToggleGroupItem value="tcp">TCP</ToggleGroupItem><ToggleGroupItem value="http">HTTP</ToggleGroupItem><ToggleGroupItem value="icmp">ICMP</ToggleGroupItem></ToggleGroup><Button variant="outline" disabled={!query.data?.some((node) => node.server && node.port) || batch.isPending} onClick={() => batch.mutate()}><GaugeIcon data-icon="inline-start" />{t("nodes.batch")}</Button><Button variant="outline" onClick={() => action(api.nodes.sync(), t("nodes.synced"))}><RefreshCcwIcon data-icon="inline-start" />{t("nodes.sync")}</Button><Button onClick={() => setImporting(true)}><DownloadIcon data-icon="inline-start" />{t("nodes.import")}</Button></div></div>
      <Card><CardHeader><CardTitle>{t("nodes.list")}</CardTitle><CardDescription>{t("nodes.description")}</CardDescription></CardHeader><CardContent>{query.data?.length ? <Table><TableHeader><TableRow><TableHead>Tag</TableHead><TableHead>{t("common.type")}</TableHead><TableHead>{t("nodes.server")}</TableHead><TableHead>{t("nodes.source")}</TableHead><TableHead>{t("common.actions")}</TableHead></TableRow></TableHeader><TableBody>{query.data.map((node) => <NodeRow key={node.tag} node={node} testType={testType} onEdit={() => setEditing(node.tag)} onTested={() => client.invalidateQueries({ queryKey: ["nodes", "results"] })} onDelete={() => removeNode(node.tag)} />)}</TableBody></Table> : <Empty><EmptyHeader><EmptyTitle>{t("nodes.empty")}</EmptyTitle><EmptyDescription>{t("nodes.emptyDescription")}</EmptyDescription></EmptyHeader></Empty>}</CardContent></Card>
      <RuntimeGroupsCard /><NodeResultsCard visibleTags={visibleTags} />
      {importing ? <ImportDialog onClose={() => setImporting(false)} onSaved={() => { setImporting(false); void refresh() }} /> : null}
      {editing ? <NodeEditorDialog tag={editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); void refresh() }} /> : null}
    </div>
  )
}
