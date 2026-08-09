import type { APIEnvelope, ApiErrorBody } from "@/lib/api/types"
import { desktopRequest, isDesktop } from "@/lib/api/desktop"
import { sessionStore } from "@/lib/session"

type UnauthorizedHandler = (() => void) | undefined
let unauthorizedHandler: UnauthorizedHandler

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(
    message: string,
    status: number,
    code = "request_failed",
  ) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
  }
}

export function setUnauthorizedHandler(handler: UnauthorizedHandler) {
  unauthorizedHandler = handler
}

export function notifyUnauthorized() {
  unauthorizedHandler?.()
}

function isEnvelope<T>(value: unknown): value is APIEnvelope<T> {
  return Boolean(value && typeof value === "object" && "status" in value && "data" in value)
}

function errorBody(value: unknown): ApiErrorBody | null {
  if (!value || typeof value !== "object") return null
  const candidate = value as Partial<ApiErrorBody>
  return typeof candidate.code === "string" && typeof candidate.message === "string"
    ? { code: candidate.code, message: candidate.message }
    : null
}

async function parseResponse(response: Response): Promise<unknown> {
  if (response.status === 204) return undefined
  const text = await response.text()
  if (!text) return undefined
  try {
    return JSON.parse(text) as unknown
  } catch {
    throw new ApiError("Invalid JSON response", response.status, "invalid_response")
  }
}

function createHeaders(init: RequestInit) {
  const headers = new Headers(init.headers)
  headers.set("Accept", "application/json")
  if (init.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }
  const token = sessionStore.get()?.token
  if (token) headers.set("Authorization", `Bearer ${token}`)
  return headers
}

function responseError<T>(response: Response, payload: unknown, envelope?: APIEnvelope<T>) {
  const body = envelope?.error ?? errorBody(payload)
  return new ApiError(body?.message ?? response.statusText, response.status, body?.code)
}

export async function apiRequestEnvelope<T>(path: string, init: RequestInit = {}): Promise<APIEnvelope<T>> {
  if (isDesktop()) {
    try {
      const method = (init.method ?? "GET").toUpperCase()
      const body = init.body === undefined ? undefined
        : typeof init.body === "string" ? init.body : init.body
      const data = await desktopRequest(path, method, body)
      // bridge 返回 ApplyResult 时回滚语义放在 data 内（status/rolled_back），
      // 这里还原为与 web 端一致的 envelope，避免回滚被误判为保存成功。
      if (data && typeof data === "object" && (data as { rolled_back?: boolean }).rolled_back) {
        const result = data as { status?: string; api_error?: { code?: string; message?: string } }
        return {
          status: result.status === "rolled_back" ? "rolled_back" : "ok",
          data: null as unknown as T,
          error: result.api_error ? { code: result.api_error.code ?? "", message: result.api_error.message ?? "" } : null,
          meta: { rolled_back: true },
        }
      }
      return { status: "ok" as const, data: data as T, error: null, meta: null }
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.status === 401) notifyUnauthorized()
        throw error
      }
      if (error instanceof Error && error.message.includes("unknown path")) {
        // bridge 未实现的路径回退到 HTTP（远程模式）。
        return requestViaFetch<T>(path, init)
      }
      throw new ApiError(
        error instanceof Error ? error.message : "Desktop request failed",
        500,
        "request_failed",
      )
    }
  }
  return requestViaFetch<T>(path, init)
}

async function requestViaFetch<T>(path: string, init: RequestInit) {
  const response = await fetch(path, { ...init, headers: createHeaders(init) })
  const payload = await parseResponse(response)
  const envelope = isEnvelope<T>(payload) ? payload : undefined
  if (!response.ok) {
    const error = responseError(response, payload, envelope)
    if (response.status === 401) notifyUnauthorized()
    throw error
  }
  return envelope ?? { status: "ok" as const, data: payload as T, error: null, meta: null }
}

export async function apiRequest<T>(path: string, init: RequestInit = {}) {
  const response = await apiRequestEnvelope<T>(path, init)
  return response.data
}

export type BinaryDownload = {
  blob: Blob
  filename: string
  contentType: string
}

function filenameFromDisposition(value: string | null): string {
  if (!value) return ""
  const utfMatch = /filename\*=UTF-8''([^;]+)/i.exec(value)
  if (utfMatch?.[1]) {
    try {
      return decodeURIComponent(utfMatch[1].trim().replace(/^"|"$/g, ""))
    } catch {
      return utfMatch[1].trim().replace(/^"|"$/g, "")
    }
  }
  const plainMatch = /filename="?([^";]+)"?/i.exec(value)
  return plainMatch?.[1]?.trim() ?? ""
}

export async function downloadBinary(path: string, init: RequestInit = {}, fallbackName = "download.bin"): Promise<BinaryDownload> {
  const headers = createHeaders(init)
  headers.delete("Accept")
  headers.set("Accept", "application/gzip, application/octet-stream, */*")
  const response = await fetch(path, { ...init, headers })
  if (!response.ok) {
    let payload: unknown
    try {
      payload = await parseResponse(response)
    } catch (error) {
      if (response.status === 401) notifyUnauthorized()
      throw error instanceof ApiError
        ? error
        : new ApiError(response.statusText || "Download failed", response.status)
    }
    const envelope = isEnvelope<unknown>(payload) ? payload : undefined
    const error = responseError(response, payload, envelope)
    if (response.status === 401) notifyUnauthorized()
    throw error
  }
  const blob = await response.blob()
  const filename = filenameFromDisposition(response.headers.get("Content-Disposition")) || fallbackName
  const contentType = response.headers.get("Content-Type") || blob.type || "application/octet-stream"
  return { blob, filename, contentType }
}

export function triggerBrowserDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement("a")
  anchor.href = url
  anchor.download = filename
  anchor.rel = "noopener"
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

