import {
  createPolicyNumberTransform,
  type PolicyNumberConstraints,
} from "@/features/policy/policy-number-transform"

const uint16 = 0xFFFF
const int32 = 0x7FFFFFFF
const uint32 = 0xFFFFFFFF

const routeNumberConstraints: PolicyNumberConstraints = {
  ip_version: { kind: "integer", maximum: 6, allowed: [4, 6] },
  source_port: { kind: "integer-list", maximum: uint16 },
  port: { kind: "integer-list", maximum: uint16 },
  user_id: { kind: "integer-list", maximum: int32 },
  override_port: { kind: "integer", maximum: uint16 },
  fallback_delay: { kind: "integer", maximum: uint32, fieldKind: "number" },
  rewrite_ttl: { kind: "integer", maximum: uint32 },
  "default_domain_resolver.rewrite_ttl": { kind: "integer", maximum: uint32 },
  "domain_resolver.rewrite_ttl": { kind: "integer", maximum: uint32 },
  default_mark: { kind: "mark", maximum: uint32 },
  routing_mark: { kind: "mark", maximum: uint32 },
}

export const transformRouteField = createPolicyNumberTransform(routeNumberConstraints)
