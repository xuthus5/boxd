import { PencilIcon, Trash2Icon } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

interface InboundCardProps {
  item: JsonObject
  onEdit: () => void
  onDelete: () => void
}

function display(value: JsonValue | undefined) {
  return typeof value === "string" || typeof value === "number" ? String(value) : "—"
}

export function InboundCard({ item, onEdit, onDelete }: InboundCardProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const tag = display(item.tag)
  const type = display(item.type)
  const address = display(item.listen ?? item.interface_name)
  const port = item.listen_port === undefined ? null : display(item.listen_port)
  return <article aria-labelledby={titleId}><Card size="sm" className="h-full">
    <CardHeader className="min-w-0"><CardTitle><h2 id={titleId} className="truncate">{tag}</h2></CardTitle>
      <CardDescription className="truncate">{address}</CardDescription><CardAction>
        <Button variant="outline" size="xs" onClick={onEdit}><PencilIcon data-icon="inline-start" />{t("common.edit")}</Button>
      </CardAction></CardHeader>
    <CardContent><div className="flex flex-wrap gap-2"><Badge>{type}</Badge>
      {port ? <Badge variant="secondary">{t("proxy.inbound.listenPort")}: {port}</Badge> : null}
    </div></CardContent>
    <CardFooter className="justify-end"><ConfirmAction trigger={<Button variant="destructive" size="xs"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>}
      title={t("proxy.deleteTitle")} description={t("proxy.deleteDescription", { tag })} confirmLabel={t("proxy.confirmDelete")}
      confirmVariant="destructive" onConfirm={onDelete} /></CardFooter>
  </Card></article>
}
