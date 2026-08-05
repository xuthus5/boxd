/**
 * desktop.ts —— 桌面运行环境探测与 bridge 调用。
 *
 * 桌面模式（Wails3）下，Wails runtime 会注入 window._wails 全局对象。
 * 通过它探测运行形态：web（浏览器）或 desktop（Wails WebView）。
 */

type WailsWindow = Window & { _wails?: unknown }

/** isDesktop 判断当前是否运行在 Wails 桌面环境。 */
export function isDesktop(): boolean {
  return Boolean((window as WailsWindow)._wails)
}

/** DesktopBridge 封装 Wails bridge 调用，路径语义与 REST 对齐。 */
export interface DesktopBridgeResponse {
  data?: unknown
  error?: string
  status?: string
}

/** callBridge 调用桌面 bridge 服务。 */
export async function callBridge(path: string, method = "GET", body?: unknown): Promise<DesktopBridgeResponse> {
  // 动态导入避免 web 模式打包时引入 wails 运行时依赖。
  const mod = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
  const req: { path: string; method: string; body?: unknown } = { path, method }
  if (body !== undefined) {
    // Wails 的 json.RawMessage 字段期望对象而非 JSON 字符串；
    // 前端 transport 传入的是 JSON.stringify 后的字符串，这里转回对象。
    req.body = typeof body === "string" ? parseJsonBody(body) : body
  }
  const resp = await mod.Call(req)
  return resp as DesktopBridgeResponse
}

/** parseJsonBody 把 JSON 字符串解析为对象；解析失败时原样返回，交由 Go 端报错。 */
function parseJsonBody(body: string): unknown {
  try {
    return JSON.parse(body)
  } catch {
    return body
  }
}

/** desktopGet 桌面模式下的通用请求入口（仅支持 GET 类只读操作）。 */
export async function desktopGet(path: string): Promise<unknown> {
  const resp = await callBridge(path)
  if (resp.error) {
    throw new Error(resp.error)
  }
  return resp.data
}

/** desktopRequest 桌面模式下完整请求入口（支持任意方法）。 */
export async function desktopRequest(path: string, method: string, body?: unknown): Promise<unknown> {
  const resp = await callBridge(path, method, body)
  if (resp.error) {
    throw new Error(resp.error)
  }
  return resp.data
}

/** EmbeddedSession 描述内嵌自动登录返回的会话。 */
export interface EmbeddedSession {
  token: string
  expires_at: string
}

/** autoLogin 内嵌模式自动登录，返回一次性 JWT 会话。 */
export async function autoLogin(): Promise<EmbeddedSession> {
  const mod = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdauthservice")
  const session = await mod.AutoLogin()
  return session as EmbeddedSession
}

/** streamPathToEventName 将 SSE 路径映射到 Wails 事件名。 */
const STREAM_EVENT_MAP: Record<string, string> = {
  "/api/stats/traffic": "boxd:traffic",
  "/api/stats/connections": "boxd:connections",
  "/api/stats/logs": "boxd:kernel-log",
  "/api/stats/app-logs": "boxd:app-log",
}

/** streamEventName 返回路径对应的事件名；未知路径返回 null。 */
export function streamEventName(path: string): string | null {
  return STREAM_EVENT_MAP[path] ?? null
}

/** StreamEventSubscription 描述一次事件订阅。 */
export interface StreamEventSubscription<T> {
  onEvent: (event: T) => void
  onStatus?: (status: "connecting" | "open" | "reconnecting" | "closed") => void
}

/**
 * subscribeDesktopStream 在桌面模式订阅 Wails 事件流，替代 SSE fetch。
 * 返回取消函数。
 */
export async function subscribeDesktopStream<T>(
  path: string,
  options: StreamEventSubscription<T>,
): Promise<() => void> {
  const eventName = streamEventName(path)
  if (!eventName) {
    options.onStatus?.("closed")
    return () => {}
  }
  const runtime = await import("@wailsio/runtime")
  options.onStatus?.("connecting")
  options.onStatus?.("open")
  const unsubscribe = runtime.Events.On(eventName, (ev: { data: unknown }) => {
    if (ev.data && typeof ev.data === "object" && "type" in (ev.data as object) && (ev.data as { type: string }).type === "heartbeat") {
      return
    }
    options.onEvent(ev.data as T)
  })
  return () => {
    unsubscribe()
    options.onStatus?.("closed")
  }
}
