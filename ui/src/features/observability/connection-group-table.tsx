import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { ConfirmAction } from "@/components/confirm-action"
import { VirtualList } from "@/components/virtual-list"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatBytes } from "@/features/dashboard/format"
import { formatConnectionRatePair } from "@/features/observability/connection-rate"
import {
  isGroupClosing,
  type ClosingTarget,
} from "@/features/observability/connection-close"
import {
  buildConnectionsHref,
  type ConnectionFacetFilters,
} from "@/features/observability/connection-facets"
import type { ConnectionGroupStat } from "@/features/observability/connection-stats"
import { useIsMobile } from "@/hooks/use-mobile"
import { useVirtualWindow } from "@/hooks/use-virtual-window"
import { cn } from "@/lib/utils"

type GroupField = "outbound" | "rule" | "process"

const DESKTOP_GROUP_ROW_HEIGHT = 52
const MOBILE_GROUP_CARD_HEIGHT = 140

function groupRate(group: ConnectionGroupStat): string {
  return group.rateSamples === group.count
    ? formatConnectionRatePair(group.uploadRate, group.downloadRate)
    : "—"
}

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
  closingId,
  onCloseGroup,
  emptyTitle,
  emptyDescription,
  emptyActionLabel,
  onEmptyAction,
  baseFilters,
}: {
  groups: ConnectionGroupStat[]
  field: GroupField
  closingId: ClosingTarget
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
      <VirtualList
        className="max-h-[36rem]"
        items={groups}
        itemHeight={MOBILE_GROUP_CARD_HEIGHT}
        getKey={(group) => group.key}
        aria-label={t("observability.group")}
        renderItem={(group) => (
          <div className="pb-2">
            <Card size="sm" className="h-full overflow-hidden">
              <CardHeader className="min-w-0">
                <CardTitle className="truncate" title={group.key}>{group.key}</CardTitle>
                <CardDescription>{t("observability.count")}: {group.count}</CardDescription>
                <CardAction className="flex flex-wrap justify-end gap-1">
                  <GroupListLink field={field} groupKey={group.key} baseFilters={baseFilters} />
                  <CloseGroupButton
                    group={group}
                    field={field}
                    busy={isGroupClosing(closingId, field, group.key)}
                    onCloseGroup={onCloseGroup}
                  />
                </CardAction>
              </CardHeader>
              <CardContent className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                <span>{t("dashboard.upload")}: {formatBytes(group.upload)}</span>
                <span>{t("dashboard.download")}: {formatBytes(group.download)}</span>
                <span>{t("observability.rate")}: {groupRate(group)}</span>
              </CardContent>
            </Card>
          </div>
        )}
      />
    )
  }
  return (
    <GroupDesktopVirtualTable
      groups={groups}
      field={field}
      closingId={closingId}
      onCloseGroup={onCloseGroup}
      baseFilters={baseFilters}
    />
  )
}

function GroupDesktopVirtualTable({
  groups,
  field,
  closingId,
  onCloseGroup,
  baseFilters,
}: {
  groups: ConnectionGroupStat[]
  field: GroupField
  closingId: ClosingTarget
  onCloseGroup: (field: GroupField, key: string) => void
  baseFilters?: ConnectionFacetFilters
}) {
  const { t } = useTranslation()
  const { parentRef, onScroll, window } = useVirtualWindow({
    count: groups.length,
    itemHeight: DESKTOP_GROUP_ROW_HEIGHT,
  })
  const slice = groups.slice(window.startIndex, window.endIndex)
  return (
    <div
      ref={parentRef}
      className="max-h-[36rem] overflow-auto"
      onScroll={onScroll}
      role="region"
      aria-label={t("observability.group")}
    >
      <Table>
        <TableHeader className="sticky top-0 z-10 bg-background">
          <TableRow>
            <TableHead>{t("observability.group")}</TableHead>
            <TableHead>{t("observability.count")}</TableHead>
            <TableHead>{t("dashboard.upload")}</TableHead>
            <TableHead>{t("dashboard.download")}</TableHead>
            <TableHead>{t("observability.rate")}</TableHead>
            <TableHead>{t("common.actions")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {window.startIndex > 0 ? (
            <TableRow aria-hidden="true">
              <td colSpan={6} style={{ height: window.offsetTop, padding: 0, border: 0 }} />
            </TableRow>
          ) : null}
          {slice.map((group) => (
            <TableRow key={group.key} className="h-[52px]">
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
              <TableCell>{groupRate(group)}</TableCell>
              <TableCell>
                <div className="flex flex-wrap items-center gap-1">
                  <GroupListLink field={field} groupKey={group.key} baseFilters={baseFilters} />
                  <CloseGroupButton
                    group={group}
                    field={field}
                    busy={isGroupClosing(closingId, field, group.key)}
                    onCloseGroup={onCloseGroup}
                  />
                </div>
              </TableCell>
            </TableRow>
          ))}
          {window.endIndex < groups.length ? (
            <TableRow aria-hidden="true">
              <td
                colSpan={6}
                style={{
                  height: window.totalHeight - window.endIndex * DESKTOP_GROUP_ROW_HEIGHT,
                  padding: 0,
                  border: 0,
                }}
              />
            </TableRow>
          ) : null}
        </TableBody>
      </Table>
    </div>
  )
}
