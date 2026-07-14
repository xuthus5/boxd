import type { APIEnvelope, ApiErrorBody } from "@/lib/api/types"
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

export async function apiRequestEnvelope<T>(path: string, init: RequestInit = {}) {
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
