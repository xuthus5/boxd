import { useMutation } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"
import { api } from "@/lib/api/endpoints"
import { triggerBrowserDownload } from "@/lib/api/client"

export function BackupExportCard() {
  const { t } = useTranslation()
  const exportBackup = useMutation({
    mutationFn: async () => {
      const file = await api.settings.exportBackup()
      triggerBrowserDownload(file.blob, file.filename)
      return file.filename
    },
    onSuccess: (filename) => toast.success(t("settings.backupExportSuccess", { filename })),
    onError: (error: Error) => reportSettingsRequestError(error, t, {
      scope: "backup-export",
      fallback: t("settings.backupExportFailed"),
    }),
  })

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("settings.backupExportTitle")}</CardTitle>
        <CardDescription>{t("settings.backupExportDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="text-xs text-muted-foreground">
        {t("settings.backupExportHint")}
      </CardContent>
      <CardFooter>
        <Button
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={exportBackup.isPending}
          onClick={() => exportBackup.mutate()}
        >
          {exportBackup.isPending ? t("settings.backupExporting") : t("settings.backupExportAction")}
        </Button>
      </CardFooter>
    </Card>
  )
}
