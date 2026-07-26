import { useMemo, useState } from "react"
import { usePlotArea } from "recharts"

import {
  getTrafficTimestampValue,
  type TrafficChartPoint,
} from "@/features/dashboard/traffic-chart-model"

const TRAFFIC_ORIGIN_BUCKET_MS = 60 * 60 * 1000

export type TrafficPlotArea = NonNullable<ReturnType<typeof usePlotArea>>

function samePlotArea(left: TrafficPlotArea | undefined, right: TrafficPlotArea | undefined) {
  return left?.x === right?.x
    && left?.y === right?.y
    && left?.width === right?.width
    && left?.height === right?.height
}

export function useStableTrafficPlotArea() {
  const current = usePlotArea()
  const [stable, setStable] = useState(current)
  if (!current || samePlotArea(current, stable)) return stable
  setStable(current)
  return current
}

function trafficTimeOrigin(data: readonly TrafficChartPoint[]) {
  const timestamp = data.map(getTrafficTimestampValue).find((value) => value > 0)
  return timestamp === undefined ? undefined : Math.floor(timestamp / TRAFFIC_ORIGIN_BUCKET_MS) * TRAFFIC_ORIGIN_BUCKET_MS
}

export function useTrafficTimeOrigin(data: readonly TrafficChartPoint[]) {
  const candidate = useMemo(() => trafficTimeOrigin(data), [data])
  const [origin, setOrigin] = useState(candidate)
  if (origin !== undefined || candidate === undefined) return origin ?? 0
  setOrigin(candidate)
  return candidate
}
