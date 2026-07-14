import { SectionConfigPage } from "@/features/advanced/section-config-page"
import { useTranslation } from "react-i18next"

export function EndpointsPage() {
  const { t } = useTranslation()
  return <SectionConfigPage section="endpoints" title={t("pages.endpoints")} description={t("advanced.endpointsDescription")} />
}
