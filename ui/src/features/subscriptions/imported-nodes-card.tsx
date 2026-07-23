import { DownloadIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useId, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { NodeEditorDialog } from "@/features/nodes/node-editor-dialog"
import { NodeImportDialog } from "@/features/nodes/node-import-dialog"
import { api } from "@/lib/api/endpoints"
import type { Outbound } from "@/lib/api/types"

function ImportedNodeItem({ node, onEdit, onDelete }: { node: Outbound; onEdit: () => void; onDelete: () => void }) {
  const { t } = useTranslation()
  const endpoint = `${node.server ?? "—"}:${node.port ?? "—"}`
  return (
    <article aria-label={node.tag}>
      <Card size="sm">
        <CardHeader className="min-w-0 gap-1.5">
          <CardTitle className="truncate" title={node.tag}>{node.tag}</CardTitle>
          <CardDescription className="truncate" title={endpoint}>{endpoint}</CardDescription>
          <CardAction><Badge variant="outline">{node.type}</Badge></CardAction>
        </CardHeader>
        <CardFooter className="grid grid-cols-2 gap-1.5">
          <Button variant="outline" size="sm" className="h-8" onClick={onEdit}>
            <PencilIcon data-icon="inline-start" />
            {t("common.edit")}
          </Button>
          <ConfirmAction
            trigger={(
              <Button variant="destructive" size="sm" className="h-8">
                <Trash2Icon data-icon="inline-start" />
                {t("common.delete")}
              </Button>
            )}
            title={t("common.deleteTitle")}
            description={t("common.deleteDescription")}
            confirmLabel={t("common.confirmDelete")}
            confirmVariant="destructive"
            onConfirm={onDelete}
          />
        </CardFooter>
      </Card>
    </article>
  )
}

export function ImportedNodesCard() {
  const { t } = useTranslation()
  const titleId = useId()
  const client = useQueryClient()
  const query = useQuery({ queryKey: ["nodes"], queryFn: api.nodes.list })
  const [importing, setImporting] = useState(false)
  const [editing, setEditing] = useState<string | null>(null)
  const nodes = (query.data ?? []).filter((node) => node.source === "import")
  const refresh = () => Promise.all([
    client.invalidateQueries({ queryKey: ["nodes"] }),
    client.invalidateQueries({ queryKey: ["nodes", "results"] }),
  ])
  const remove = (tag: string) => api.nodes.delete(tag)
    .then(() => api.nodes.sync())
    .then(refresh)
    .then(() => toast.success(t("nodes.deleted")))
    .catch((error: Error) => toast.error(error.message))
  const content = query.isLoading
    ? <Skeleton className="h-32 w-full" />
    : query.error
      ? (
        <Alert variant="destructive">
          <AlertTitle>{t("common.loadFailed")}</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      )
      : nodes.length
        ? (
          <div className="grid gap-2 sm:grid-cols-2 sm:gap-3 xl:grid-cols-3">
            {nodes.map((node) => (
              <ImportedNodeItem
                key={node.tag}
                node={node}
                onEdit={() => setEditing(node.tag)}
                onDelete={() => remove(node.tag)}
              />
            ))}
          </div>
        )
        : (
          <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("nodes.empty")}</EmptyTitle>
              <EmptyDescription>{t("subscriptions.importedNodesEmpty")}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        )
  return (
    <>
      <section aria-labelledby={titleId} className="flex flex-col gap-2.5 sm:gap-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 id={titleId} className="text-base font-medium sm:text-lg">{t("subscriptions.importedNodes")}</h2>
            <p className="text-xs text-muted-foreground sm:text-sm">{t("subscriptions.importedNodesDescription")}</p>
          </div>
          <Button size="sm" className="h-8 shrink-0" onClick={() => setImporting(true)}>
            <DownloadIcon data-icon="inline-start" />
            {t("nodes.import")}
          </Button>
        </div>
        {content}
      </section>
      {importing ? (
        <NodeImportDialog
          onClose={() => setImporting(false)}
          onSaved={() => { setImporting(false); void refresh() }}
        />
      ) : null}
      {editing ? (
        <NodeEditorDialog
          tag={editing}
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); void refresh() }}
        />
      ) : null}
    </>
  )
}
