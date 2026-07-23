import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  parseLogSearchParams,
  toLogSearchParams,
  type LogSearchFilters,
} from "@/features/observability/log-filter-presets"
import { LogPanel } from "@/features/observability/log-panel"
import { api } from "@/lib/api/endpoints"

export function LogsPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseLogSearchParams(searchParams), [searchParams])
  const tab = filters.tab ?? "kernel"

  const onTabChange = (next: string | number | null) => {
    const value = String(next) === "application" ? "application" : "kernel"
    const nextFilters: LogSearchFilters = {
      tab: value === "application" ? "application" : undefined,
      query: filters.query,
      minimum: filters.minimum,
      preset: filters.preset,
    }
    setSearchParams(toLogSearchParams(nextFilters), { replace: true })
  }

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-semibold">{t("observability.logs")}</h1>
      <Tabs value={tab} onValueChange={onTabChange}>
        <TabsList>
          <TabsTrigger value="kernel">{t("observability.kernelLogs")}</TabsTrigger>
          <TabsTrigger value="application">{t("observability.appLogs")}</TabsTrigger>
        </TabsList>
        <TabsContent value="kernel" keepMounted>
          <LogPanel path={api.stats.paths.logs} title={t("observability.kernelLogs")} />
        </TabsContent>
        <TabsContent value="application" keepMounted>
          <LogPanel path={api.stats.paths.appLogs} title={t("observability.appLogs")} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
