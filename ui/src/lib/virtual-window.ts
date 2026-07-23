/** Fixed-height list window math for lightweight virtualization. */

export interface VirtualWindowInput {
  count: number
  scrollTop: number
  viewportHeight: number
  itemHeight: number
  overscan?: number
}

export interface VirtualWindow {
  startIndex: number
  endIndex: number
  offsetTop: number
  totalHeight: number
}

export function computeVirtualWindow(input: VirtualWindowInput): VirtualWindow {
  const count = Math.max(0, Math.floor(input.count))
  const itemHeight = Math.max(1, Math.floor(input.itemHeight))
  const overscan = Math.max(0, Math.floor(input.overscan ?? 6))
  const viewportHeight = Math.max(0, Math.floor(input.viewportHeight))
  const scrollTop = Math.max(0, input.scrollTop)
  const totalHeight = count * itemHeight
  if (count === 0) {
    return { startIndex: 0, endIndex: 0, offsetTop: 0, totalHeight: 0 }
  }
  const rawStart = Math.floor(scrollTop / itemHeight)
  const visible = Math.ceil(viewportHeight / itemHeight) + 1
  let startIndex = Math.max(0, rawStart - overscan)
  let endIndex = Math.min(count, rawStart + visible + overscan)
  if (startIndex > endIndex) {
    startIndex = Math.max(0, endIndex - visible - overscan)
  }
  if (endIndex < startIndex) {
    endIndex = startIndex
  }
  return {
    startIndex,
    endIndex,
    offsetTop: startIndex * itemHeight,
    totalHeight,
  }
}
