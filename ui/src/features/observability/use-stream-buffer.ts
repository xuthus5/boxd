import { useCallback, useEffect, useState } from "react"

import { openSSE, type SSEStatus } from "@/lib/api/sse"

export type StreamConnectionStatus = SSEStatus

export function useStreamBuffer<T>(path: string, token: string, limit = 500) {
  const [items, setItems] = useState<T[]>([])
  const [error, setError] = useState("")
  const [paused, setPaused] = useState(false)
  const [status, setStatus] = useState<StreamConnectionStatus>("connecting")
  const [generation, setGeneration] = useState(0)

  const updatePaused = useCallback((next: boolean) => {
    setPaused(next)
    if (next) setStatus("closed")
  }, [])

  const reconnect = useCallback(() => {
    setError("")
    setStatus("connecting")
    setGeneration((current) => current + 1)
  }, [])

  useEffect(() => {
    if (!token || paused) return
    return openSSE<T>({
      path,
      token,
      onEvent: (item) => {
        setError("")
        setItems((current) => [...current, item].slice(-limit))
      },
      onError: (reason) => setError(reason.message),
      onStatus: (next) => {
        setStatus(next)
        if (next === "open") setError("")
      },
    })
  }, [generation, limit, path, paused, token])

  return {
    items,
    error,
    status,
    paused,
    setPaused: updatePaused,
    reconnect,
    clear: () => setItems([]),
  }
}
