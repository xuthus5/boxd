import { kernel114FieldsEn, kernel114FieldsZh } from "@/i18n/locales/kernel114-fields"
import { diagnostics114En, diagnostics114Zh } from "@/i18n/locales/kernel114-diagnostics"

export const kernel114Zh = {
  fields: kernel114FieldsZh,
  diagnostics: diagnostics114Zh,
  ruleSetKernelManaged: "由内核按 update_interval 使用配置的 HTTP 客户端更新；应用不会另行直连下载。",
  ruleSetHTTPClient: "HTTP 客户端：{{client}}（{{source}}）",
  inlineHTTPClient: "内联或默认客户端",
  httpClientSources: { rule_set: "规则集指定", inline: "内联配置", route_default: "路由默认", global_default: "全局默认", default_outbound: "默认出站", legacy_detour: "旧下载出站", unknown: "未指定来源" },
  invalidJSONValue: "请输入有效 JSON；字符串需要双引号。",
  jsonValueHint: "按字段要求填写 JSON 字符串、布尔值或对象。",
  itemJSON: "共享资源 JSON",
  resourceJSON: "共享资源列表 JSON",
  emptyResources: "尚未配置共享资源。添加后可在相应配置中引用其标签。",
  invalidResourceList: "此配置必须是对象数组，请在高级 JSON 中修复。",
  editorDescription: "使用 sing-box 1.14 的字段。复杂参数可在高级 JSON 中完整编辑，保存配置前可执行内核校验。",
  editResource: "编辑 {{tag}}",
  deleteResource: "删除 {{tag}}",
  deleteDescription: "删除后，请同步更新引用此资源的配置。保存时会重新校验。",
  network_namespaces: {
    title: "网络命名空间", add: "添加命名空间", edit: "编辑命名空间",
    description: "复用已有命名空间，或通过 unshare 创建隔离网络。支持 Linux 和 Android，可由 netns 引用。",
  },
  certificate_providers: {
    title: "证书提供者", add: "添加证书提供者", edit: "编辑证书提供者",
    description: "配置 ACME、Tailscale 或 Cloudflare Origin CA，在 TLS 配置中通过 certificate_provider 复用。",
  },
  http_clients: {
    title: "HTTP 客户端", add: "添加 HTTP 客户端", edit: "编辑 HTTP 客户端",
    description: "共享 HTTP/1、HTTP/2 或 HTTP/3 的 TLS、拨号与请求参数，供规则集、证书和面板下载使用。",
  },
}

export const kernel114En = {
  fields: kernel114FieldsEn,
  diagnostics: diagnostics114En,
  ruleSetKernelManaged: "The kernel updates this rule set through its configured HTTP client at update_interval. The app does not download it directly.",
  ruleSetHTTPClient: "HTTP client: {{client}} ({{source}})",
  inlineHTTPClient: "Inline or default client",
  httpClientSources: { rule_set: "Rule set override", inline: "Inline configuration", route_default: "Route default", global_default: "Global default", default_outbound: "Default outbound", legacy_detour: "Legacy download detour", unknown: "Unspecified source" },
  invalidJSONValue: "Enter valid JSON; strings need double quotes.",
  jsonValueHint: "Enter a JSON string, boolean, or object as required by this field.",
  itemJSON: "Shared resource JSON",
  resourceJSON: "Shared resource list JSON",
  emptyResources: "No shared resources configured. Add one to reference its tag in other settings.",
  invalidResourceList: "This section must be an array of objects. Fix it in Advanced JSON.",
  editorDescription: "Configure sing-box 1.14 fields. Advanced JSON supports all complex options; validate with the kernel before saving configuration.",
  editResource: "Edit {{tag}}",
  deleteResource: "Delete {{tag}}",
  deleteDescription: "Update settings that reference this resource. Configuration is validated when saved.",
  network_namespaces: {
    title: "Network namespaces", add: "Add namespace", edit: "Edit namespace",
    description: "Reuse a namespace or create an isolated network with unshare. Supported on Linux and Android; reference it using netns.",
  },
  certificate_providers: {
    title: "Certificate providers", add: "Add certificate provider", edit: "Edit certificate provider",
    description: "Configure ACME, Tailscale, or Cloudflare Origin CA and reuse them through TLS certificate_provider settings.",
  },
  http_clients: {
    title: "HTTP clients", add: "Add HTTP client", edit: "Edit HTTP client",
    description: "Share HTTP/1, HTTP/2, or HTTP/3 TLS, dialer, and request options for rule set, certificate, and dashboard downloads.",
  },
}
