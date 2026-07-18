export function formatLatency(latency: number) {
  return `${latency < 10 ? latency.toFixed(1) : latency.toFixed(0)} ms`
}
