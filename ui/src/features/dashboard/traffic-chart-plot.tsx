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
const TRAFFIC_ORIGIN_BUCKET_MS = 60 * 60 * 1000
const EMPTY_CURVE_POINTS: readonly TrafficCurvePoint[] = []

type TrafficPlotArea = NonNullable<ReturnType<typeof usePlotArea>>

interface TrafficCurvePoint {
  x: number | null
  y: number | null
  payload?: { timestamp?: unknown }
}

interface SmoothTrafficCurveProps extends Omit<CurveProps, "points"> {
  points?: readonly TrafficCurvePoint[]
  translateX?: number
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
  timeOrigin: number
  timeDomain: TrafficDomain
  valueDomain: TrafficDomain
}

export function SmoothTrafficCurve({
  points = EMPTY_CURVE_POINTS,
  translateX = 0,
  transitionDuration = TRAFFIC_UPDATE_DURATION_MS,
  ...curveProps
}: SmoothTrafficCurveProps) {
  const motionRef = useRef<SVGGElement | null>(null)
  const hasPoints = points.length > 0
  useEffect(() => {
    const motion = motionRef.current
    if (!motion) return
    if (!hasPoints) {
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
  }, [hasPoints])

  return <g
    ref={motionRef}
    className="ease-linear motion-reduce:transition-none will-change-transform"
    style={{ transform: `translateX(${translateX}px)`, transitionDuration: `${transitionDuration}ms` }}
  >
    <Curve {...curveProps} points={points} />
  </g>
}

function samePlotArea(left: TrafficPlotArea | undefined, right: TrafficPlotArea | undefined) {
  return left?.x === right?.x
    && left?.y === right?.y
    && left?.width === right?.width
    && left?.height === right?.height
}

function useStablePlotArea() {
  const current = usePlotArea()
  const [stable, setStable] = useState(current)
  if (!current || samePlotArea(current, stable)) return stable
  setStable(current)
  return current
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
  const windowSize = input.timeDomain[1] - input.timeDomain[0]
  const pixelsPerMillisecond = windowSize > 0 ? input.plotArea.width / windowSize : 0
  for (const point of input.data) {
    const timestamp = getTrafficTimestampValue(point)
    const value = getTrafficPointValue(point, input.dataKey)
    if (value === undefined || timestamp < input.timeDomain[0] || timestamp > input.timeDomain[1]) continue
    points.push({
      x: (timestamp - input.timeOrigin) * pixelsPerMillisecond,
      y: scaleCoordinate({ value, domain: input.valueDomain, start: input.plotArea.y, length: input.plotArea.height, invert: true }),
      payload: point,
    })
  }
  return points
}

function trafficCurveTranslateX(input: Pick<TrafficSeriesProps, "timeDomain" | "timeOrigin"> & { plotArea: TrafficPlotArea }) {
  const windowSize = input.timeDomain[1] - input.timeDomain[0]
  if (windowSize <= 0) return input.plotArea.x
  const pixelsPerMillisecond = input.plotArea.width / windowSize
  return input.plotArea.x - ((input.timeDomain[0] - input.timeOrigin) * pixelsPerMillisecond)
}

function TrafficSeries(props: TrafficSeriesProps) {
  const { data, dataKey, timeDomain, timeOrigin, valueDomain } = props
  const plotArea = useStablePlotArea()
  const geometry = useMemo(
    () => plotArea ? {
      points: scaledCurvePoints({ data, dataKey, timeDomain, timeOrigin, valueDomain, plotArea }),
      translateX: trafficCurveTranslateX({ timeDomain, timeOrigin, plotArea }),
    } : undefined,
    [data, dataKey, plotArea, timeDomain, timeOrigin, valueDomain],
  )
  return <SmoothTrafficCurve
    className="traffic-chart-curve"
    data-series={dataKey}
    points={geometry?.points}
    translateX={geometry?.translateX}
    type="monotone"
    fill="none"
    stroke={`var(--color-${dataKey})`}
    strokeLinecap="round"
    strokeLinejoin="round"
  />
}

function trafficTimeOrigin(data: readonly TrafficChartPoint[]) {
  const timestamp = data.map(getTrafficTimestampValue).find((value) => value > 0)
  return timestamp === undefined ? undefined : Math.floor(timestamp / TRAFFIC_ORIGIN_BUCKET_MS) * TRAFFIC_ORIGIN_BUCKET_MS
}

function useTrafficTimeOrigin(data: readonly TrafficChartPoint[]) {
  const candidate = trafficTimeOrigin(data)
  const [origin, setOrigin] = useState(candidate)
  if (origin !== undefined || candidate === undefined) return origin ?? 0
  setOrigin(candidate)
  return candidate
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
  const [initialDomainEnd] = useState(() => timeDomain[1])
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
  const timeOrigin = useTrafficTimeOrigin(props.data)
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
    <TrafficSeries data={props.data} dataKey={props.uploadKey} timeDomain={props.timeDomain} timeOrigin={timeOrigin} valueDomain={props.valueDomain} />
    <TrafficSeries data={props.data} dataKey={props.downloadKey} timeDomain={props.timeDomain} timeOrigin={timeOrigin} valueDomain={props.valueDomain} />
  </>
}
