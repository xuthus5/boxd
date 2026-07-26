import { useEffect, useId, useMemo, useRef } from "react"

import {
  TRAFFIC_UPDATE_DURATION_MS,
  getTrafficPointValue,
  getTrafficTimestampValue,
  type TrafficChartPoint,
  type TrafficDomain,
} from "@/features/dashboard/traffic-chart-model"
import {
  useStableTrafficPlotArea,
  type TrafficPlotArea,
} from "@/features/dashboard/traffic-chart-layout"

interface TrafficSeriesProps {
  data: TrafficChartPoint[]
  dataKey: string
  timeOrigin: number
  timeDomain: TrafficDomain
  valueDomain: TrafficDomain
}

interface TrafficSeriesPoint {
  timestamp: number
  value: number
}

interface TrafficSegment {
  key: string
  path: string
}

function trafficSeriesPoints(data: readonly TrafficChartPoint[], dataKey: string) {
  const points: TrafficSeriesPoint[] = []
  for (const point of data) {
    const timestamp = getTrafficTimestampValue(point)
    const value = getTrafficPointValue(point, dataKey)
    if (timestamp <= 0 || value === undefined) continue
    const previous = points.at(-1)
    if (previous?.timestamp === timestamp) {
      points[points.length - 1] = { timestamp, value }
      continue
    }
    if (!previous || timestamp > previous.timestamp) points.push({ timestamp, value })
  }
  return points
}

function trafficSegments(points: readonly TrafficSeriesPoint[], timeOrigin: number) {
  const segments: TrafficSegment[] = []
  for (let index = 1; index < points.length; index += 1) {
    const previous = points[index - 1]
    const current = points[index]
    segments.push({
      key: `${previous.timestamp}:${current.timestamp}`,
      path: `M${previous.timestamp - timeOrigin},${-previous.value}L${current.timestamp - timeOrigin},${-current.value}`,
    })
  }
  return segments
}

function trafficSeriesTransform(input: {
  plotArea: TrafficPlotArea
  timeDomain: TrafficDomain
  timeOrigin: number
  valueDomain: TrafficDomain
}) {
  const timeSpan = input.timeDomain[1] - input.timeDomain[0]
  const valueSpan = input.valueDomain[1] - input.valueDomain[0]
  const scaleX = timeSpan > 0 ? input.plotArea.width / timeSpan : 1
  const scaleY = valueSpan > 0 ? input.plotArea.height / valueSpan : 1
  const translateX = input.plotArea.x - ((input.timeDomain[0] - input.timeOrigin) * scaleX)
  const translateY = input.plotArea.y + input.plotArea.height + (input.valueDomain[0] * scaleY)
  return `matrix(${scaleX}, 0, 0, ${scaleY}, ${translateX}, ${translateY})`
}

function useTrafficMotion(motionRef: React.RefObject<SVGGElement | null>, enabled: boolean) {
  useEffect(() => {
    const motion = motionRef.current
    if (!motion) return
    if (!enabled) {
      motion.classList.remove("transition-transform")
      return
    }
    let readyFrame = 0
    const paintFrame = window.requestAnimationFrame(() => {
      readyFrame = window.requestAnimationFrame(() => motion.classList.add("transition-transform"))
    })
    return () => {
      window.cancelAnimationFrame(paintFrame)
      window.cancelAnimationFrame(readyFrame)
    }
  }, [enabled, motionRef])
}

export function TrafficSeries({ data, dataKey, timeDomain, timeOrigin, valueDomain }: TrafficSeriesProps) {
  const plotArea = useStableTrafficPlotArea()
  const motionRef = useRef<SVGGElement | null>(null)
  const rawId = useId()
  const clipId = `traffic-clip-${rawId.replace(/:/g, "")}`
  const segments = useMemo(
    () => trafficSegments(trafficSeriesPoints(data, dataKey), timeOrigin),
    [data, dataKey, timeOrigin],
  )
  const transform = plotArea ? trafficSeriesTransform({ plotArea, timeDomain, timeOrigin, valueDomain }) : undefined
  useTrafficMotion(motionRef, segments.length > 0)
  if (!plotArea || !transform) return null

  return <>
    <defs><clipPath id={clipId}><rect x={plotArea.x} y={plotArea.y} width={plotArea.width} height={plotArea.height} /></clipPath></defs>
    <g clipPath={`url(#${clipId})`}>
      <g
        ref={motionRef}
        data-traffic-series={dataKey}
        className="ease-linear motion-reduce:transition-none will-change-transform"
        style={{ transform, transformBox: "view-box", transformOrigin: "0 0", transitionDuration: `${TRAFFIC_UPDATE_DURATION_MS}ms` }}
      >
        {segments.map((segment) => <path
          key={segment.key}
          className="traffic-chart-curve"
          data-series={dataKey}
          d={segment.path}
          fill="none"
          stroke={`var(--color-${dataKey})`}
          strokeLinecap="round"
          strokeLinejoin="round"
          vectorEffect="non-scaling-stroke"
        />)}
      </g>
    </g>
  </>
}
