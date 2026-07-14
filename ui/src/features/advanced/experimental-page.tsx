import { SectionConfigPage } from "@/features/advanced/section-config-page"
import { useTranslation } from "react-i18next"

export function ExperimentalPage() {
  const { t } = useTranslation()
  return <SectionConfigPage section="experimental" title={t("pages.experimental")} description={t("advanced.experimentalDescription")} />
}
