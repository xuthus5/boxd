import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { buildNodesHref, type ProblemNodePreview } from "@/features/nodes/nodes-filter"
import { formatLatency } from "@/features/nodes/node-format"
import { cn } from "@/lib/utils"

function stabilityLabelKey(stability: ProblemNodePreview["stability"]) {
  return stability === "failed" ? "nodes.healthFailed" : "nodes.healthPoor"
}

export function ProblemNodesPreview({
  items,
  total,
}: {
  items: readonly ProblemNodePreview[]
  total: number
}) {
  const { t } = useTranslation()
  if (total <= 0 || items.length === 0) return null
  const remaining = Math.max(0, total - items.length)
  return (
    <div className="flex w-full flex-col gap-2" data-slot="problem-nodes-preview">
      <ul className="flex flex-col gap-1.5">
        {items.map((item) => {
          const href = buildNodesHref({ query: item.tag, stability: item.stability })
          const toneClass = item.stability === "failed"
            ? "border-destructive/30 bg-destructive/5"
            : "border-orange-500/30 bg-orange-500/5"
          return (
            <li
              key={`${item.stability}-${item.tag}`}
              className={cn(
                "flex min-w-0 flex-col gap-1 rounded-md border px-2.5 py-1.5 sm:flex-row sm:items-center sm:justify-between",
                toneClass,
              )}
            >
              <div className="min-w-0">
                <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                  <Link
                    to={href}
                    className="truncate text-sm font-medium underline-offset-4 hover:underline"
                    title={item.tag}
                  >
                    {item.tag}
                  </Link>
                  <Badge variant="outline" className="font-mono text-[10px]">{item.type}</Badge>
                  <Badge
                    variant={item.stability === "failed" ? "destructive" : "outline"}
                    className="text-[10px]"
                  >
                    {t(stabilityLabelKey(item.stability))}
                  </Badge>
                </div>
                <p className="line-clamp-1 text-xs text-muted-foreground">
                  {t("nodes.healthSummary", {
                    percent: item.percent,
                    success: item.success,
                    total: item.count,
                  })}
                  {item.latest !== undefined ? ` · ${formatLatency(item.latest)}` : ""}
                </p>
              </div>
              <Link
                to={href}
                className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8 shrink-0")}
                aria-label={`${t("dashboard.openProblemNode")}: ${item.tag}`}
              >
                {t("dashboard.openProblemNode")}
              </Link>
            </li>
          )
        })}
      </ul>
      {remaining > 0 ? (
        <p className="text-xs text-muted-foreground">
          {t("dashboard.problemNodesMore", { count: remaining })}
        </p>
      ) : null}
    </div>
  )
}
