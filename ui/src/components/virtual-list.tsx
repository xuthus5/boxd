import type { ReactNode } from "react"

import { useVirtualWindow } from "@/hooks/use-virtual-window"
import { cn } from "@/lib/utils"

interface VirtualListProps<T> {
  items: readonly T[]
  itemHeight: number
  overscan?: number
  className?: string
  getKey: (item: T, index: number) => string | number
  renderItem: (item: T, index: number) => ReactNode
  role?: string
  "aria-label"?: string
}

export function VirtualList<T>({
  items,
  itemHeight,
  overscan = 6,
  className,
  getKey,
  renderItem,
  role = "list",
  "aria-label": ariaLabel,
}: VirtualListProps<T>) {
  const { parentRef, onScroll, window } = useVirtualWindow({
    count: items.length,
    itemHeight,
    overscan,
  })
  const slice = items.slice(window.startIndex, window.endIndex)

  return (
    <div
      ref={parentRef}
      className={cn("overflow-auto", className)}
      onScroll={onScroll}
      role={role}
      aria-label={ariaLabel}
    >
      <div className="relative w-full" style={{ height: window.totalHeight }}>
        <div className="absolute inset-x-0 top-0" style={{ transform: `translateY(${window.offsetTop}px)` }}>
          {slice.map((item, offset) => {
            const index = window.startIndex + offset
            return (
              <div
                key={getKey(item, index)}
                role={role === "list" ? "listitem" : undefined}
                style={{ height: itemHeight }}
                className="box-border"
              >
                {renderItem(item, index)}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
