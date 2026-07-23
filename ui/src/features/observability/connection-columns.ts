/** Connection table column visibility helpers (localStorage-backed). */

export const CONNECTION_COLUMN_STORAGE_KEY = "boxd.connection-columns.v1"

export type ConnectionColumnId =
  | "target"
  | "source"
  | "network"
  | "inbound"
  | "outbound"
  | "rule"
  | "protocol"
  | "process"
  | "upload"
  | "download"
  | "duration"
  | "actions"

export type ConnectionColumnSpec = {
  id: ConnectionColumnId
  labelKey: string
  required?: boolean
  defaultVisible?: boolean
}

export const CONNECTION_COLUMNS: readonly ConnectionColumnSpec[] = [
  { id: "target", labelKey: "observability.target", required: true, defaultVisible: true },
  { id: "source", labelKey: "observability.source", defaultVisible: false },
  { id: "network", labelKey: "observability.network", defaultVisible: true },
  { id: "inbound", labelKey: "observability.inbound", defaultVisible: false },
  { id: "outbound", labelKey: "observability.outbound", required: true, defaultVisible: true },
  { id: "rule", labelKey: "observability.rule", defaultVisible: true },
  { id: "protocol", labelKey: "observability.protocol", defaultVisible: false },
  { id: "process", labelKey: "observability.process", defaultVisible: true },
  { id: "upload", labelKey: "dashboard.upload", defaultVisible: true },
  { id: "download", labelKey: "dashboard.download", defaultVisible: true },
  { id: "duration", labelKey: "observability.duration", defaultVisible: true },
  { id: "actions", labelKey: "common.actions", required: true, defaultVisible: true },
] as const

export function defaultConnectionColumns(): ConnectionColumnId[] {
  return CONNECTION_COLUMNS.filter((column) => column.required || column.defaultVisible).map((column) => column.id)
}

export function isConnectionColumnId(value: unknown): value is ConnectionColumnId {
  return typeof value === "string" && CONNECTION_COLUMNS.some((column) => column.id === value)
}

export function normalizeConnectionColumns(input: unknown): ConnectionColumnId[] {
  const selected = Array.isArray(input)
    ? input.filter(isConnectionColumnId)
    : defaultConnectionColumns()
  const set = new Set(selected)
  for (const column of CONNECTION_COLUMNS) {
    if (column.required) set.add(column.id)
  }
  return CONNECTION_COLUMNS.map((column) => column.id).filter((id) => set.has(id))
}

export function loadConnectionColumns(storage: Pick<Storage, "getItem"> | null | undefined = globalThis.localStorage): ConnectionColumnId[] {
  try {
    const raw = storage?.getItem(CONNECTION_COLUMN_STORAGE_KEY)
    if (!raw) return defaultConnectionColumns()
    return normalizeConnectionColumns(JSON.parse(raw) as unknown)
  } catch {
    return defaultConnectionColumns()
  }
}

export function saveConnectionColumns(
  columns: readonly ConnectionColumnId[],
  storage: Pick<Storage, "setItem"> | null | undefined = globalThis.localStorage,
): ConnectionColumnId[] {
  const next = normalizeConnectionColumns(columns)
  try {
    storage?.setItem(CONNECTION_COLUMN_STORAGE_KEY, JSON.stringify(next))
  } catch {
    // Ignore quota / private-mode write failures.
  }
  return next
}

export function toggleConnectionColumn(
  columns: readonly ConnectionColumnId[],
  id: ConnectionColumnId,
  enabled: boolean,
): ConnectionColumnId[] {
  const column = CONNECTION_COLUMNS.find((item) => item.id === id)
  if (!column || column.required) return normalizeConnectionColumns(columns)
  const set = new Set(columns)
  if (enabled) set.add(id)
  else set.delete(id)
  return normalizeConnectionColumns([...set])
}

export function connectionColumnVisible(columns: readonly ConnectionColumnId[], id: ConnectionColumnId): boolean {
  return columns.includes(id)
}
