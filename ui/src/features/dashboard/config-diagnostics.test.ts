import { describe, expect, it } from "vitest"

import {
  configDiagnosticHref,
  configDiagnosticIssueHintKey,
  configDiagnosticIssueLabelKey,
  configDiagnosticStatusKey,
  enabledConfigFeatures,
} from "@/features/dashboard/config-diagnostics"

describe("config diagnostics helpers", () => {
  it("maps statuses and known or unknown issue codes", () => {
    expect(configDiagnosticStatusKey("healthy")).toBe("configDiagnostics.healthy")
    expect(configDiagnosticStatusKey("warning")).toBe("configDiagnostics.warning")
    expect(configDiagnosticStatusKey("error")).toBe("configDiagnostics.error")
    expect(configDiagnosticIssueLabelKey("invalid_json")).toContain("invalidJSON")
    expect(configDiagnosticIssueHintKey("future_code")).toContain("unknown")
  })

  it("links issue paths to the closest editor", () => {
    expect(configDiagnosticHref("inbounds[0]")).toBe("/proxy/inbounds?path=inbounds%5B0%5D")
    expect(configDiagnosticHref("outbounds[1].detour")).toBe("/proxy/outbounds?path=outbounds%5B1%5D.detour")
    expect(configDiagnosticHref("endpoints[0].tls.insecure")).toBe("/advanced/endpoints?path=endpoints%5B0%5D.tls.insecure")
    expect(configDiagnosticHref("route.rules[0]")).toBe("/policy/route?path=route.rules%5B0%5D")
    expect(configDiagnosticHref("dns.rules[0]")).toBe("/policy/dns?path=dns.rules%5B0%5D")
    expect(configDiagnosticHref("experimental.cache_file")).toBe("/advanced/experimental?path=experimental.cache_file")
    expect(configDiagnosticHref("config")).toBe("/advanced/raw")
  })

  it("returns only enabled sing-box feature labels", () => {
    expect(enabledConfigFeatures({
      tun: true,
      clash_api: false,
      cache_file: true,
      fakeip: false,
      selector: true,
      urltest: false,
      wireguard: false,
      remote_rule_set: true,
    })).toEqual([
      "configDiagnostics.featureLabels.tun",
      "configDiagnostics.featureLabels.cacheFile",
      "configDiagnostics.featureLabels.selector",
      "configDiagnostics.featureLabels.remoteRuleSet",
    ])
  })
})
