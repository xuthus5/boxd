import { useEffect, useId, useMemo, useRef, useState } from "react"

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

interface TrafficMatrix {
  scaleX: number
  scaleY: number
  translateX: number
  translateY: number
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
}): TrafficMatrix {
  const timeSpan = input.timeDomain[1] - input.timeDomain[0]
  const valueSpan = input.valueDomain[1] - input.valueDomain[0]
  const scaleX = timeSpan > 0 ? input.plotArea.width / timeSpan : 1
  const scaleY = valueSpan > 0 ? input.plotArea.height / valueSpan : 1
  const translateX = input.plotArea.x - ((input.timeDomain[0] - input.timeOrigin) * scaleX)
  const translateY = input.plotArea.y + input.plotArea.height + (input.valueDomain[0] * scaleY)
  return { scaleX, scaleY, translateX, translateY }
}

function formatTrafficMatrix(matrix: TrafficMatrix) {
  return `matrix(${matrix.scaleX} 0 0 ${matrix.scaleY} ${matrix.translateX} ${matrix.translateY})`
}

function sameTrafficMatrix(left: TrafficMatrix, right: TrafficMatrix) {
  return left.scaleX === right.scaleX
    && left.scaleY === right.scaleY
    && left.translateX === right.translateX
    && left.translateY === right.translateY
}

function interpolateTrafficMatrix(start: TrafficMatrix, end: TrafficMatrix, progress: number): TrafficMatrix {
  const mix = (from: number, to: number) => from + ((to - from) * progress)
  return {
    scaleX: mix(start.scaleX, end.scaleX),
    scaleY: mix(start.scaleY, end.scaleY),
    translateX: mix(start.translateX, end.translateX),
    translateY: mix(start.translateY, end.translateY),
  }
}

function prefersReducedTrafficMotion() {
  return typeof window.matchMedia === "function"
    && window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

function useTrafficMotion(
  motionRef: React.RefObject<SVGGElement | null>,
  target: TrafficMatrix | undefined,
  enabled: boolean,
) {
  const renderedRef = useRef<TrafficMatrix | null>(null)
  const frameRef = useRef(0)

  useEffect(() => {
    const motion = motionRef.current
    if (!motion || !target) return
    const start = renderedRef.current
    if (!enabled || !start || sameTrafficMatrix(start, target) || prefersReducedTrafficMotion()) {
      renderedRef.current = target
      motion.setAttribute("transform", formatTrafficMatrix(target))
      return
    }

    const startedAt = performance.now()
    const update = (timestamp: number) => {
      const progress = Math.min(1, Math.max(0, (timestamp - startedAt) / TRAFFIC_UPDATE_DURATION_MS))
      const next = interpolateTrafficMatrix(start, target, progress)
      renderedRef.current = next
      motion.setAttribute("transform", formatTrafficMatrix(next))
      frameRef.current = progress < 1 ? window.requestAnimationFrame(update) : 0
    }

    window.cancelAnimationFrame(frameRef.current)
    frameRef.current = window.requestAnimationFrame(update)
    return () => {
      window.cancelAnimationFrame(frameRef.current)
      frameRef.current = 0
    }
  }, [enabled, motionRef, target])
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
  const transform = useMemo(
    () => plotArea ? trafficSeriesTransform({ plotArea, timeDomain, timeOrigin, valueDomain }) : undefined,
    [plotArea, timeDomain, timeOrigin, valueDomain],
  )
  const targetTransform = useMemo(() => transform ? formatTrafficMatrix(transform) : undefined, [transform])
  const [initialTransform] = useState(() => targetTransform)
  useTrafficMotion(motionRef, transform, segments.length > 0)
  if (!plotArea || !transform) return null

  return <>
    <defs><clipPath id={clipId}><rect x={plotArea.x} y={plotArea.y} width={plotArea.width} height={plotArea.height} /></clipPath></defs>
    <g clipPath={`url(#${clipId})`}>
      <g
        ref={motionRef}
        data-traffic-series={dataKey}
        data-traffic-transform-target={targetTransform}
        transform={initialTransform}
      >
        {segments.map((segment) => <path
          key={segment.key}
          className="traffic-chart-curve"
          data-series={dataKey}
          d={segment.path}
          fill="none"
          stroke={`var(--color-${dataKey})`}
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
          vectorEffect="non-scaling-stroke"
        />)}
      </g>
    </g>
  </>
}
