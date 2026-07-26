import type { ConfigDiagnostic, ConfigDiagnostics, ConfigDiagnosticsStatus } from "@/lib/api/types"
import { configSectionFromPath, configSectionHref, withConfigPathQuery } from "@/features/config/config-save-error"

const issueLabelKeys: Record<string, string> = {
  config_missing: "configDiagnostics.issueLabels.configMissing",
  config_unreadable: "configDiagnostics.issueLabels.configUnreadable",
  invalid_json: "configDiagnostics.issueLabels.invalidJSON",
  invalid_root: "configDiagnostics.issueLabels.invalidRoot",
  invalid_singbox_config: "configDiagnostics.issueLabels.invalidSingBoxConfig",
  no_inbounds: "configDiagnostics.issueLabels.noInbounds",
  no_outbounds: "configDiagnostics.issueLabels.noOutbounds",
  duplicate_tag: "configDiagnostics.issueLabels.duplicateTag",
  unknown_outbound_reference: "configDiagnostics.issueLabels.unknownOutboundReference",
  unknown_ruleset_reference: "configDiagnostics.issueLabels.unknownRulesetReference",
  unknown_dns_reference: "configDiagnostics.issueLabels.unknownDNSReference",
  tls_insecure: "configDiagnostics.issueLabels.tlsInsecure",
  legacy_dns_server: "configDiagnostics.issueLabels.legacyDNSServer",
  legacy_dns_fakeip: "configDiagnostics.issueLabels.legacyDNSFakeIP",
  outbound_dns_rule_item: "configDiagnostics.issueLabels.outboundDNSRuleItem",
  missing_domain_resolver: "configDiagnostics.issueLabels.missingDomainResolver",
  legacy_domain_strategy: "configDiagnostics.issueLabels.legacyDomainStrategy",
}

const issueHintKeys: Record<string, string> = {
  config_missing: "configDiagnostics.issueHints.configMissing",
  config_unreadable: "configDiagnostics.issueHints.configUnreadable",
  invalid_json: "configDiagnostics.issueHints.invalidJSON",
  invalid_root: "configDiagnostics.issueHints.invalidRoot",
  invalid_singbox_config: "configDiagnostics.issueHints.invalidSingBoxConfig",
  no_inbounds: "configDiagnostics.issueHints.noInbounds",
  no_outbounds: "configDiagnostics.issueHints.noOutbounds",
  duplicate_tag: "configDiagnostics.issueHints.duplicateTag",
  unknown_outbound_reference: "configDiagnostics.issueHints.unknownOutboundReference",
  unknown_ruleset_reference: "configDiagnostics.issueHints.unknownRulesetReference",
  unknown_dns_reference: "configDiagnostics.issueHints.unknownDNSReference",
  tls_insecure: "configDiagnostics.issueHints.tlsInsecure",
  legacy_dns_server: "configDiagnostics.issueHints.legacyDNSServer",
  legacy_dns_fakeip: "configDiagnostics.issueHints.legacyDNSFakeIP",
  outbound_dns_rule_item: "configDiagnostics.issueHints.outboundDNSRuleItem",
  missing_domain_resolver: "configDiagnostics.issueHints.missingDomainResolver",
  legacy_domain_strategy: "configDiagnostics.issueHints.legacyDomainStrategy",
}

const migrationHrefs: Record<string, string> = {
  legacy_dns_server: "https://sing-box.sagernet.org/migration/#migrate-to-new-dns-server-formats",
  legacy_dns_fakeip: "https://sing-box.sagernet.org/migration/#migrate-to-new-dns-server-formats",
  outbound_dns_rule_item: "https://sing-box.sagernet.org/migration/#migrate-outbound-dns-rule-items-to-domain-resolver",
  missing_domain_resolver: "https://sing-box.sagernet.org/migration/#migrate-outbound-dns-rule-items-to-domain-resolver",
  legacy_domain_strategy: "https://sing-box.sagernet.org/migration/#migrate-outbound-domain-strategy-option-to-domain-resolver",
}

export function configDiagnosticStatusKey(status: ConfigDiagnosticsStatus | string): string {
  if (status === "healthy") return "configDiagnostics.healthy"
  if (status === "warning") return "configDiagnostics.warning"
  return "configDiagnostics.error"
}

export function configDiagnosticIssueLabelKey(code: string): string {
  return issueLabelKeys[code] ?? "configDiagnostics.issueLabels.unknown"
}

export function configDiagnosticIssueHintKey(code: string): string {
  return issueHintKeys[code] ?? "configDiagnostics.issueHints.unknown"
}

export function configDiagnosticMigrationHref(code: string): string | undefined {
  return migrationHrefs[code]
}

export function configDiagnosticHref(path?: string): string {
  const value = path?.trim()
  if (!value || value.toLowerCase() === "config") return "/advanced/raw"
  const section = configSectionFromPath(value)?.toLowerCase()
  return withConfigPathQuery(configSectionHref(section), value)
}

export function configDiagnosticIssues(diagnostics: ConfigDiagnostics): ConfigDiagnostic[] {
  return Array.isArray(diagnostics.issues) ? diagnostics.issues : []
}

export function enabledConfigFeatures(features: ConfigDiagnostics["features"]): string[] {
  const labels: Array<[boolean, string]> = [
    [features.tun, "configDiagnostics.featureLabels.tun"],
    [features.clash_api, "configDiagnostics.featureLabels.clashAPI"],
    [features.cache_file, "configDiagnostics.featureLabels.cacheFile"],
    [features.fakeip, "configDiagnostics.featureLabels.fakeIP"],
    [features.selector, "configDiagnostics.featureLabels.selector"],
    [features.urltest, "configDiagnostics.featureLabels.urltest"],
    [features.wireguard, "configDiagnostics.featureLabels.wireguard"],
    [features.remote_rule_set, "configDiagnostics.featureLabels.remoteRuleSet"],
  ]
  return labels.filter(([enabled]) => enabled).map(([, label]) => label)
}
