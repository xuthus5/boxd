import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { ConfirmAction } from "@/components/confirm-action"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatBytes } from "@/features/dashboard/format"
import type { ConnectionGroupStat } from "@/features/observability/connection-stats"

export function ConnectionGroupTable({
  groups,
  field,
  busy,
  onCloseGroup,
  emptyTitle,
  emptyDescription,
}: {
  groups: ConnectionGroupStat[]
  field: "outbound" | "rule"
  busy: boolean
  onCloseGroup: (field: "outbound" | "rule", key: string) => void
  emptyTitle: string
  emptyDescription: string
}) {
  const { t } = useTranslation()
  if (groups.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{emptyTitle}</EmptyTitle>
          <EmptyDescription>{emptyDescription}</EmptyDescription>
        </EmptyHeader>
      </Empty>
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
            <TableCell className="max-w-[14rem] truncate" title={group.key}>{group.key}</TableCell>
            <TableCell>{group.count}</TableCell>
            <TableCell>{formatBytes(group.upload)}</TableCell>
            <TableCell>{formatBytes(group.download)}</TableCell>
            <TableCell>
              <ConfirmAction
                trigger={
                  <Button size="sm" variant="destructive" disabled={busy || group.key === "—"}>
                    {t("observability.closeGroup")}
                  </Button>
                }
                title={t("observability.closeGroupTitle", { group: group.key })}
                description={t("observability.closeGroupDescription", { count: group.count, group: group.key })}
                confirmLabel={t("observability.confirmClose")}
                confirmVariant="destructive"
                onConfirm={() => onCloseGroup(field, group.key)}
              />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
