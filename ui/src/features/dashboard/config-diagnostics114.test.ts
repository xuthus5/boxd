import { describe, expect, it } from "vitest"

import { configDiagnosticHref, configDiagnosticIssueHintKey, configDiagnosticIssueLabelKey } from "@/features/dashboard/config-diagnostics"
import { hasProxyOutbound } from "@/features/dashboard/setup-checklist"
import { i18n } from "@/i18n"
import { diagnostic114Codes } from "@/i18n/locales/kernel114-diagnostics"

describe("sing-box 1.14 diagnostics", () => {
  it.each(["build_feature_unavailable", "build_features_unknown"])("explains %s using build capabilities", (code) => {
    const label = configDiagnosticIssueLabelKey(code)
    const hint = configDiagnosticIssueHintKey(code)
    expect(label).toBe(`kernel114.diagnostics.${code}.label`)
    expect(i18n.getFixedT("zh")(hint)).toContain("构建")
    expect(i18n.getFixedT("en")(hint)).toContain("build")
  })

  it("provides Chinese and English explanations for every new backend diagnostic", () => {
    for (const language of ["zh", "en"]) {
      const translate = i18n.getFixedT(language)
      for (const code of diagnostic114Codes) {
        for (const key of [configDiagnosticIssueLabelKey(code), configDiagnosticIssueHintKey(code)]) {
          expect(translate(key), `${language}: ${code}`).not.toBe(key)
          expect(translate(key).length).toBeGreaterThan(3)
        }
      }
    }
  })

  it.each([
    ["network_namespaces[0].path", "/advanced/network-namespaces"],
    ["certificate_providers[1].http_client", "/advanced/certificate-providers"],
    ["http_clients[0].version", "/advanced/http-clients"],
    ["dns.rules[2].match_response", "/policy/dns"],
  ])("links %s to its editable section", (path, href) => {
    expect(configDiagnosticHref(path)).toBe(`${href}?path=${encodeURIComponent(path)}`)
  })

  it("counts upstream VPN endpoints and Snell for onboarding but excludes bridge/server endpoints", () => {
    for (const type of ["openvpn-client", "openconnect", "tailscale", "wireguard"]) {
      expect(hasProxyOutbound({ endpoints: [{ type, tag: "vpn" }] })).toBe(true)
    }
    expect(hasProxyOutbound({ outbounds: [{ type: "snell", tag: "snell" }] })).toBe(true)
    expect(hasProxyOutbound({ outbounds: [{ type: "bridge", tag: "bridge" }], endpoints: [{ type: "openvpn-server", tag: "server" }] })).toBe(false)
  })
})
