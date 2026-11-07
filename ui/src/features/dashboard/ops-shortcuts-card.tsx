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
      <CardContent className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
        {shortcuts.map((item) => {
          const Icon = item.icon
          return (
            <Link
              key={item.labelKey}
              to={item.to}
              className={cn(buttonVariants({ variant: "ghost" }), "h-auto min-h-16 flex-col items-start justify-between gap-2 rounded-xl border border-border/70 bg-muted/15 p-3 text-xs font-medium transition-all hover:-translate-y-0.5 hover:border-primary/20 hover:bg-primary/5 hover:text-primary")}
            >
              <span className="flex size-7 items-center justify-center rounded-lg bg-background text-muted-foreground ring-1 ring-border/80 group-hover/button:text-primary">
                <Icon className="size-3.5" />
              </span>
              <span className="text-left leading-snug">{t(item.labelKey)}</span>
            </Link>
          )
        })}
      </CardContent>
    </Card>
  )
}
