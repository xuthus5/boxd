import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { setupCapabilityLabel, setupMessage } from "@/features/setup/setup-messages"
import type { SetupStatus } from "@/lib/api/setup"

export function SetupCapabilities({ capabilities }: { capabilities: SetupStatus["capabilities"] }) {
  const { t } = useTranslation()
  return <Alert>
    <AlertTitle>{t("setup.environment")}</AlertTitle>
    <AlertDescription className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-1.5">
        <Badge variant="secondary">{capabilities.platform}</Badge>
        <Badge variant="outline">{t(capabilities.container ? "setup.container" : "setup.host")}</Badge>
        <Badge variant="outline">{t(setupCapabilityLabel("tun", capabilities.tun_available, capabilities.tun_reason))}</Badge>
        <Badge variant="outline">{t(setupCapabilityLabel("ipv6", capabilities.ipv6_available, capabilities.ipv6_reason))}</Badge>
      </div>
      {!capabilities.tun_available && capabilities.tun_reason ? <p>{setupMessage(capabilities.tun_reason, t)}</p> : null}
      {!capabilities.ipv6_available && capabilities.ipv6_reason ? <p>{setupMessage(capabilities.ipv6_reason, t)}</p> : null}
      <p>{t("setup.defaultListen", { address: capabilities.default_listen })}</p>
    </AlertDescription>
  </Alert>
}
