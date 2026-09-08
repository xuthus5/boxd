import { apiRequest, apiRequestEnvelope } from "@/lib/api/client"
import type { SingBoxConfig } from "@/lib/api/types"

export const setupModuleIDs = ["inbounds", "outbounds", "dns", "route", "rule_sets", "experimental"] as const
export type SetupModuleID = typeof setupModuleIDs[number]

export interface SetupModule {
  id: SetupModuleID
  state: "ready" | "missing" | "invalid"
  count: number
  dependencies: string[]
  message?: string
}

export interface SetupListener {
  tag: string
  type: string
  listen: string
  listen_port?: number
  address?: string[] | string
}

export interface SetupStatus {
  source_hash: string
  capabilities: {
    platform: string
    container: boolean
    tun_available: boolean
    tun_reason?: string
    ipv6_available: boolean
    ipv6_reason?: string
    default_listen: string
  }
  modules: SetupModule[]
  listeners: SetupListener[]
  kernel_running: boolean
  proxy_ready: boolean
  config_error?: string
}

export interface SetupInput {
  modules: SetupModuleID[]
  inbound_mode: "proxy" | "tun" | "preserve"
  ipv6: "auto" | "off" | "on"
  reset_invalid?: boolean
}

export interface SetupPreview {
  source_hash: string
  current_config: SingBoxConfig | null
  config: SingBoxConfig
  modules: SetupModuleID[]
  warnings: string[]
  will_restart: boolean
}

const post = (body: unknown): RequestInit => ({ method: "POST", body: JSON.stringify(body) })

export const setupAPI = {
  status: () => apiRequest<SetupStatus>("/api/config/setup"),
  preview: (input: SetupInput) => apiRequest<SetupPreview>("/api/config/setup/preview", post(input)),
  apply: (input: SetupInput & { source_hash: string }) => apiRequestEnvelope<{ modules: SetupModuleID[]; config_hash: string }>(
    "/api/config/setup/apply", post(input),
  ),
}
