import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { diffConfig } from "@/features/config/config-diff"
import { ConfigDiffPanel } from "@/features/config/config-diff-panel"
import { setupMessage } from "@/features/setup/setup-messages"
import type { SetupPreview as Preview } from "@/lib/api/setup"

export function SetupPreview({ preview }: { preview: Preview }) {
  const { t } = useTranslation()
  return <div className="flex flex-col gap-3" aria-live="polite">
    <Alert>
      <AlertTitle>{t("setup.previewModules")}</AlertTitle>
      <AlertDescription className="flex flex-col gap-2">
        <div className="flex flex-wrap gap-1.5">{preview.modules.map((id) => <Badge key={id} variant="outline">{t(`setup.${id}`)}</Badge>)}</div>
        <p>{t(preview.will_restart ? "setup.previewRestart" : "setup.previewStopped")}</p>
        {preview.warnings?.length ? <ul className="list-disc pl-4">{preview.warnings.map((warning, index) => <li key={`${index}-${warning}`}>{setupMessage(warning, t)}</li>)}</ul> : null}
      </AlertDescription>
    </Alert>
    <ConfigDiffPanel items={diffConfig(preview.current_config, preview.config)} />
  </div>
}
