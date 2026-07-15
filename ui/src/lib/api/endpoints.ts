import { apiRequest, apiRequestEnvelope } from "@/lib/api/client"
import type {
  AuthResponse,
  ConnectionEvent,
  ImportResult,
  JsonValue,
  MemoryStats,
  Outbound,
  OutboundGroup,
  ServiceStatus,
  SingBoxConfig,
  Subscription,
  TestResult,
  TrafficHistoryPoint,
  URLTestDefaults,
  URLTestOverrides,
  VersionInfo,
} from "@/lib/api/types"

const json = (method: string, body?: unknown): RequestInit => ({
  method,
  body: body === undefined ? undefined : JSON.stringify(body),
})
const segment = (value: string) => encodeURIComponent(value)

export interface LoginInput { username: string; password: string }
export interface NodeInput { tag: string; type: string; server: string; port: number; config: JsonValue }
export interface SubscriptionInput {
  name: string
  url: string
  interval_min: number
  urltest?: URLTestOverrides | null
}
export interface TestInput { tag: string; test_type: "tcp" | "http" | "icmp"; server: string; port: number }

export const api = {
  auth: {
    login: (input: LoginInput) => apiRequest<AuthResponse>("/api/auth/login", json("POST", input)),
    logout: () => apiRequest<void>("/api/auth/logout", json("POST")),
  },
  config: {
    get: () => apiRequest<SingBoxConfig>("/api/config/"),
    update: (config: SingBoxConfig) => apiRequestEnvelope<JsonValue>("/api/config/", json("PUT", config)),
    getRaw: () => apiRequest<SingBoxConfig>("/api/config/raw"),
    updateRaw: (config: SingBoxConfig) => apiRequestEnvelope<JsonValue>("/api/config/raw", json("PUT", config)),
    installDNS: () => apiRequestEnvelope<JsonValue>("/api/config/dns/defaults", json("POST")),
    installRuleSets: () => apiRequestEnvelope<JsonValue>("/api/config/rule-sets/defaults", json("POST")),
    installOutbounds: () => apiRequestEnvelope<JsonValue>("/api/config/outbounds/defaults", json("POST")),
    installRoute: () => apiRequestEnvelope<JsonValue>("/api/config/route/defaults", json("POST")),
  },
  service: {
    status: () => apiRequest<ServiceStatus>("/api/service/status"),
    start: () => apiRequest<void>("/api/service/start", json("POST")),
    stop: () => apiRequest<void>("/api/service/stop", json("POST")),
    restart: () => apiRequest<void>("/api/service/restart", json("POST")),
  },
  stats: {
    history: () => apiRequest<{ points: TrafficHistoryPoint[] }>("/api/stats/traffic/history"),
    closeAll: () => apiRequest<{ closed: number }>("/api/stats/connections", json("DELETE")),
    closeConnection: (id: string) => apiRequest<void>(`/api/stats/connections/${segment(id)}`, json("DELETE")),
    paths: {
      traffic: "/api/stats/traffic",
      logs: "/api/stats/logs",
      appLogs: "/api/stats/app-logs",
      connections: "/api/stats/connections",
    },
  },
  import: {
    link: (link: string) => apiRequest<ImportResult>("/api/import/link", json("POST", { link })),
    save: (input: NodeInput) => apiRequest<void>("/api/import/save", json("POST", input)),
  },
  nodes: {
    list: () => apiRequest<Outbound[]>("/api/nodes/"),
    get: (tag: string) => apiRequest<Outbound>(`/api/nodes/${segment(tag)}`),
    update: (tag: string, input: NodeInput) => apiRequest<void>(`/api/nodes/${segment(tag)}`, json("PUT", input)),
    delete: (tag: string) => apiRequest<void>(`/api/nodes/${segment(tag)}`, json("DELETE")),
    groups: () => apiRequest<{ groups: OutboundGroup[] }>("/api/nodes/groups"),
    delay: (tag: string, options?: { url?: string; timeout?: number }) => {
      const query = new URLSearchParams()
      if (options?.url) query.set("url", options.url)
      if (options?.timeout) query.set("timeout", String(options.timeout))
      const suffix = query.size ? `?${query}` : ""
      return apiRequest<JsonValue>(`/api/nodes/${segment(tag)}/delay${suffix}`)
    },
    test: (input: TestInput) => apiRequest<TestResult>("/api/nodes/test", json("POST", input)),
    testBatch: (items: TestInput[], concurrency = 8) => apiRequest<{ results: TestResult[] }>(
      "/api/nodes/test-batch",
      json("POST", { items, concurrency }),
    ),
    results: () => apiRequest<Record<string, Record<string, TestResult>>>("/api/nodes/test-results"),
    select: (group: string, tag: string) => apiRequest<{ selected: string }>(
      `/api/nodes/selectors/${segment(group)}/select`,
      json("POST", { tag }),
    ),
    urlTest: (group: string) => apiRequest<Record<string, number>>(
      `/api/nodes/groups/${segment(group)}/urltest`,
      json("POST"),
    ),
    sync: () => apiRequest<JsonValue>("/api/nodes/sync-config", json("POST")),
  },
  subscriptions: {
    list: () => apiRequest<Subscription[]>("/api/subscriptions/"),
    create: (input: SubscriptionInput) => apiRequest<Subscription>("/api/subscriptions/", json("POST", input)),
    get: (id: string) => apiRequest<Subscription>(`/api/subscriptions/${segment(id)}`),
    update: (id: string, input: SubscriptionInput) => apiRequest<void>(
      `/api/subscriptions/${segment(id)}`,
      json("PUT", input),
    ),
    delete: (id: string) => apiRequest<void>(`/api/subscriptions/${segment(id)}`, json("DELETE")),
    refresh: (id: string) => apiRequest<JsonValue>(`/api/subscriptions/${segment(id)}/refresh`, json("POST")),
    refreshAll: () => apiRequestEnvelope<JsonValue>("/api/subscriptions/refresh-all", json("POST")),
  },
  settings: {
    testURL: () => apiRequest<{ url: string }>("/api/settings/url-test"),
    setTestURL: (url: string) => apiRequest<{ url: string }>("/api/settings/url-test", json("PUT", { url })),
    urlTestDefaults: () => apiRequest<URLTestDefaults>("/api/settings/urltest-defaults"),
    setURLTestDefaults: (input: URLTestDefaults) => apiRequest<URLTestDefaults>(
      "/api/settings/urltest-defaults",
      json("PUT", input),
    ),
    autostart: () => apiRequest<{ enabled: boolean }>("/api/settings/kernel-autostart"),
    setAutostart: (enabled: boolean) => apiRequest<{ enabled: boolean }>(
      "/api/settings/kernel-autostart",
      json("PUT", { enabled }),
    ),
    jwt: () => apiRequest<{ masked: string; present: boolean; length: number }>("/api/settings/jwt-secret"),
    setJWT: (secret: string) => apiRequest<{ masked: string; length: number }>(
      "/api/settings/jwt-secret",
      json("PUT", { secret }),
    ),
    password: () => apiRequest<{ defaultPassword: boolean }>("/api/settings/password"),
    changePassword: (currentPassword: string, newPassword: string) => apiRequest<{ changed: boolean }>(
      "/api/settings/password",
      json("PUT", { currentPassword, newPassword }),
    ),
  },
  network: {
    interfaces: () => apiRequest<JsonValue>("/api/network/interfaces"),
  },
  runtime: {
    version: () => apiRequest<VersionInfo>("/api/runtime/version"),
    memory: () => apiRequest<MemoryStats>("/api/runtime/memory"),
    gc: () => apiRequest<void>("/api/runtime/gc", json("POST")),
    flushDNS: () => apiRequest<void>("/api/runtime/dns/flush", json("POST")),
    flushFakeIP: () => apiRequest<void>("/api/runtime/fakeip/flush", json("POST")),
  },
} satisfies Record<string, unknown>

export type ConnectionsSnapshot = ConnectionEvent
