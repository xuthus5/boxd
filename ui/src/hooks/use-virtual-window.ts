import { useCallback, useLayoutEffect, useRef, useState } from "react"

import { computeVirtualWindow, type VirtualWindow } from "@/lib/virtual-window"

interface Options {
  count: number
  itemHeight: number
  overscan?: number
}

export function useVirtualWindow({ count, itemHeight, overscan = 6 }: Options) {
  const parentRef = useRef<HTMLDivElement | null>(null)
  const [scrollTop, setScrollTop] = useState(0)
  const [viewportHeight, setViewportHeight] = useState(0)

  useLayoutEffect(() => {
    const node = parentRef.current
    if (!node) return
    const sync = () => setViewportHeight(node.clientHeight)
    sync()
    if (typeof ResizeObserver === "undefined") return
    const observer = new ResizeObserver(() => sync())
    observer.observe(node)
    return () => observer.disconnect()
  }, [count])

  const onScroll = useCallback(() => {
    const node = parentRef.current
    if (!node) return
    setScrollTop(node.scrollTop)
  }, [])

  const window: VirtualWindow = computeVirtualWindow({
    count,
    scrollTop,
    viewportHeight,
    itemHeight,
    overscan,
  })

  return { parentRef, onScroll, window }
}
