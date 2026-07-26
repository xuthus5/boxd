import { useState } from "react"
import { Line, XAxis, YAxis } from "recharts"

import {
  TRAFFIC_AXIS_WIDTH,
  TRAFFIC_UPDATE_DURATION_MS,
  formatTrafficTimestamp,
  getTrafficTimestampValue,
  type TrafficChartPoint,
  type TrafficDomain,
} from "@/features/dashboard/traffic-chart-model"
import {
  useStableTrafficPlotArea,
  useTrafficTimeOrigin,
} from "@/features/dashboard/traffic-chart-layout"
import { TrafficSeries } from "@/features/dashboard/traffic-chart-series"

const TRAFFIC_TIME_TICK_STEPS = [15_000, 30_000, 60_000] as const
const TRAFFIC_MIN_TIME_TICK_SPACING = 80
const TRAFFIC_VALUE_TICK_COUNT = 4
const TRAFFIC_X_LABEL_OFFSET = 20
const TRAFFIC_Y_LABEL_OFFSET = 8

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

function scaleCoordinate(input: {
  value: number
  domain: TrafficDomain
  start: number
  length: number
  invert?: boolean
}) {
  const [minimum, maximum] = input.domain
  const progress = (input.value - minimum) / Math.max(maximum - minimum, Number.EPSILON)
  return input.start + (input.invert ? 1 - progress : progress) * input.length
}

function domainTicks(domain: TrafficDomain, count: number) {
  const step = (domain[1] - domain[0]) / count
  return Array.from({ length: count + 1 }, (_, index) => domain[0] + step * index)
}

function timeTickStep(width: number) {
  const tickCount = Math.max(1, Math.floor(width / TRAFFIC_MIN_TIME_TICK_SPACING))
  const windowSize = TRAFFIC_TIME_TICK_STEPS[TRAFFIC_TIME_TICK_STEPS.length - 1]
  const desiredStep = windowSize / tickCount
  return TRAFFIC_TIME_TICK_STEPS.find((step) => step >= desiredStep)!
}

function timeTicks(domain: TrafficDomain, width: number) {
  const ticks: number[] = []
  const step = timeTickStep(width)
  const first = Math.ceil(domain[0] / step) * step
  for (let value = first; value <= domain[1]; value += step) ticks.push(value)
  return ticks
}

function TrafficAxes({ timeDomain, valueDomain, formatter, hasData }: TrafficAxesProps) {
  const plotArea = useStableTrafficPlotArea()
  const [initialDomainEnd] = useState(() => timeDomain[1])
  if (!plotArea || !hasData) return null
  const xTicks = timeTicks(timeDomain, plotArea.width)
  const yTicks = domainTicks(valueDomain, TRAFFIC_VALUE_TICK_COUNT)
  const tickMotionClass = timeDomain[1] === initialDomainEnd
    ? undefined
    : "transition-transform ease-linear motion-reduce:transition-none"
  return <g aria-hidden="true" pointerEvents="none">
    {xTicks.map((value) => <g
      key={value}
      data-traffic-time-tick={value}
      className={tickMotionClass}
      style={{
        transform: `translateX(${scaleCoordinate({ value, domain: timeDomain, start: plotArea.x, length: plotArea.width })}px)`,
        transitionDuration: `${TRAFFIC_UPDATE_DURATION_MS}ms`,
      }}
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
      style={{
        transform: `translateY(${scaleCoordinate({ value, domain: valueDomain, start: plotArea.y, length: plotArea.height, invert: true })}px)`,
        transitionDuration: `${TRAFFIC_UPDATE_DURATION_MS}ms`,
      }}
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
