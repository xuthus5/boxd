import { ProxyListPage } from "@/features/proxy/proxy-list-page"
import { useTranslation } from "react-i18next"

export function OutboundsPage() {
  const { t } = useTranslation()
  return <ProxyListPage configKey="outbounds" title={t("pages.outbounds")} addLabel={t("proxy.addOutbound")} />
}
