import { useTranslation } from "react-i18next"
import type { RuleSetStatusItem } from "@/lib/api/types"

const sources = new Set(["rule_set", "inline", "route_default", "global_default", "default_outbound", "legacy_detour"])

export function RuleSetClientStatus({ statuses }: { statuses: RuleSetStatusItem[] }) {
  const { t } = useTranslation()
  return <div className="flex flex-col gap-1.5 text-xs text-muted-foreground">
    {statuses.map((status) => <div key={status.tag}>
      {statuses.length > 1 ? <p className="font-medium">{status.tag}</p> : null}
      {status.http_client || status.http_client_source ? <p>{t("kernel114.ruleSetHTTPClient", {
        client: status.http_client || t("kernel114.inlineHTTPClient"),
        source: t(`kernel114.httpClientSources.${sources.has(status.http_client_source ?? "") ? status.http_client_source : "unknown"}`),
      })}</p> : null}
      {status.kernel_managed || status.note_code === "kernel_managed"
        ? <p>{t("kernel114.ruleSetKernelManaged")}</p>
        : status.note ? <p>{status.note}</p> : null}
    </div>)}
  </div>
}
