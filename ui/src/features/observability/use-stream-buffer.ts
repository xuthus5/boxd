import { useEffect, useState } from "react"

import { openSSE, type SSEStatus } from "@/lib/api/sse"

export type StreamConnectionStatus = SSEStatus

export function useStreamBuffer<T>(path: string, token: string, limit = 500) {
  const [items, setItems] = useState<T[]>([])
  const [error, setError] = useState("")
  const [paused, setPaused] = useState(false)
  const [status, setStatus] = useState<StreamConnectionStatus>("connecting")

  useEffect(() => {
    if (!token || paused) {
      if (paused) setStatus("closed")
      return
    }
    setStatus("connecting")
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
  }, [limit, path, paused, token])

  return { items, error, status, paused, setPaused, clear: () => setItems([]) }
}
