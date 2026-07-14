import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useTranslation } from "react-i18next"
import { LogPanel } from "@/features/observability/log-panel"
import { api } from "@/lib/api/endpoints"

export function LogsPage() {
  const { t } = useTranslation()
  return <div className="flex flex-col gap-4"><h1 className="text-2xl font-semibold">{t("observability.logs")}</h1><Tabs defaultValue="kernel"><TabsList><TabsTrigger value="kernel">{t("observability.kernelLogs")}</TabsTrigger><TabsTrigger value="application">{t("observability.appLogs")}</TabsTrigger></TabsList><TabsContent value="kernel" keepMounted><LogPanel path={api.stats.paths.logs} title={t("observability.kernelLogs")} /></TabsContent><TabsContent value="application" keepMounted><LogPanel path={api.stats.paths.appLogs} title={t("observability.appLogs")} /></TabsContent></Tabs></div>
}
