import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { ConfirmAction } from "@/components/confirm-action"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatBytes } from "@/features/dashboard/format"
import {
  buildConnectionsHref,
  type ConnectionFacetFilters,
} from "@/features/observability/connection-facets"
import type { ConnectionGroupStat } from "@/features/observability/connection-stats"
import { useIsMobile } from "@/hooks/use-mobile"
import { cn } from "@/lib/utils"

type GroupField = "outbound" | "rule" | "process"

function groupListHref(field: GroupField, key: string, base: ConnectionFacetFilters = {}) {
  return buildConnectionsHref({
    ...base,
    [field]: key,
    view: undefined,
  })
}

function CloseGroupButton({
  group,
  field,
  busy,
  onCloseGroup,
}: {
  group: ConnectionGroupStat
  field: GroupField
  busy: boolean
  onCloseGroup: (field: GroupField, key: string) => void
}) {
  const { t } = useTranslation()
  return (
    <ConfirmAction
      trigger={
        <Button size="sm" className="h-8" variant="destructive" disabled={busy || group.key === "—"}>
          {t("observability.closeGroup")}
        </Button>
      }
      title={t("observability.closeGroupTitle", { group: group.key })}
      description={t("observability.closeGroupDescription", { count: group.count, group: group.key })}
      confirmLabel={t("observability.confirmClose")}
      confirmVariant="destructive"
      onConfirm={() => onCloseGroup(field, group.key)}
    />
  )
}

function GroupListLink({ field, groupKey, baseFilters }: {
  field: GroupField
  groupKey: string
  baseFilters?: ConnectionFacetFilters
}) {
  const { t } = useTranslation()
  return (
    <Link
      to={groupListHref(field, groupKey, baseFilters)}
      aria-label={`${t("observability.viewGroupConnections")}: ${groupKey}`}
      className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
    >
      {t("observability.viewGroupConnections")}
    </Link>
  )
}

export function ConnectionGroupTable({
  groups,
  field,
  busy,
  onCloseGroup,
  emptyTitle,
  emptyDescription,
  emptyActionLabel,
  onEmptyAction,
  baseFilters,
}: {
  groups: ConnectionGroupStat[]
  field: GroupField
  busy: boolean
  onCloseGroup: (field: GroupField, key: string) => void
  emptyTitle: string
  emptyDescription: string
  emptyActionLabel?: string
  onEmptyAction?: () => void
  baseFilters?: ConnectionFacetFilters
}) {
  const { t } = useTranslation()
  const isMobile = useIsMobile()
  if (groups.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{emptyTitle}</EmptyTitle>
          <EmptyDescription>{emptyDescription}</EmptyDescription>
        </EmptyHeader>
        {emptyActionLabel && onEmptyAction ? (
          <EmptyContent>
            <Button type="button" variant="outline" onClick={onEmptyAction}>
              {emptyActionLabel}
            </Button>
          </EmptyContent>
        ) : null}
      </Empty>
    )
  }
  if (isMobile) {
    return (
      <div className="flex flex-col gap-2">
        {groups.map((group) => (
          <Card key={group.key} size="sm">
            <CardHeader className="min-w-0">
              <CardTitle className="truncate" title={group.key}>{group.key}</CardTitle>
              <CardDescription>{t("observability.count")}: {group.count}</CardDescription>
              <CardAction className="flex flex-wrap justify-end gap-1">
                <GroupListLink field={field} groupKey={group.key} baseFilters={baseFilters} />
                <CloseGroupButton group={group} field={field} busy={busy} onCloseGroup={onCloseGroup} />
              </CardAction>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
              <span>{t("dashboard.upload")}: {formatBytes(group.upload)}</span>
              <span>{t("dashboard.download")}: {formatBytes(group.download)}</span>
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("observability.group")}</TableHead>
          <TableHead>{t("observability.count")}</TableHead>
          <TableHead>{t("dashboard.upload")}</TableHead>
          <TableHead>{t("dashboard.download")}</TableHead>
          <TableHead>{t("common.actions")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {groups.map((group) => (
          <TableRow key={group.key}>
            <TableCell className="max-w-[14rem] truncate" title={group.key}>
              <Link
                to={groupListHref(field, group.key, baseFilters)}
                className="underline-offset-4 hover:underline"
                title={group.key}
              >
                {group.key}
              </Link>
            </TableCell>
            <TableCell>{group.count}</TableCell>
            <TableCell>{formatBytes(group.upload)}</TableCell>
            <TableCell>{formatBytes(group.download)}</TableCell>
            <TableCell>
              <div className="flex flex-wrap items-center gap-1">
                <GroupListLink field={field} groupKey={group.key} baseFilters={baseFilters} />
                <CloseGroupButton group={group} field={field} busy={busy} onCloseGroup={onCloseGroup} />
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
