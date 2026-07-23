import {
  ActivityIcon,
  AlertTriangleIcon,
  GlobeIcon,
  NetworkIcon,
  RouteIcon,
  ScrollTextIcon,
  Share2Icon,
  ShieldXIcon,
} from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { buildNodesHref } from "@/features/nodes/nodes-filter"
import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import { buildDNSHref } from "@/features/policy/dns-filter"
import { buildRouteHref } from "@/features/policy/route-rule-filter"
import { buildSubscriptionsHref } from "@/features/subscriptions/subscription-list"
import { cn } from "@/lib/utils"

interface Shortcut {
  labelKey: string
  to: string
  icon: typeof ActivityIcon
}

const shortcuts: Shortcut[] = [
  { labelKey: "dashboard.shortcutConnections", to: buildConnectionsHref(), icon: ActivityIcon },
  { labelKey: "dashboard.shortcutLogs", to: buildLogsHref(), icon: ScrollTextIcon },
  { labelKey: "dashboard.shortcutErrorLogs", to: buildLogsHref({ preset: "errors" }), icon: ScrollTextIcon },
  { labelKey: "dashboard.shortcutRejectLogs", to: buildLogsHref({ preset: "reject" }), icon: ShieldXIcon },
  { labelKey: "dashboard.shortcutFailedSubs", to: buildSubscriptionsHref({ status: "error" }), icon: AlertTriangleIcon },
  { labelKey: "dashboard.shortcutUnstableNodes", to: buildNodesHref({ stability: "unstable" }), icon: NetworkIcon },
  { labelKey: "dashboard.shortcutRejectRules", to: buildRouteHref({ action: "reject" }), icon: RouteIcon },
  { labelKey: "dashboard.shortcutDNSReject", to: buildDNSHref({ ruleAction: "reject" }), icon: ShieldXIcon },
  { labelKey: "dashboard.shortcutDNSLogs", to: buildLogsHref({ preset: "dns" }), icon: GlobeIcon },
  { labelKey: "dashboard.shortcutNodes", to: buildNodesHref(), icon: NetworkIcon },
  { labelKey: "dashboard.shortcutOutbounds", to: "/proxy/outbounds", icon: Share2Icon },
  { labelKey: "dashboard.shortcutRoute", to: buildRouteHref(), icon: RouteIcon },
  { labelKey: "dashboard.shortcutDNS", to: buildDNSHref(), icon: GlobeIcon },
]

export function OpsShortcutsCard() {
  const { t } = useTranslation()
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("dashboard.opsShortcutsTitle")}</CardTitle>
        <CardDescription>{t("dashboard.opsShortcutsDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-wrap gap-1.5 sm:gap-2">
        {shortcuts.map((item) => {
          const Icon = item.icon
          return (
            <Link
              key={item.labelKey}
              to={item.to}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8 gap-1.5")}
            >
              <Icon data-icon="inline-start" />
              {t(item.labelKey)}
            </Link>
          )
        })}
      </CardContent>
    </Card>
  )
}
