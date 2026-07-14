import { notifyUnauthorized } from "@/lib/api/client"

export interface SSEOptions<T> {
  path: string
  token: string
  signal?: AbortSignal
  onEvent: (event: T) => void
  onError?: (error: Error) => void
}

function eventData(block: string) {
  return block
    .split(/\r?\n/)
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trimStart())
    .join("\n")
}

function parseEvent<T>(block: string): T | undefined {
  const data = eventData(block)
  if (!data) return undefined
  try {
    return JSON.parse(data) as T
  } catch {
    throw new Error("Invalid SSE event data")
  }
}

export async function consumeSSEStream<T>(
  stream: ReadableStream<Uint8Array>,
  onEvent: (event: T) => void,
) {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ""

  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value, { stream: !done })
    const blocks = buffer.split(/\r?\n\r?\n/)
    buffer = blocks.pop() ?? ""
    blocks.forEach((block) => {
      const event = parseEvent<T>(block)
      if (event !== undefined) onEvent(event)
    })
    if (done) break
  }
  const event = parseEvent<T>(buffer)
  if (event !== undefined) onEvent(event)
}

export function openSSE<T>(options: SSEOptions<T>) {
  let active: AbortController | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  let attempt = 0
  let stopped = false

  const stop = () => {
    stopped = true
    if (timer) clearTimeout(timer)
    active?.abort()
  }
  const retry = () => {
    if (stopped) return
    const delay = Math.min(1000 * (2 ** attempt), 10000)
    attempt += 1
    timer = setTimeout(connect, delay)
  }
  const connect = async () => {
    active = new AbortController()
    try {
      const response = await fetch(options.path, {
        headers: { Authorization: `Bearer ${options.token}`, Accept: "text/event-stream" },
        signal: active.signal,
      })
      if (response.status === 401) {
        stopped = true
        notifyUnauthorized()
        options.onError?.(new Error("SSE request failed with status 401"))
        return
      }
      if (!response.ok || !response.body) throw new Error(`SSE request failed with status ${response.status}`)
      attempt = 0
      await consumeSSEStream(response.body, options.onEvent)
      retry()
    } catch (reason: unknown) {
      if (stopped || active.signal.aborted) return
      options.onError?.(reason instanceof Error ? reason : new Error("SSE connection failed"))
      retry()
    }
  }

  options.signal?.addEventListener("abort", stop, { once: true })
  void connect()
  return () => {
    options.signal?.removeEventListener("abort", stop)
    stop()
  }
}
