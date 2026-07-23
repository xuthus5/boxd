import type { JsonValue, ServiceStatus, SingBoxConfig, Subscription } from "@/lib/api/types"

export type SetupStepId =
  | "kernel"
  | "inbounds"
  | "outbounds"
  | "subscriptions"
  | "route"
  | "clashApi"

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
    if (!tag) return false
    if (type === "selector" || type === "urltest") {
      const members = Array.isArray(item.outbounds) ? item.outbounds : []
      return members.length > 0
    }
    return !["direct", "block", "dns", "blackhole"].includes(type)
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
  const clash = isObject(experimental?.clash_api) ? experimental.clash_api : undefined
  const controller = typeof clash?.external_controller === "string" ? clash.external_controller.trim() : ""
  return controller.length > 0
}

export function hasSubscriptions(subscriptions: Subscription[] | undefined) {
  return Array.isArray(subscriptions) && subscriptions.length > 0
}

export function buildSetupSteps(input: {
  status?: ServiceStatus
  config?: SingBoxConfig
  subscriptions?: Subscription[]
}): SetupStep[] {
  return [
    { id: "kernel", done: Boolean(input.status?.running), href: "/dashboard" },
    { id: "inbounds", done: hasLocalInbound(input.config), href: "/proxy/inbounds" },
    { id: "outbounds", done: hasProxyOutbound(input.config), href: "/proxy/outbounds" },
    { id: "subscriptions", done: hasSubscriptions(input.subscriptions), href: "/subscriptions" },
    { id: "route", done: hasRouteRules(input.config), href: "/policy/route" },
    { id: "clashApi", done: hasClashAPI(input.config), href: "/advanced/experimental" },
  ]
}

export function setupProgress(steps: SetupStep[]) {
  const done = steps.filter((step) => step.done).length
  return { done, total: steps.length, complete: done === steps.length }
}
