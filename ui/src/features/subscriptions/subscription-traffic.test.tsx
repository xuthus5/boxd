import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { I18nextProvider } from "react-i18next"

import { SubscriptionTrafficBadges } from "@/features/subscriptions/subscription-traffic"
import { i18n } from "@/i18n"

function renderTraffic(traffic: Parameters<typeof SubscriptionTrafficBadges>[0]["traffic"]) {
  return render(<I18nextProvider i18n={i18n}><SubscriptionTrafficBadges traffic={traffic} /></I18nextProvider>)
}

describe("SubscriptionTrafficBadges", () => {
  it("renders nothing without traffic", () => {
    const { container } = renderTraffic(undefined)
    expect(container).toBeEmptyDOMElement()
  })

  it("renders used traffic against a total quota", async () => {
    await i18n.changeLanguage("zh")
    renderTraffic({ upload: 512, download: 512, total: 2048 })
    expect(screen.getByText(/已用/)).toBeInTheDocument()
    expect(screen.getByText(/不限|KB|B/)).toBeInTheDocument()
  })

  it("marks expired subscriptions", async () => {
    await i18n.changeLanguage("zh")
    renderTraffic({ upload: 1, download: 2, total: 0, expire: "2000-01-01T00:00:00.000Z" })
    expect(screen.getByText(/到期/)).toBeInTheDocument()
  })
})
