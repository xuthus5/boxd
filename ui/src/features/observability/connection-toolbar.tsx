import { Columns3Icon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  CONNECTION_COLUMNS,
  type ConnectionColumnId,
} from "@/features/observability/connection-columns"
import type { ConnectionSortKey } from "@/features/observability/connection-export"
import type { ConnectionFacetOption } from "@/features/observability/connection-facets"

type Props = {
  query: string
  network: string
  protocol: string
  sort: ConnectionSortKey
  columns: ConnectionColumnId[]
  networkOptions: ConnectionFacetOption[]
  protocolOptions: ConnectionFacetOption[]
  sortOptions: { label: string; value: ConnectionSortKey }[]
  facetsActive: boolean
  filteredCount: number
  busy: boolean
  canExport: boolean
  paused: boolean
  onQueryChange: (value: string) => void
  onNetworkChange: (value: string) => void
  onProtocolChange: (value: string) => void
  onSortChange: (value: ConnectionSortKey) => void
  onToggleColumn: (id: ConnectionColumnId, enabled: boolean) => void
  onClearFacets: () => void
  onTogglePause: () => void
  onExport: () => void
  onCloseFiltered: () => void
}

export function ConnectionToolbar({
  query,
  network,
  protocol,
  sort,
  columns,
  networkOptions,
  protocolOptions,
  sortOptions,
  facetsActive,
  filteredCount,
  busy,
  canExport,
  paused,
  onQueryChange,
  onNetworkChange,
  onProtocolChange,
  onSortChange,
  onToggleColumn,
  onClearFacets,
  onTogglePause,
  onExport,
  onCloseFiltered,
}: Props) {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
      <label className="sr-only" htmlFor="connections-search">{t("observability.searchConnections")}</label>
      <Input
        id="connections-search"
        value={query}
        onChange={(event) => onQueryChange(event.target.value)}
        placeholder={t("observability.searchConnectionsPlaceholder")}
        className="sm:max-w-xs"
        aria-label={t("observability.searchConnections")}
      />
      <Select
        items={[{ label: t("observability.filterAll"), value: "__all__" }, ...networkOptions.map((option) => ({ label: `${option.value} (${option.count})`, value: option.value }))]}
        value={network || "__all__"}
        onValueChange={(value) => onNetworkChange(String(value) === "__all__" ? "" : String(value))}
      >
        <SelectTrigger aria-label={t("observability.filterNetwork")} className="w-full sm:w-36">
          <SelectValue placeholder={t("observability.filterNetwork")} />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value="__all__">{t("observability.filterAll")}</SelectItem>
            {networkOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>{option.value} ({option.count})</SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      <Select
        items={[{ label: t("observability.filterAll"), value: "__all__" }, ...protocolOptions.map((option) => ({ label: `${option.value} (${option.count})`, value: option.value }))]}
        value={protocol || "__all__"}
        onValueChange={(value) => onProtocolChange(String(value) === "__all__" ? "" : String(value))}
      >
        <SelectTrigger aria-label={t("observability.filterProtocol")} className="w-full sm:w-36">
          <SelectValue placeholder={t("observability.filterProtocol")} />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value="__all__">{t("observability.filterAll")}</SelectItem>
            {protocolOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>{option.value} ({option.count})</SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      {(network || protocol || query.trim()) ? (
        <Button variant="ghost" onClick={onClearFacets}>
          {t("observability.clearFacets")}
        </Button>
      ) : null}
      <Select items={sortOptions} value={sort} onValueChange={(value) => onSortChange(String(value) as ConnectionSortKey)}>
        <SelectTrigger aria-label={t("observability.sortConnections")} className="w-full sm:w-44">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {sortOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="outline" />}>
          <Columns3Icon data-icon="inline-start" />
          {t("observability.columns")}
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-52">
          <DropdownMenuGroup>
            <DropdownMenuLabel>{t("observability.columns")}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {CONNECTION_COLUMNS.map((column) => (
              <DropdownMenuCheckboxItem
                key={column.id}
                checked={columns.includes(column.id)}
                disabled={column.required}
                onCheckedChange={(checked) => onToggleColumn(column.id, checked === true)}
              >
                {t(column.labelKey)}
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <Button variant="outline" onClick={onTogglePause}>
        {paused ? t("observability.resume") : t("observability.pause")}
      </Button>
      <Button variant="outline" disabled={!canExport} onClick={onExport}>
        {t("observability.exportConnections")}
      </Button>
      {facetsActive && filteredCount > 0 ? (
        <ConfirmAction
          trigger={<Button variant="outline" disabled={busy}>{t("observability.closeFiltered")}</Button>}
          title={t("observability.closeFilteredTitle")}
          description={t("observability.closeFilteredDescription", { count: filteredCount })}
          confirmLabel={t("observability.confirmClose")}
          confirmVariant="destructive"
          onConfirm={onCloseFiltered}
        />
      ) : null}
    </div>
  )
}
