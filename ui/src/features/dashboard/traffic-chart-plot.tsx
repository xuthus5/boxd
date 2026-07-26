import { useEffect, useMemo, useRef, useState } from "react"
import { Curve, Line, XAxis, YAxis, type CurveProps, usePlotArea } from "recharts"

import {
  TRAFFIC_AXIS_WIDTH,
  TRAFFIC_UPDATE_DURATION_MS,
  formatTrafficTimestamp,
  getTrafficPointValue,
  getTrafficTimestampValue,
  type TrafficChartPoint,
  type TrafficDomain,
} from "@/features/dashboard/traffic-chart-model"

const TRAFFIC_TIME_TICK_STEPS = [15_000, 30_000, 60_000] as const
const TRAFFIC_MIN_TIME_TICK_SPACING = 80
const TRAFFIC_VALUE_TICK_COUNT = 4
const TRAFFIC_X_LABEL_OFFSET = 20
const TRAFFIC_Y_LABEL_OFFSET = 8
const EMPTY_CURVE_POINTS: readonly TrafficCurvePoint[] = []

type TrafficPlotArea = NonNullable<ReturnType<typeof usePlotArea>>

interface TrafficCurvePoint {
  x: number | null
  y: number | null
  payload?: { timestamp?: unknown }
}

interface SmoothTrafficCurveProps extends Omit<CurveProps, "points"> {
  points?: readonly TrafficCurvePoint[]
  transitionDuration?: number
}

interface TrafficPlotProps {
  data: TrafficChartPoint[]
  uploadKey: string
  downloadKey: string
  formatter: (value: number) => string
  timeDomain: TrafficDomain
  valueDomain: TrafficDomain
}

interface TrafficAxesProps extends Pick<TrafficPlotProps, "timeDomain" | "valueDomain" | "formatter"> {
  hasData: boolean
}

interface TrafficSeriesProps {
  data: TrafficChartPoint[]
  dataKey: string
  timeDomain: TrafficDomain
  valueDomain: TrafficDomain
}

function curveTimestamp(point: TrafficCurvePoint) {
  const timestamp = point.payload?.timestamp
  return typeof timestamp === "string" ? timestamp : undefined
}

function curveStartPoints(previous: readonly TrafficCurvePoint[], target: readonly TrafficCurvePoint[]) {
  if (previous.length === 0 || target.length === 0) return undefined
  const previousByTimestamp = new Map<string, TrafficCurvePoint>()
  for (const point of previous) {
    const timestamp = curveTimestamp(point)
    if (timestamp) previousByTimestamp.set(timestamp, point)
  }
  const tail = previous.at(-1)
  let matches = 0
  const start = target.map((point, index) => {
    const timestamp = curveTimestamp(point)
    const matched = timestamp ? previousByTimestamp.get(timestamp) : previous[index]
    if (matched) matches += 1
    const origin = matched ?? tail
    return origin ? { ...point, x: origin.x, y: origin.y } : point
  })
  return matches > 0 ? start : undefined
}

function interpolateCoordinate(start: number | null, target: number | null, progress: number) {
  if (start === null || target === null) return target
  return start + ((target - start) * progress)
}

function interpolateCurvePoints(
  start: readonly TrafficCurvePoint[],
  target: readonly TrafficCurvePoint[],
  progress: number,
) {
  return target.map((point, index) => {
    const origin = start[index] ?? point
    return {
      ...point,
      x: interpolateCoordinate(origin.x, point.x, progress),
      y: interpolateCoordinate(origin.y, point.y, progress),
    }
  })
}

function shouldSkipCurveMotion(duration: number) {
  return duration <= 0 || window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

export function SmoothTrafficCurve({
  points,
  transitionDuration = TRAFFIC_UPDATE_DURATION_MS,
  ...curveProps
}: SmoothTrafficCurveProps) {
  const [displayedPoints, setDisplayedPoints] = useState<readonly TrafficCurvePoint[]>(points ?? EMPTY_CURVE_POINTS)
  const displayedPointsRef = useRef(displayedPoints)

  useEffect(() => {
    if (points === undefined || displayedPointsRef.current === points) return
    const start = curveStartPoints(displayedPointsRef.current, points)
    if (!start || shouldSkipCurveMotion(transitionDuration)) {
      displayedPointsRef.current = points
      setDisplayedPoints(points)
      return
    }
    const startedAt = performance.now()
    let frame = 0
    const update = (timestamp: number) => {
      const progress = Math.min(1, Math.max(0, (timestamp - startedAt) / transitionDuration))
      const next = progress === 1 ? points : interpolateCurvePoints(start, points, progress)
      displayedPointsRef.current = next
      setDisplayedPoints(next)
      if (progress < 1) frame = window.requestAnimationFrame(update)
    }
    frame = window.requestAnimationFrame(update)
    return () => window.cancelAnimationFrame(frame)
  }, [points, transitionDuration])

  return <Curve {...curveProps} points={displayedPoints} />
}

function useStablePlotArea() {
  const current = usePlotArea()
  const [stable, setStable] = useState(current)
  useEffect(() => {
    if (!current || current === stable) return
    const frame = window.requestAnimationFrame(() => setStable(current))
    return () => window.cancelAnimationFrame(frame)
  }, [current, stable])
  return current ?? stable
}

function scaleCoordinate(input: {
  value: number
  domain: TrafficDomain
  start: number
  length: number
  invert?: boolean
}) {
  const [minimum, maximum] = input.domain
  const progress = maximum === minimum ? 0 : (input.value - minimum) / (maximum - minimum)
  return input.start + (input.invert ? 1 - progress : progress) * input.length
}

function scaledCurvePoints(input: TrafficSeriesProps & { plotArea: TrafficPlotArea }) {
  const points: TrafficCurvePoint[] = []
  for (const point of input.data) {
    const timestamp = getTrafficTimestampValue(point)
    const value = getTrafficPointValue(point, input.dataKey)
    if (value === undefined || timestamp < input.timeDomain[0] || timestamp > input.timeDomain[1]) continue
    points.push({
      x: scaleCoordinate({ value: timestamp, domain: input.timeDomain, start: input.plotArea.x, length: input.plotArea.width }),
      y: scaleCoordinate({ value, domain: input.valueDomain, start: input.plotArea.y, length: input.plotArea.height, invert: true }),
      payload: point,
    })
  }
  return points
}

function TrafficSeries(props: TrafficSeriesProps) {
  const { data, dataKey, timeDomain, valueDomain } = props
  const plotArea = useStablePlotArea()
  const points = useMemo(
    () => plotArea ? scaledCurvePoints({ data, dataKey, timeDomain, valueDomain, plotArea }) : undefined,
    [data, dataKey, plotArea, timeDomain, valueDomain],
  )
  return <SmoothTrafficCurve
    className="traffic-chart-curve"
    data-series={dataKey}
    points={points}
    type="monotone"
    fill="none"
    stroke={`var(--color-${dataKey})`}
    strokeLinecap="round"
    strokeLinejoin="round"
  />
}

function domainTicks(domain: TrafficDomain, count: number) {
  const step = (domain[1] - domain[0]) / count
  return Array.from({ length: count + 1 }, (_, index) => domain[0] + step * index)
}

function timeTickStep(width: number) {
  const tickCount = Math.max(1, Math.floor(width / TRAFFIC_MIN_TIME_TICK_SPACING))
  const windowSize = TRAFFIC_TIME_TICK_STEPS.at(-1) ?? 60_000
  const desiredStep = windowSize / tickCount
  return TRAFFIC_TIME_TICK_STEPS.find((step) => step >= desiredStep) ?? windowSize
}

function timeTicks(domain: TrafficDomain, width: number) {
  const ticks: number[] = []
  const step = timeTickStep(width)
  const first = Math.ceil(domain[0] / step) * step
  for (let value = first; value <= domain[1]; value += step) ticks.push(value)
  return ticks
}

function TrafficAxes({ timeDomain, valueDomain, formatter, hasData }: TrafficAxesProps) {
  const plotArea = useStablePlotArea()
  const [initialDomainEnd] = useState(timeDomain[1])
  if (!plotArea || !hasData) return null
  const xTicks = timeTicks(timeDomain, plotArea.width)
  const yTicks = domainTicks(valueDomain, TRAFFIC_VALUE_TICK_COUNT)
  const tickMotionClass = timeDomain[1] === initialDomainEnd
    ? undefined
    : "transition-transform duration-[800ms] ease-linear motion-reduce:transition-none"
  return <g aria-hidden="true" pointerEvents="none">
    {xTicks.map((value) => <g
      key={value}
      data-traffic-time-tick={value}
      className={tickMotionClass}
      style={{ transform: `translateX(${scaleCoordinate({ value, domain: timeDomain, start: plotArea.x, length: plotArea.width })}px)` }}
    >
      <line className="stroke-border/50" y1={plotArea.y} y2={plotArea.y + plotArea.height} />
      <text className="fill-muted-foreground text-xs" y={plotArea.y + plotArea.height + TRAFFIC_X_LABEL_OFFSET} textAnchor="middle">
        {formatTrafficTimestamp(value)}
      </text>
    </g>)}
    {yTicks.map((value) => <g
      key={value}
      data-traffic-value-tick={value}
      className={tickMotionClass}
      style={{ transform: `translateY(${scaleCoordinate({ value, domain: valueDomain, start: plotArea.y, length: plotArea.height, invert: true })}px)` }}
    >
      <line className="stroke-border/50" x1={plotArea.x} x2={plotArea.x + plotArea.width} />
      <text className="fill-muted-foreground text-xs" x={plotArea.x - TRAFFIC_Y_LABEL_OFFSET} dominantBaseline="middle" textAnchor="end">
        {formatter(value)}
      </text>
    </g>)}
  </g>
}

function TrafficLine({ dataKey }: { dataKey: string }) {
  return <Line
    id={`traffic-${dataKey}`}
    dataKey={dataKey}
    type="monotone"
    stroke={`var(--color-${dataKey})`}
    dot={false}
    activeDot={false}
    opacity={0}
    isAnimationActive={false}
    animateNewValues={false}
  />
}

export function TrafficPlot(props: TrafficPlotProps) {
  const hasData = props.data.some((point) => getTrafficTimestampValue(point) > 0)
  return <>
    <YAxis width={TRAFFIC_AXIS_WIDTH} domain={props.valueDomain} allowDataOverflow tick={false} tickLine={false} axisLine={false} tickFormatter={props.formatter} />
    <XAxis
      type="number"
      dataKey={getTrafficTimestampValue}
      domain={props.timeDomain}
      allowDataOverflow
      tick={false}
      tickLine={false}
      axisLine={false}
      tickFormatter={formatTrafficTimestamp}
    />
    <TrafficAxes timeDomain={props.timeDomain} valueDomain={props.valueDomain} formatter={props.formatter} hasData={hasData} />
    <TrafficLine dataKey={props.uploadKey} />
    <TrafficLine dataKey={props.downloadKey} />
    <TrafficSeries data={props.data} dataKey={props.uploadKey} timeDomain={props.timeDomain} valueDomain={props.valueDomain} />
    <TrafficSeries data={props.data} dataKey={props.downloadKey} timeDomain={props.timeDomain} valueDomain={props.valueDomain} />
  </>
}
