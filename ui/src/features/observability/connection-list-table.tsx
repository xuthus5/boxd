import { useTranslation } from "react-i18next"

import { VirtualList } from "@/components/virtual-list"
import { Button } from "@/components/ui/button"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  CONNECTION_COLUMNS,
  connectionColumnVisible,
  type ConnectionColumnId,
} from "@/features/observability/connection-columns"
import {
  ConnectionDesktopRow,
  ConnectionMobileCard,
} from "@/features/observability/connection-list-rows"
import { useIsMobile } from "@/hooks/use-mobile"
import { useVirtualWindow } from "@/hooks/use-virtual-window"
import type { Connection } from "@/lib/api/types"

const DESKTOP_ROW_HEIGHT = 56
const MOBILE_CARD_HEIGHT = 176

export function ConnectionListTable({
  connections,
  columns,
  busy,
  onClose,
  emptyActionLabel,
  onEmptyAction,
}: {
  connections: Connection[]
  columns: readonly ConnectionColumnId[]
  busy: boolean
  onClose: (id: string) => void
  emptyActionLabel?: string
  onEmptyAction?: () => void
}) {
  const { t } = useTranslation()
  const isMobile = useIsMobile()
  if (connections.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{t("observability.noMatch")}</EmptyTitle>
          <EmptyDescription>{t("observability.noMatchDescription")}</EmptyDescription>
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
        items={connections}
        itemHeight={MOBILE_CARD_HEIGHT}
        getKey={(connection) => connection.id}
        aria-label={t("observability.connections")}
        renderItem={(connection) => (
          <div className="pb-2">
            <ConnectionMobileCard
              connection={connection}
              columns={columns}
              busy={busy}
              onClose={onClose}
            />
          </div>
        )}
      />
    )
  }
  return (
    <ConnectionDesktopVirtualTable
      connections={connections}
      columns={columns}
      busy={busy}
      onClose={onClose}
    />
  )
}

function ConnectionDesktopVirtualTable({
  connections,
  columns,
  busy,
  onClose,
}: {
  connections: Connection[]
  columns: readonly ConnectionColumnId[]
  busy: boolean
  onClose: (id: string) => void
}) {
  const { t } = useTranslation()
  const visible = CONNECTION_COLUMNS.filter((column) => connectionColumnVisible(columns, column.id))
  const { parentRef, onScroll, window } = useVirtualWindow({
    count: connections.length,
    itemHeight: DESKTOP_ROW_HEIGHT,
  })
  const slice = connections.slice(window.startIndex, window.endIndex)
  return (
    <div
      ref={parentRef}
      className="max-h-[36rem] overflow-auto"
      onScroll={onScroll}
      role="region"
      aria-label={t("observability.connections")}
    >
      <Table>
        <TableHeader className="sticky top-0 z-10 bg-background">
          <TableRow>
            {visible.map((column) => (
              <TableHead key={column.id}>{t(column.labelKey)}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {window.startIndex > 0 ? (
            <TableRow aria-hidden="true">
              <TableCellSpacer colSpan={visible.length} height={window.offsetTop} />
            </TableRow>
          ) : null}
          {slice.map((connection) => (
            <ConnectionDesktopRow
              key={connection.id}
              connection={connection}
              columns={columns}
              busy={busy}
              onClose={onClose}
            />
          ))}
          {window.endIndex < connections.length ? (
            <TableRow aria-hidden="true">
              <TableCellSpacer
                colSpan={visible.length}
                height={window.totalHeight - window.endIndex * DESKTOP_ROW_HEIGHT}
              />
            </TableRow>
          ) : null}
        </TableBody>
      </Table>
    </div>
  )
}

function TableCellSpacer({ colSpan, height }: { colSpan: number; height: number }) {
  return (
    <td colSpan={colSpan} style={{ height, padding: 0, border: 0 }} />
  )
}
