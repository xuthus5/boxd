import { describe, expect, it } from "vitest"

import {
  connectionTargetLogQuery,
  logConnectionQuery,
  logConnectionsHref,
  logDNSHref,
  logDNSServerTag,
} from "@/features/observability/connection-log-links"

describe("connection log links", () => {
  it("normalizes targets with ports and IPv6 brackets", () => {
    expect(connectionTargetLogQuery(undefined)).toBe("")
    expect(connectionTargetLogQuery("  ")).toBe("")
    expect(connectionTargetLogQuery("[2001:db8::1]:443")).toBe("2001:db8::1")
    expect(connectionTargetLogQuery("[]")).toBe("")
    expect(connectionTargetLogQuery("[broken")).toBe("[broken")
    expect(connectionTargetLogQuery("example.com:443")).toBe("example.com")
    expect(connectionTargetLogQuery("example.com:abc")).toBe("example.com:abc")
    expect(connectionTargetLogQuery("2001:db8::1")).toBe("2001:db8::1")
  })

  it("extracts connection hosts from sing-box log shapes", () => {
    expect(logConnectionQuery(undefined)).toBe("")
    expect(logConnectionQuery("  ")).toBe("")
    expect(logConnectionQuery("outbound connection to example.com:443")).toBe("example.com")
    expect(logConnectionQuery("inbound connection to [2001:db8::2]:443")).toBe("2001:db8::2")
    expect(logConnectionQuery("inbound connection from 8.8.8.8:53")).toBe("8.8.8.8")
    expect(logConnectionQuery("inbound connection from 127.0.0.1:1234 lookup example.com")).toBe("example.com")
    expect(logConnectionQuery("lookup example.org")).toBe("example.org")
    expect(logConnectionQuery("query example.net")).toBe("example.net")
    expect(logConnectionQuery("resolve example.dev")).toBe("example.dev")
    expect(logConnectionQuery("dial [2001:db8::3]:443")).toBe("2001:db8::3")
    expect(logConnectionQuery("dial [2001:db8::3]:0")).toBe("")
  })

  it("skips invalid hosts and returns the first plausible fallback", () => {
    expect(logConnectionQuery("dial example.com:70000")).toBe("")
    expect(logConnectionQuery("dial 999.1.1.1:443")).toBe("")
    expect(logConnectionQuery("dial 8.8.8.8:70000")).toBe("")
    expect(logConnectionQuery("dial 127.0.0.1:80 0.0.0.0:80 1.1.1.1:53")).toBe("1.1.1.1")
    expect(logConnectionQuery("no host here")).toBe("")
    expect(logConnectionsHref("outbound connection to example.com:443")).toBe(
      "/observability/connections?q=example.com",
    )
    expect(logConnectionsHref("no host here")).toBe("")
  })

  it("extracts DNS server tags from common phrases", () => {
    expect(logDNSServerTag(undefined)).toBe("")
    expect(logDNSServerTag("  ")).toBe("")
    expect(logDNSServerTag("dns/cloudflare[cf]")).toBe("cf")
    expect(logDNSServerTag("using dns/cloudflare")).toBe("cloudflare")
    expect(logDNSServerTag("server tag: 'remote-dns'")).toBe("remote-dns")
    expect(logDNSServerTag("using quad9 for dns query")).toBe("quad9")
    expect(logDNSServerTag("server: dns")).toBe("")
    expect(logDNSServerTag("dns/udp")).toBe("")
    expect(logDNSServerTag("using udp for lookup")).toBe("")
    expect(logDNSServerTag("ordinary message")).toBe("")
  })

  it("builds DNS policy links by precedence", () => {
    expect(logDNSHref(undefined)).toBe("")
    expect(logDNSHref("ordinary message")).toBe("")
    expect(logDNSHref("dns/cloudflare[cf] lookup example.com")).toBe("/policy/dns?sq=cf")
    expect(logDNSHref("lookup example.com")).toBe("/policy/dns?rq=example.com")
    expect(logDNSHref("dns reject blocked request")).toBe("/policy/dns?raction=reject")
    expect(logDNSHref("dns request observed")).toBe("/policy/dns")
  })
})
