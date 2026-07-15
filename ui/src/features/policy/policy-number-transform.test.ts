import { describe, expect, it } from "vitest"

import { transformDNSField } from "@/features/policy/dns-form-model"
import {
  createPolicyNumberTransform,
  type PolicyNumberConstraints,
} from "@/features/policy/policy-number-transform"
import { transformRouteField } from "@/features/policy/route-form-transform"
import { setPolicyPath, type PolicyFieldSpec, type PolicyFieldTransform } from "@/features/policy/policy-form-model"

const field = (path: string, kind: PolicyFieldSpec["kind"] = "number"): PolicyFieldSpec => ({ path, label: path, kind })
const apply = (transform: PolicyFieldTransform, path: string, raw: string, kind?: PolicyFieldSpec["kind"]) => (
  transform({}, field(path, kind), raw)
)

describe("generic policy number transforms", () => {
  const constraints: PolicyNumberConstraints = {
    count: { kind: "integer", maximum: 10 },
    ports: { kind: "integer-list", maximum: 65535 },
    mark: { kind: "mark", maximum: 0xFFFFFFFF },
  }
  const transform = createPolicyNumberTransform(constraints)

  it("writes bounded decimal integers and removes empty values", () => {
    expect(apply(transform, "count", "10")).toEqual({ count: 10 })
    expect(transform({ count: 1 }, field("count"), "")).toEqual({})
    for (const raw of ["-1", "1.5", "NaN", "Infinity", "11"]) {
      expect(apply(transform, "count", raw)).toBeNull()
    }
  })

  it("writes integer lists and rejects any invalid token", () => {
    expect(apply(transform, "ports", "0, 53\n65535", "number-list")).toEqual({ ports: [0, 53, 65535] })
    expect(apply(transform, "ports", "", "number-list")).toEqual({})
    for (const raw of ["-1", "1.5", "NaN", "Infinity", "65536", "53, bad"]) {
      expect(apply(transform, "ports", raw, "number-list")).toBeNull()
    }
  })

  it("writes decimal marks as numbers and preserves prefixed uint32 strings", () => {
    expect(apply(transform, "mark", "123", "text")).toEqual({ mark: 123 })
    for (const raw of ["0x7b", "0o173", "0b1111011"]) {
      expect(apply(transform, "mark", raw, "text")).toEqual({ mark: raw })
    }
    for (const raw of ["-1", "1.5", "0x100000000", "0o40000000000", "0b2"]) {
      expect(apply(transform, "mark", raw, "text")).toBeNull()
    }
  })

  it("returns undefined for unconstrained paths", () => {
    expect(apply(transform, "duration", "1s", "text")).toBeUndefined()
  })
})

describe("Route numeric constraints", () => {
  it("enforces uint16 ports, int32 user IDs, uint32 values, marks, and IP versions", () => {
    expect(apply(transformRouteField, "source_port", "0,65535", "number-list"))
      .toEqual({ source_port: [0, 65535] })
    expect(apply(transformRouteField, "port", "65536", "number-list")).toBeNull()
    expect(apply(transformRouteField, "user_id", "2147483647", "number-list"))
      .toEqual({ user_id: [2147483647] })
    expect(apply(transformRouteField, "user_id", "2147483648", "number-list")).toBeNull()
    for (const path of ["rewrite_ttl", "default_domain_resolver.rewrite_ttl", "domain_resolver.rewrite_ttl"]) {
      expect(apply(transformRouteField, path, "4294967295"))
        .toEqual(setPolicyPath({}, path, 4294967295))
      expect(apply(transformRouteField, path, "4294967296")).toBeNull()
    }
    expect(apply(transformRouteField, "override_port", "65535")).toEqual({ override_port: 65535 })
    expect(apply(transformRouteField, "fallback_delay", "10")).toEqual({ fallback_delay: 10 })
    expect(apply(transformRouteField, "fallback_delay", "10", "text")).toBeUndefined()
    expect(apply(transformRouteField, "default_mark", "0x10", "text")).toEqual({ default_mark: "0x10" })
    expect(apply(transformRouteField, "routing_mark", "16", "text")).toEqual({ routing_mark: 16 })
    expect(apply(transformRouteField, "ip_version", "4")).toEqual({ ip_version: 4 })
    expect(apply(transformRouteField, "ip_version", "6")).toEqual({ ip_version: 6 })
    expect(apply(transformRouteField, "ip_version", "5")).toBeNull()
  })
})

describe("DNS numeric constraints", () => {
  it("extends query transforms with bounded DNS numbers", () => {
    for (const path of ["cache_capacity", "rewrite_ttl", "domain_resolver.rewrite_ttl"]) {
      expect(apply(transformDNSField, path, "4294967296")).toBeNull()
      expect(apply(transformDNSField, path, "1.5")).toBeNull()
    }
    expect(apply(transformDNSField, "server_port", "65535")).toEqual({ server_port: 65535 })
    expect(apply(transformDNSField, "server_port", "65536")).toBeNull()
    expect(apply(transformDNSField, "port", "53,443", "number-list")).toEqual({ port: [53, 443] })
    expect(apply(transformDNSField, "user_id", "-1", "number-list")).toBeNull()
    expect(apply(transformDNSField, "routing_mark", "0o20", "text")).toEqual({ routing_mark: "0o20" })
    expect(apply(transformDNSField, "ip_version", "7")).toBeNull()
  })
})
