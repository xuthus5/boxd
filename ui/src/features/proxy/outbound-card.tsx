import { PencilIcon, Trash2Icon } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

function text(value: JsonValue | undefined) {
  return typeof value === "string" || typeof value === "number" ? String(value) : "—"
}

function stringList(value: JsonValue | undefined) {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : []
}

export function OutboundCard({ item, onEdit, onDelete }: { item: JsonObject; onEdit: () => void; onDelete: () => void }) {
  const { t } = useTranslation()
  const titleId = useId()
  const type = text(item.type)
  const tag = text(item.tag)
  const server = item.server === undefined ? null : `${text(item.server)}${item.server_port === undefined ? "" : `:${text(item.server_port)}`}`
  const members = stringList(item.outbounds)
  const transport = typeof item.transport === "object" && item.transport && !Array.isArray(item.transport) ? text(item.transport.type) : null
  const tls = typeof item.tls === "object" && item.tls && !Array.isArray(item.tls) && item.tls.enabled === true
  return <article aria-labelledby={titleId}><Card size="sm" className="h-full">
    <CardHeader className="min-w-0"><CardTitle><h2 id={titleId} className="truncate">{tag}</h2></CardTitle><CardDescription className="truncate">{server ?? t("proxy.outbound.groupOutbound")}</CardDescription><CardAction><Button variant="outline" size="xs" onClick={onEdit}><PencilIcon data-icon="inline-start" />{t("common.edit")}</Button></CardAction></CardHeader>
    <CardContent className="flex flex-col gap-3"><div className="flex flex-wrap gap-2"><Badge>{type}</Badge>{tls ? <Badge variant="secondary">TLS</Badge> : null}{transport && transport !== "—" ? <Badge variant="outline">{transport}</Badge> : null}</div>{members.length ? <p className="line-clamp-2 text-sm text-muted-foreground">{t("proxy.outbound.members", { members: members.join(", ") })}</p> : null}{item.detour ? <p className="truncate text-sm text-muted-foreground">{t("proxy.outbound.detourSummary", { detour: text(item.detour) })}</p> : null}</CardContent>
    <CardFooter className="justify-end"><ConfirmAction trigger={<Button variant="destructive" size="xs"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>} title={t("proxy.deleteTitle")} description={t("proxy.deleteDescription", { tag })} confirmLabel={t("proxy.confirmDelete")} confirmVariant="destructive" onConfirm={onDelete} /></CardFooter>
  </Card></article>
}
