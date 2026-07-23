import { ActivityIcon, PencilIcon, ScrollTextIcon, Trash2Icon } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { ConfirmAction } from "@/components/confirm-action"
import { CopyTagButton } from "@/features/proxy/copy-tag-button"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import type { JsonValue } from "@/lib/api/types"
import { cn } from "@/lib/utils"

type JsonObject = Record<string, JsonValue>

interface InboundCardProps {
  item: JsonObject
  onEdit: () => void
  onDelete: () => void
  onPatch?: (next: JsonObject) => void
  busy?: boolean
}

function display(value: JsonValue | undefined) {
  return typeof value === "string" || typeof value === "number" ? String(value) : "—"
}

function asBool(value: JsonValue | undefined) {
  return value === true
}

export function InboundCard({ item, onEdit, onDelete, onPatch, busy }: InboundCardProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const tag = display(item.tag)
  const type = display(item.type)
  const address = display(item.listen ?? item.interface_name)
  const port = item.listen_port === undefined ? null : display(item.listen_port)
  const supportsSystemProxy = type === "mixed" || type === "http"
  const isTun = type === "tun"
  const systemProxy = asBool(item.set_system_proxy)
  const autoRoute = asBool(item.auto_route)
  const hasTag = Boolean(tag && tag !== "—")

  return (
    <article aria-labelledby={titleId}>
      <Card size="sm" className="h-full">
        <CardHeader className="min-w-0 gap-1.5">
          <CardTitle className="min-w-0">
            <h2 id={titleId} className="truncate" title={tag}>{tag}</h2>
          </CardTitle>
          <CardDescription className="truncate">{address}</CardDescription>
          <CardAction className="flex gap-1">
            <CopyTagButton tag={tag} />
            <Button variant="outline" size="xs" onClick={onEdit}>
              <PencilIcon data-icon="inline-start" />
              {t("common.edit")}
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <div className="flex flex-wrap gap-1.5">
            <Badge>{type}</Badge>
            {port ? <Badge variant="secondary">{t("proxy.inbound.listenPort")}: {port}</Badge> : null}
            {supportsSystemProxy && systemProxy
              ? <Badge variant="outline">{t("proxy.inbound.systemProxyOn")}</Badge>
              : null}
            {isTun
              ? <Badge variant="outline">{autoRoute ? t("proxy.inbound.autoRouteOn") : t("proxy.inbound.autoRouteOff")}</Badge>
              : null}
          </div>
          {hasTag ? (
            <div className="flex flex-wrap gap-1.5">
              <Link
                to={buildConnectionsHref({ inbound: tag })}
                aria-label={`${t("proxy.viewConnections")}: ${tag}`}
                className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
              >
                <ActivityIcon data-icon="inline-start" />
                {t("proxy.viewConnections")}
              </Link>
              <Link
                to={buildLogsHref({ query: tag })}
                aria-label={`${t("proxy.viewLogs")}: ${tag}`}
                className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
              >
                <ScrollTextIcon data-icon="inline-start" />
                {t("proxy.viewLogs")}
              </Link>
            </div>
          ) : null}
          {supportsSystemProxy && onPatch ? (
            <div className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{t("proxy.inbound.setSystemProxy")}</p>
                <p className="truncate text-xs text-muted-foreground">{t("proxy.inbound.setSystemProxyQuick")}</p>
              </div>
              <Switch
                checked={systemProxy}
                disabled={busy}
                aria-label={t("proxy.inbound.setSystemProxy")}
                onCheckedChange={(checked) => onPatch({ ...item, set_system_proxy: checked === true })}
              />
            </div>
          ) : null}
          {isTun && onPatch ? (
            <div className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{t("proxy.inbound.autoRoute")}</p>
                <p className="truncate text-xs text-muted-foreground">{t("proxy.inbound.autoRouteQuick")}</p>
              </div>
              <Switch
                checked={autoRoute}
                disabled={busy}
                aria-label={t("proxy.inbound.autoRoute")}
                onCheckedChange={(checked) => onPatch({ ...item, auto_route: checked === true })}
              />
            </div>
          ) : null}
        </CardContent>
        <CardFooter className="justify-end">
          <ConfirmAction
            trigger={<Button variant="destructive" size="xs"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>}
            title={t("proxy.deleteTitle")}
            description={t("proxy.deleteDescription", { tag })}
            confirmLabel={t("proxy.confirmDelete")}
            confirmVariant="destructive"
            onConfirm={onDelete}
          />
        </CardFooter>
      </Card>
    </article>
  )
}
