import { describe, expect, it } from "vitest"

import {
  configDiagnosticHref,
  configDiagnosticIssueHintKey,
  configDiagnosticIssueLabelKey,
  configDiagnosticMigrationHref,
  configDiagnosticStatusKey,
  enabledConfigFeatures,
} from "@/features/dashboard/config-diagnostics"

describe("config diagnostics helpers", () => {
  it("maps statuses and known or unknown issue codes", () => {
    expect(configDiagnosticStatusKey("healthy")).toBe("configDiagnostics.healthy")
    expect(configDiagnosticStatusKey("warning")).toBe("configDiagnostics.warning")
    expect(configDiagnosticStatusKey("error")).toBe("configDiagnostics.error")
    expect(configDiagnosticIssueLabelKey("invalid_json")).toContain("invalidJSON")
    expect(configDiagnosticIssueLabelKey("missing_tag")).toContain("missingTag")
    expect(configDiagnosticIssueHintKey("empty_group")).toContain("emptyGroup")
    expect(configDiagnosticIssueHintKey("future_code")).toContain("unknown")
  })

  it("links sing-box deprecations to official migration guides", () => {
    expect(configDiagnosticMigrationHref("legacy_dns_server")).toContain("migrate-to-new-dns-server-formats")
    expect(configDiagnosticMigrationHref("outbound_dns_rule_item")).toContain("migrate-outbound-dns-rule-items")
    expect(configDiagnosticMigrationHref("legacy_domain_strategy")).toContain(
      "migrate-outbound-domain-strategy-option-to-domain-resolver",
    )
    expect(configDiagnosticMigrationHref("duplicate_tag")).toBeUndefined()
  })

  it("links issue paths to the closest editor", () => {
    expect(configDiagnosticHref("inbounds[0]")).toBe("/proxy/inbounds?path=inbounds%5B0%5D")
    expect(configDiagnosticHref("outbounds[1].detour")).toBe("/proxy/outbounds?path=outbounds%5B1%5D.detour")
    expect(configDiagnosticHref("endpoints[0].tls.insecure")).toBe(
      "/advanced/endpoints?path=endpoints%5B0%5D.tls.insecure",
    )
    expect(configDiagnosticHref("route.rules[0]")).toBe("/policy/route?path=route.rules%5B0%5D")
    expect(configDiagnosticHref("route.rule_set[0].tag")).toBe("/policy/route?path=route.rule_set%5B0%5D.tag")
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
