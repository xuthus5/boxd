import {
  ActivityIcon,
  BoxIcon,
  CircleGaugeIcon,
  GlobeIcon,
  HistoryIcon,
  ListTreeIcon,
  NetworkIcon,
  RadioTowerIcon,
  RouteIcon,
  ScrollTextIcon,
  ServerCogIcon,
  SettingsIcon,
  Share2Icon,
  ShieldCheckIcon,
  TerminalIcon,
} from "lucide-react"
import type { ComponentType } from "react"

export interface NavigationItem {
  label: string
  to: string
  icon: ComponentType
  match?: readonly string[]
}

export interface NavigationGroup {
  label: string
  items: NavigationItem[]
}

export const primaryItems: NavigationItem[] = [
  { label: "nav.dashboard", to: "/dashboard", icon: CircleGaugeIcon },
]

export const navigationGroups: NavigationGroup[] = [
  { label: "nav.proxy", items: [
    { label: "nav.inbounds", to: "/proxy/inbounds", icon: RadioTowerIcon },
    { label: "nav.outbounds", to: "/proxy/outbounds", icon: Share2Icon },
  ] },
  { label: "nav.policy", items: [
    { label: "nav.route", to: "/policy/route", icon: RouteIcon },
    { label: "nav.dns", to: "/policy/dns", icon: GlobeIcon },
  ] },
  { label: "nav.resources", items: [
    { label: "nav.nodes", to: "/nodes", icon: NetworkIcon },
    { label: "nav.subscriptions", to: "/subscriptions", icon: ListTreeIcon },
  ] },
  { label: "nav.observability", items: [
    { label: "nav.connections", to: "/observability/connections", icon: ActivityIcon },
    { label: "nav.logs", to: "/observability/logs", icon: ScrollTextIcon },
  ] },
  { label: "nav.advanced", items: [
    { label: "nav.endpointsAndServices", to: "/advanced/endpoints", icon: BoxIcon, match: ["/advanced/endpoints", "/advanced/services"] },
    { label: "nav.certificate", to: "/advanced/certificate", icon: ShieldCheckIcon, match: ["/advanced/certificate", "/advanced/certificate-providers"] },
    { label: "nav.sharedResources", to: "/advanced/network-namespaces", icon: NetworkIcon, match: ["/advanced/network-namespaces", "/advanced/http-clients"] },
    { label: "nav.kernelOptions", to: "/advanced/log", icon: ServerCogIcon, match: ["/advanced/log", "/advanced/ntp", "/advanced/experimental"] },
    { label: "nav.configManagement", to: "/advanced/history", icon: HistoryIcon, match: ["/advanced/history", "/advanced/raw"] },
  ] },
]

export function isNavigationItemActive(item: NavigationItem, pathname: string) {
  return (item.match ?? [item.to]).includes(pathname)
}

export const footerItems: NavigationItem[] = [
  { label: "nav.terminal", to: "/terminal", icon: TerminalIcon },
  { label: "nav.settings", to: "/settings", icon: SettingsIcon },
]
