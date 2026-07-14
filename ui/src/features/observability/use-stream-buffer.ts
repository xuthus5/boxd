import { useEffect, useState } from "react"

import { openSSE } from "@/lib/api/sse"

export function useStreamBuffer<T>(path: string, token: string, limit = 500) {
  const [items, setItems] = useState<T[]>([])
  const [error, setError] = useState("")
  const [paused, setPaused] = useState(false)

  useEffect(() => {
    if (!token || paused) return
    return openSSE<T>({
      path,
      token,
      onEvent: (item) => { setError(""); setItems((current) => [...current, item].slice(-limit)) },
      onError: (reason) => setError(reason.message),
    })
  }, [limit, path, paused, token])

  return { items, error, paused, setPaused, clear: () => setItems([]) }
}
