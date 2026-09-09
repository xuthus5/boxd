import { endpointDialerFields } from "@/features/advanced/endpoints-form-model"
import { http2Fields } from "@/features/config/kernel114-fields"
import type { JsonObject, PolicyFieldSpec } from "@/features/policy/policy-form-model"

const isAPI = { path: "type", is: "api" }
const isUSBClient = { path: "type", is: "usbip-client" }
const isRealm = { path: "type", is: "hysteria-realm" }

export const services114Fields: readonly PolicyFieldSpec[] = [
  { path: "secret", label: "apiSecret", when: isAPI },
  { path: "access_control_allow_origin", label: "accessControlAllowOrigin", kind: "list", when: isAPI },
  { path: "access_control_allow_private_network", label: "accessControlAllowPrivateNetwork", kind: "boolean", when: isAPI },
  { path: "dashboard", label: "apiDashboard", kind: "json-value", when: isAPI },
  { path: "users", label: "realmUsers", kind: "json-array", required: true, when: isRealm },
  ...http2Fields.map((field) => ({ ...field, when: isRealm })),
  { path: "provider", label: "usbProvider", kind: "select", options: ["default", "dynamic"], when: { path: "type", is: "usbip-server" } },
  { path: "devices", label: "usbDevices", kind: "json-array", when: { path: "type", is: ["usbip-server", "usbip-client"] } },
  { path: "server", label: "server", required: true, when: isUSBClient },
  { path: "server_port", label: "serverPort", kind: "number", when: isUSBClient },
  ...endpointDialerFields.map((field) => ({ ...field, when: isUSBClient })),
]

export function createService114Draft(type: string): JsonObject | undefined {
  if (type === "api") return { type, tag: "", listen: "127.0.0.1", listen_port: 9090 }
  if (type === "hysteria-realm") return { type, tag: "", listen: "127.0.0.1", listen_port: 8080, users: [] }
  if (type === "usbip-server") return { type, tag: "", listen: "127.0.0.1", listen_port: 3240 }
  if (type === "usbip-client") return { type, tag: "", server: "", server_port: 3240 }
  return undefined
}
