import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { PolicyPage } from "@/features/policy/policy-page"
import { useConfigQuery } from "@/features/config/config-hooks"
import { RouteVisualEditor } from "@/features/policy/route-visual-editor"
import { RouteModeCard } from "@/features/policy/route-mode-card"
import { api } from "@/lib/api/endpoints"
import type { RouteRuleMetadata } from "@/lib/api/types"
import { useTranslation } from "react-i18next"

export function RoutePage() {
  const { t } = useTranslation()
  const queryKey = ["route-rule-metadata"] as const
  const queryClient = useQueryClient()
  const query = useQuery({ queryKey, queryFn: api.config.getRouteRuleMetadata })
  const config = useConfigQuery()
  const service = useQuery({
    queryKey: ["service"],
    queryFn: api.service.status,
    refetchInterval: 1000,
    staleTime: 900,
    retry: false,
  })
  const [draft, setDraft] = useState<RouteRuleMetadata[] | null>(null)
  const metadata = draft ?? (Array.isArray(query.data) ? query.data : [])
  const updateMetadata = (next: RouteRuleMetadata[]) => setDraft(next)
  const saveMetadata = async () => {
    if (draft === null) return
    const saved = await api.config.updateRouteRuleMetadata(draft)
    queryClient.setQueryData(queryKey, saved)
    setDraft(null)
  }
  const refreshMetadata = async () => {
    await query.refetch()
    setDraft(null)
  }
  return <div className="flex flex-col gap-3 sm:gap-4">
    <RouteModeCard enabled={Boolean(service.data?.running)} />
    <PolicyPage section="route" title={t("pages.route")} installLabel={t("policy.installRoute")}
      renderVisual={(props) => <RouteVisualEditor {...props} outbounds={config.data?.outbounds} metadata={metadata} metadataLoading={query.isLoading}
        metadataError={query.error} onMetadataRetry={() => { void query.refetch() }} onMetadataChange={updateMetadata} />}
      afterSave={saveMetadata} afterInstall={refreshMetadata} installInVisual install={async () => {
      const rules = await api.config.installRuleSets()
      return rules.status === "rolled_back" ? rules : api.config.installRoute()
    }} />
  </div>
}
