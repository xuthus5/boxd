export type JsonPrimitive = boolean | number | string | null
export type JsonValue = JsonPrimitive | JsonValue[] | { [key: string]: JsonValue }
export type SingBoxConfig = Record<string, JsonValue>

export interface AuthResponse {
  token: string
  expires_at: string
}

export interface ServiceStatus {
  running: boolean
  uptime?: string
  memory?: number
  version?: string
}

export interface TrafficEvent {
  upload_bytes: number
  download_bytes: number
  timestamp: string
}

export type TrafficHistoryPoint = TrafficEvent

export interface MemoryStats {
  alloc: number
  total: number
  sys: number
  num_gc: number
  heap_inuse: number
  stack_inuse: number
}

export interface VersionInfo {
  version: string
  kernel_version: string
}

export interface LogEvent {
  level: string
  message: string
}

export interface Connection {
  id: number
  target: string
  outbound: string
  upload: number
  download: number
  start: string
}

export interface ConnectionEvent {
  active_connections: number
  list?: Connection[]
}

export interface Outbound {
  tag: string
  type: string
  server?: string
  port?: number
  raw?: JsonValue
  source?: "import" | "subscription"
  source_name?: string
}

export interface OutboundGroup {
  type: "selector" | "urltest" | string
  tag: string
  now: string
  all: string[]
}

export interface Subscription {
  id: string
  name: string
  url: string
  interval_min: number
  urltest?: URLTestOverrides
  last_updated: string
  error?: string
  outbounds?: Outbound[]
}

export interface URLTestDefaults {
  enabled: boolean
  url: string
  interval: string
  tolerance: number
}

export interface URLTestOverrides {
  enabled?: boolean
  url?: string
  interval?: string
  tolerance?: number
}

export interface ImportResult {
  tag: string
  type: string
  server: string
  port: number
  config: JsonValue
}

export interface TestResult {
  tag: string
  test_type: string
  success: boolean
  latency_ms?: number
  error?: string
}

export interface ApiErrorBody {
  code: string
  message: string
}

export interface APIEnvelope<T> {
  status: "ok" | "error" | "partial" | "rolled_back"
  data: T
  error: ApiErrorBody | null
  meta: JsonValue
}
