import type { JsonValue, ServiceStatus, SingBoxConfig, Subscription } from "@/lib/api/types"
import type { SetupStatus } from "@/lib/api/setup"

export type SetupStepId =
  | "kernel"
  | "modules"
  | "nodes"
  | "access"

export interface SetupStep {
  id: SetupStepId
  done: boolean
  href: string
}

function isObject(value: JsonValue | undefined): value is Record<string, JsonValue> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

function objects(value: JsonValue | undefined) {
  return Array.isArray(value)
    ? value.filter((item): item is Record<string, JsonValue> => Boolean(item && typeof item === "object" && !Array.isArray(item)))
    : []
}

export function hasProxyOutbound(config: SingBoxConfig | undefined) {
  return objects(config?.outbounds).some((item) => {
    const type = String(item.type ?? "")
    const tag = String(item.tag ?? "")
    if (!tag || !type) return false
    return !["direct", "block", "dns", "blackhole", "selector", "urltest"].includes(type)
  })
}

export function hasLocalInbound(config: SingBoxConfig | undefined) {
  return objects(config?.inbounds).some((item) => {
    const type = String(item.type ?? "")
    return type === "mixed" || type === "http" || type === "socks" || type === "tun"
  })
}

export function hasRouteRules(config: SingBoxConfig | undefined) {
  const route = isObject(config?.route) ? config.route : undefined
  const rules = Array.isArray(route?.rules) ? route.rules : []
  return rules.length > 0
}

export function hasClashAPI(config: SingBoxConfig | undefined) {
  const experimental = isObject(config?.experimental) ? config.experimental : undefined
  return isObject(experimental?.clash_api)
}

export function hasSubscriptions(subscriptions: Subscription[] | undefined) {
  return Array.isArray(subscriptions) && subscriptions.some((item) => !item.error && Array.isArray(item.outbounds) && item.outbounds.length > 0)
}

function baseModulesReady(setup: SetupStatus | undefined, config: SingBoxConfig | undefined) {
  if (setup?.config_error) return false
  if (Array.isArray(setup?.modules)) {
    return ["outbounds", "dns", "route"].every((id) => setup.modules.some((module) => module.id === id && module.state === "ready"))
  }
  const dns = isObject(config?.dns) ? config.dns : undefined
  return objects(config?.outbounds).length > 0 && objects(dns?.servers).length > 0 && hasRouteRules(config)
}

function accessConfigured(setup: SetupStatus | undefined, config: SingBoxConfig | undefined) {
  if (Array.isArray(setup?.modules) && setup.modules.some((module) => module.id === "inbounds" && module.state === "invalid")) return false
  if (!Array.isArray(setup?.listeners)) return hasLocalInbound(config)
  return setup.listeners.some((listener) => listener.type === "tun"
    ? setup.capabilities?.tun_available === true
    : ["mixed", "http", "socks"].includes(listener.type) && Number(listener.listen_port) > 0)
}

export function buildSetupSteps(input: {
  status?: ServiceStatus
  config?: SingBoxConfig
  subscriptions?: Subscription[]
  setup?: SetupStatus
}): SetupStep[] {
  return [
    { id: "modules", done: baseModulesReady(input.setup, input.config), href: "/advanced/raw" },
    { id: "nodes", done: input.setup?.proxy_ready ?? (hasProxyOutbound(input.config) || hasSubscriptions(input.subscriptions)), href: "/subscriptions" },
    { id: "access", done: accessConfigured(input.setup, input.config), href: "/proxy/inbounds" },
    { id: "kernel", done: Boolean(input.status?.running ?? input.setup?.kernel_running), href: "/dashboard" },
  ]
}

export function setupProgress(steps: SetupStep[]) {
  const done = steps.filter((step) => step.done).length
  return { done, total: steps.length, complete: done === steps.length }
}
