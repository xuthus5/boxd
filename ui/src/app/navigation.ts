import {
  ActivityIcon,
  BoxIcon,
  BracesIcon,
  CircleGaugeIcon,
  FlaskConicalIcon,
  GlobeIcon,
  ListTreeIcon,
  NetworkIcon,
  RadioTowerIcon,
  RouteIcon,
  ScrollTextIcon,
  SettingsIcon,
  Share2Icon,
} from "lucide-react"
import type { ComponentType } from "react"

export interface NavigationItem {
  label: string
  to: string
  icon: ComponentType
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
    { label: "nav.endpoints", to: "/advanced/endpoints", icon: BoxIcon },
    { label: "nav.experimental", to: "/advanced/experimental", icon: FlaskConicalIcon },
    { label: "nav.rawConfig", to: "/advanced/raw", icon: BracesIcon },
  ] },
]

export const footerItems: NavigationItem[] = [
  { label: "nav.settings", to: "/settings", icon: SettingsIcon },
]
