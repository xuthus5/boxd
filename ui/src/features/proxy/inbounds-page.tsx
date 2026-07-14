import { ProxyListPage } from "@/features/proxy/proxy-list-page"
import { useTranslation } from "react-i18next"

export function InboundsPage() {
  const { t } = useTranslation()
  return <ProxyListPage configKey="inbounds" title={t("pages.inbounds")} addLabel={t("proxy.addInbound")} />
}
