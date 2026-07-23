import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  type ConfigDiffItem,
  formatConfigDiffSummary,
  previewJsonValue,
  summarizeConfigDiff,
} from "@/features/config/config-diff"
import { cn } from "@/lib/utils"

interface ConfigDiffPanelProps {
  items: readonly ConfigDiffItem[]
  onSelectPath?: (path: string) => void
  className?: string
  maxItems?: number
}

function kindVariant(kind: ConfigDiffItem["kind"]): "default" | "secondary" | "destructive" | "outline" {
  switch (kind) {
    case "added":
      return "default"
    case "removed":
      return "destructive"
    default:
      return "outline"
  }
}

function kindClass(kind: ConfigDiffItem["kind"]) {
  switch (kind) {
    case "added":
      return "border-transparent bg-emerald-600 text-white dark:bg-emerald-500"
    case "removed":
      return ""
    default:
      return "border-amber-500/40 text-amber-700 dark:text-amber-300"
  }
}

export function ConfigDiffPanel({ items, onSelectPath, className, maxItems = 24 }: ConfigDiffPanelProps) {
  const { t } = useTranslation()
  const labels = {
    added: t("config.diffAdded"),
    removed: t("config.diffRemoved"),
    changed: t("config.diffChanged"),
    none: t("config.diffNone"),
    more: t("config.diffMore"),
  }
  const summary = formatConfigDiffSummary(items, labels)
  const counts = summarizeConfigDiff(items)
  const visible = items.slice(0, maxItems)
  const rest = items.length - visible.length

  if (!items.length) {
    return <p className={cn("text-sm text-muted-foreground", className)} data-testid="config-diff-panel">{labels.none}</p>
  }

  return (
    <div className={cn("flex flex-col gap-2 rounded-lg border p-3", className)} data-testid="config-diff-panel">
      <div className="flex flex-wrap items-center gap-2">
        <p className="text-sm font-medium">{t("config.diffTitle")}</p>
        <Badge variant="outline">{t("config.diffCountChanged", { count: counts.changed })}</Badge>
        <Badge variant="secondary">{t("config.diffCountAdded", { count: counts.added })}</Badge>
        <Badge variant="destructive">{t("config.diffCountRemoved", { count: counts.removed })}</Badge>
      </div>
      <p className="text-xs text-muted-foreground" data-testid="config-diff-summary">{summary}</p>
      <ScrollArea className="max-h-56">
        <ul className="flex flex-col gap-2 pr-3">
          {visible.map((item) => (
            <li key={`${item.kind}:${item.path}`} className="rounded-md border bg-muted/20 p-2">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant={kindVariant(item.kind)} className={cn(kindClass(item.kind))}>
                  {item.kind === "added" ? labels.added : item.kind === "removed" ? labels.removed : labels.changed}
                </Badge>
                {onSelectPath ? (
                  <Button
                    type="button"
                    variant="link"
                    size="xs"
                    className="h-auto px-0 font-mono text-xs"
                    onClick={() => onSelectPath(item.path)}
                  >
                    {item.path}
                  </Button>
                ) : (
                  <code className="font-mono text-xs">{item.path}</code>
                )}
              </div>
              <div className="mt-1 grid gap-1 text-xs text-muted-foreground sm:grid-cols-2">
                {item.kind !== "added" ? (
                  <p className="min-w-0 break-all">
                    <span className="text-destructive">− </span>
                    {previewJsonValue(item.before)}
                  </p>
                ) : null}
                {item.kind !== "removed" ? (
                  <p className="min-w-0 break-all">
                    <span className="text-emerald-600 dark:text-emerald-400">+ </span>
                    {previewJsonValue(item.after)}
                  </p>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      </ScrollArea>
      {rest > 0 ? <p className="text-xs text-muted-foreground">{labels.more.replace("{{count}}", String(rest))}</p> : null}
    </div>
  )
}
