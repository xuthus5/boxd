import { PolicyPage } from "@/features/policy/policy-page"
import { api } from "@/lib/api/endpoints"
import { useTranslation } from "react-i18next"

export function RoutePage() {
  const { t } = useTranslation()
  return <PolicyPage section="route" title={t("pages.route")} installLabel={t("policy.installRoute")} install={async () => {
    const rules = await api.config.installRuleSets()
    return rules.status === "rolled_back" ? rules : api.config.installRoute()
  }} />
}
