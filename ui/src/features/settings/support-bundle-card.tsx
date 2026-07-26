import { FileDownIcon, ShieldCheckIcon } from "lucide-react"
import { useMutation } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"
import {
  buildSupportBundleFilename,
  collectSupportBundle,
  countUnavailableSources,
  formatSupportBundle,
  type SupportBundleLoaders,
} from "@/features/settings/support-bundle"
import { api } from "@/lib/api/endpoints"
import { triggerBrowserDownload } from "@/lib/api/client"
import type { UIPreferences } from "@/lib/api/types"

const loaders: SupportBundleLoaders = {
  version: api.runtime.version,
  service: api.service.status,
  readiness: api.health.readiness,
  memory: api.runtime.memory,
  config_diagnostics: api.config.diagnostics,
  rule_sets: api.config.ruleSetsStatus,
  rule_set_auto_update: api.config.ruleSetsAutoUpdate,
  subscriptions: api.subscriptions.list,
  nodes: api.nodes.list,
  node_history: api.nodes.testHistory,
  apply_history: api.config.applyHistory,
  network: api.network.interfaces,
}

interface SupportBundleCardProps {
  preferences: UIPreferences
}

export function SupportBundleCard({ preferences }: SupportBundleCardProps) {
  const { t } = useTranslation()
  const exportSupportBundle = useMutation({
    mutationFn: async () => {
      const bundle = await collectSupportBundle(loaders, preferences)
      const filename = buildSupportBundleFilename()
      const blob = new Blob([formatSupportBundle(bundle)], { type: "application/json;charset=utf-8" })
      triggerBrowserDownload(blob, filename)
      return { filename, unavailable: countUnavailableSources(bundle.requests) }
    },
    onSuccess: ({ filename, unavailable }) => {
      const message = unavailable > 0
        ? t("settings.supportBundleExportPartial", { filename, count: unavailable })
        : t("settings.supportBundleExportSuccess", { filename })
      toast.success(message)
    },
    onError: (error: Error) => reportSettingsRequestError(error, t, {
      scope: "support-bundle-export",
      fallback: t("settings.supportBundleExportFailed"),
    }),
  })

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="flex items-center gap-2 truncate">
          <ShieldCheckIcon className="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
          <span className="truncate">{t("settings.supportBundleTitle")}</span>
        </CardTitle>
        <CardDescription>{t("settings.supportBundleDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="text-xs text-muted-foreground">
        {t("settings.supportBundleHint")}
      </CardContent>
      <CardFooter>
        <Button
          type="button"
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={exportSupportBundle.isPending}
          onClick={() => exportSupportBundle.mutate()}
        >
          <FileDownIcon data-icon="inline-start" />
          {exportSupportBundle.isPending ? t("settings.supportBundleExporting") : t("settings.supportBundleAction")}
        </Button>
      </CardFooter>
    </Card>
  )
}
