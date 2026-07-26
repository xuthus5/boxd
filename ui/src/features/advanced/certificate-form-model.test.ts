import { describe, expect, it } from "vitest"

import {
  applyCertificateConfig,
  certificateFields,
  certificateStores,
  hasCertificateSources,
  isCertificateStructureValid,
  normalizeCertificateObject,
  prepareCertificateObject,
} from "@/features/advanced/certificate-form-model"

describe("certificate form model", () => {
  it("normalizes the default store and sing-box Listable values", () => {
    expect(normalizeCertificateObject(undefined)).toEqual({ store: "system" })
    expect(normalizeCertificateObject({
      store: "mozilla",
      certificate: "-----BEGIN CERTIFICATE-----",
      certificate_path: "/etc/boxd/ca.pem",
      certificate_directory_path: ["/etc/boxd/certs"],
    })).toEqual({
      store: "mozilla",
      certificate: ["-----BEGIN CERTIFICATE-----"],
      certificate_path: ["/etc/boxd/ca.pem"],
      certificate_directory_path: ["/etc/boxd/certs"],
    })
  })

  it("accepts valid Listable fields and rejects malformed structures", () => {
    expect(isCertificateStructureValid({ store: "chrome", certificate: ["pem"] })).toBe(true)
    expect(isCertificateStructureValid({ certificate_path: "ca.pem" })).toBe(true)
    expect(isCertificateStructureValid({ certificate: ["pem", "pem-2"] })).toBe(true)
    expect(isCertificateStructureValid([])).toBe(false)
    expect(isCertificateStructureValid({ certificate: { pem: true } })).toBe(false)
    expect(isCertificateStructureValid({ certificate_path: ["ca.pem", 42] })).toBe(false)
    expect(isCertificateStructureValid({ certificate_directory_path: false })).toBe(false)
  })

  it("removes default values and empty entries while preserving unknown fields", () => {
    const prepared = prepareCertificateObject({
      store: "system",
      certificate: ["", "  ", "PEM DATA"],
      certificate_path: [" /etc/boxd/ca.pem ", ""],
      certificate_directory_path: ["/etc/boxd/certs"],
      future_option: { enabled: true },
    })
    expect(prepared).toEqual({
      certificate: ["PEM DATA"],
      certificate_path: ["/etc/boxd/ca.pem"],
      certificate_directory_path: ["/etc/boxd/certs"],
      future_option: { enabled: true },
    })
    expect(prepareCertificateObject({ store: "system" })).toEqual({})
    expect(prepareCertificateObject({ store: "none" })).toEqual({ store: "none" })
  })

  it("detects only non-empty custom certificate sources", () => {
    expect(hasCertificateSources({ certificate: ["PEM DATA"] })).toBe(true)
    expect(hasCertificateSources({ certificate_path: " /etc/boxd/ca.pem " })).toBe(true)
    expect(hasCertificateSources({ certificate_directory_path: ["", "  "] })).toBe(false)
    expect(hasCertificateSources({ future_option: true })).toBe(false)
  })

  it("omits an empty default certificate section from the full config", () => {
    const config = { log: { level: "info" }, certificate: { store: "system" } }
    expect(applyCertificateConfig(config, { store: "system" })).toEqual({ log: { level: "info" } })
    expect(config).toHaveProperty("certificate.store", "system")
    expect(applyCertificateConfig(config, { store: "none" })).toEqual({
      log: { level: "info" },
      certificate: { store: "none" },
    })
  })

  it("exposes the supported stores and visual fields", () => {
    expect(certificateStores).toEqual(["system", "mozilla", "chrome", "none"])
    expect(new Set(certificateFields.map((field) => field.path))).toEqual(new Set([
      "store",
      "certificate",
      "certificate_path",
      "certificate_directory_path",
    ]))
    expect(certificateFields.find((field) => field.path === "certificate")?.kind).toBe("textarea")
  })
})
