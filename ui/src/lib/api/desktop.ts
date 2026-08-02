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
export async function callBridge(path: string): Promise<DesktopBridgeResponse> {
  // 动态导入避免 web 模式打包时引入 wails 运行时依赖。
  const mod = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
  const resp = await mod.Call({ path })
  return resp as DesktopBridgeResponse
}

/** desktopFetch 桌面模式下的通用请求入口（仅支持 GET 类只读操作）。 */
export async function desktopGet(path: string): Promise<unknown> {
  const resp = await callBridge(path)
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
