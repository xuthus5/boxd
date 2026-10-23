import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { useLocation, useNavigate } from "react-router-dom"

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"

export type AdvancedSection = "connections" | "certificates" | "resources" | "kernel" | "configuration"

interface AdvancedTab {
  label: string
  to: string
}

interface AdvancedSectionConfig {
  label: string
  tabs: readonly AdvancedTab[]
}

const advancedSections: Record<AdvancedSection, AdvancedSectionConfig> = {
  connections: {
    label: "nav.endpointsAndServices",
    tabs: [
      { label: "nav.endpoints", to: "/advanced/endpoints" },
      { label: "nav.services", to: "/advanced/services" },
    ],
  },
  certificates: {
    label: "nav.certificate",
    tabs: [
      { label: "advanced.certificate.storeTitle", to: "/advanced/certificate" },
      { label: "kernel114.certificate_providers.title", to: "/advanced/certificate-providers" },
    ],
  },
  resources: {
    label: "nav.sharedResources",
    tabs: [
      { label: "kernel114.network_namespaces.title", to: "/advanced/network-namespaces" },
      { label: "kernel114.http_clients.title", to: "/advanced/http-clients" },
    ],
  },
  kernel: {
    label: "nav.kernelOptions",
    tabs: [
      { label: "nav.logConfig", to: "/advanced/log" },
      { label: "nav.ntp", to: "/advanced/ntp" },
      { label: "nav.experimental", to: "/advanced/experimental" },
    ],
  },
  configuration: {
    label: "nav.configManagement",
    tabs: [
      { label: "nav.configHistory", to: "/advanced/history" },
      { label: "nav.rawConfig", to: "/advanced/raw" },
    ],
  },
}

export function AdvancedSectionTabs({ section }: { section: AdvancedSection }) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const current = advancedSections[section]
  return (
    <Tabs value={pathname} onValueChange={(value) => navigate(String(value))} aria-label={t(current.label)}>
      <TabsList variant="line" className="h-auto w-full max-w-full justify-start overflow-x-auto overflow-y-hidden">
        {current.tabs.map((tab) => <TabsTrigger key={tab.to} value={tab.to}>{t(tab.label)}</TabsTrigger>)}
      </TabsList>
    </Tabs>
  )
}

export function AdvancedSectionLayout({ section, children }: { section: AdvancedSection; children: ReactNode }) {
  return <div className="flex min-w-0 flex-col gap-3"><AdvancedSectionTabs section={section} />{children}</div>
}
