import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatBytes } from "@/features/dashboard/format"
import type { MemoryStats } from "@/lib/api/types"

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg border bg-muted/30 px-3 py-2">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="truncate text-sm font-semibold tabular-nums" title={value}>{value}</p>
    </div>
  )
}

export function RuntimeStatsCard({
  memory,
  panelVersion,
  kernelVersion,
}: {
  memory: MemoryStats
  panelVersion: string
  kernelVersion: string
}) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("dashboard.runtimeStatsTitle")}</CardTitle>
        <CardDescription>{t("dashboard.runtimeStatsDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-2 sm:grid-cols-2">
        <Stat label={t("dashboard.memoryAlloc")} value={formatBytes(memory.alloc)} />
        <Stat label={t("dashboard.memorySys")} value={formatBytes(memory.sys)} />
        <Stat label={t("dashboard.memoryHeap")} value={formatBytes(memory.heap_inuse)} />
        <Stat label={t("dashboard.memoryStack")} value={formatBytes(memory.stack_inuse)} />
        <Stat label={t("dashboard.memoryGC")} value={String(memory.num_gc)} />
        <Stat
          label={t("dashboard.memoryGoroutines")}
          value={String(memory.num_goroutine ?? "—")}
        />
        <Stat label={t("dashboard.kernelVersion")} value={kernelVersion || "—"} />
        <Stat label={t("dashboard.panelVersion")} value={panelVersion || "—"} />
      </CardContent>
    </Card>
  )
}
