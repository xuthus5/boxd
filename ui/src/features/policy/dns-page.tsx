import { PolicyPage } from "@/features/policy/policy-page"
import { api } from "@/lib/api/endpoints"
import { useTranslation } from "react-i18next"

export function DNSPage() {
  const { t } = useTranslation()
  return <PolicyPage section="dns" title={t("pages.dns")} installLabel={t("policy.installDNS")} install={api.config.installDNS} />
}
