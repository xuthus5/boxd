import { useCallback, useEffect, useState } from "react"

import { isDesktop, subscribeDesktopStream } from "@/lib/api/desktop"
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
    const append = (item: T) => {
      setError("")
      setItems((current) => [...current, item].slice(-limit))
    }
    const onError = (reason: Error) => setError(reason.message)
    const onStatus = (next: StreamConnectionStatus) => {
      setStatus(next)
      if (next === "open") setError("")
    }

    if (isDesktop()) {
      let disposed = false
      let cleanup: (() => void) | undefined
      void subscribeDesktopStream<T>(path, {
        onEvent: append,
        onStatus,
      }).then((stop) => {
        if (disposed) {
          stop()
          return
        }
        cleanup = stop
      })
      return () => {
        disposed = true
        cleanup?.()
      }
    }

    return openSSE<T>({
      path,
      token,
      onEvent: append,
      onError,
      onStatus,
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
