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

type FacetSelectProps = {
  label: string
  value: string
  options: ConnectionFacetOption[]
  allLabel: string
  onChange: (value: string) => void
}

function FacetSelect({ label, value, options, allLabel, onChange }: FacetSelectProps) {
  const items = [{ label: allLabel, value: "__all__" }, ...options.map((option) => ({
    label: `${option.value} (${option.count})`,
    value: option.value,
  }))]
  return (
    <Select
      items={items}
      value={value || "__all__"}
      onValueChange={(next) => onChange(String(next) === "__all__" ? "" : String(next))}
    >
      <SelectTrigger aria-label={label} className="w-full sm:w-36">
        <SelectValue placeholder={label} />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectItem value="__all__">{allLabel}</SelectItem>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>{option.value} ({option.count})</SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  )
}

type Props = {
  query: string
  network: string
  protocol: string
  outbound: string
  rule: string
  sort: ConnectionSortKey
  columns: ConnectionColumnId[]
  networkOptions: ConnectionFacetOption[]
  protocolOptions: ConnectionFacetOption[]
  outboundOptions: ConnectionFacetOption[]
  ruleOptions: ConnectionFacetOption[]
  sortOptions: { label: string; value: ConnectionSortKey }[]
  facetsActive: boolean
  filteredCount: number
  busy: boolean
  canExport: boolean
  paused: boolean
  onQueryChange: (value: string) => void
  onNetworkChange: (value: string) => void
  onProtocolChange: (value: string) => void
  onOutboundChange: (value: string) => void
  onRuleChange: (value: string) => void
  onSortChange: (value: ConnectionSortKey) => void
  onToggleColumn: (id: ConnectionColumnId, enabled: boolean) => void
  onClearFacets: () => void
  onTogglePause: () => void
  onExport: () => void
  onCloseFiltered: () => void
}

export function ConnectionToolbar(props: Props) {
  const { t } = useTranslation()
  const allLabel = t("observability.filterAll")
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
      <label className="sr-only" htmlFor="connections-search">{t("observability.searchConnections")}</label>
      <Input
        id="connections-search"
        value={props.query}
        onChange={(event) => props.onQueryChange(event.target.value)}
        placeholder={t("observability.searchConnectionsPlaceholder")}
        className="sm:max-w-xs"
        aria-label={t("observability.searchConnections")}
      />
      <FacetSelect label={t("observability.filterNetwork")} value={props.network} options={props.networkOptions} allLabel={allLabel} onChange={props.onNetworkChange} />
      <FacetSelect label={t("observability.filterProtocol")} value={props.protocol} options={props.protocolOptions} allLabel={allLabel} onChange={props.onProtocolChange} />
      <FacetSelect label={t("observability.filterOutbound")} value={props.outbound} options={props.outboundOptions} allLabel={allLabel} onChange={props.onOutboundChange} />
      <FacetSelect label={t("observability.filterRule")} value={props.rule} options={props.ruleOptions} allLabel={allLabel} onChange={props.onRuleChange} />
      {props.facetsActive ? (
        <Button variant="ghost" onClick={props.onClearFacets}>{t("observability.clearFacets")}</Button>
      ) : null}
      <Select items={props.sortOptions} value={props.sort} onValueChange={(value) => props.onSortChange(String(value) as ConnectionSortKey)}>
        <SelectTrigger aria-label={t("observability.sortConnections")} className="w-full sm:w-44">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {props.sortOptions.map((option) => (
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
                checked={props.columns.includes(column.id)}
                disabled={column.required}
                onCheckedChange={(checked) => props.onToggleColumn(column.id, checked === true)}
              >
                {t(column.labelKey)}
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <Button variant="outline" onClick={props.onTogglePause}>
        {props.paused ? t("observability.resume") : t("observability.pause")}
      </Button>
      <Button variant="outline" disabled={!props.canExport} onClick={props.onExport}>
        {t("observability.exportConnections")}
      </Button>
      {props.facetsActive && props.filteredCount > 0 ? (
        <ConfirmAction
          trigger={<Button variant="outline" disabled={props.busy}>{t("observability.closeFiltered")}</Button>}
          title={t("observability.closeFilteredTitle")}
          description={t("observability.closeFilteredDescription", { count: props.filteredCount })}
          confirmLabel={t("observability.confirmClose")}
          confirmVariant="destructive"
          onConfirm={props.onCloseFiltered}
        />
      ) : null}
    </div>
  )
}
