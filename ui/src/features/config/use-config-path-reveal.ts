import { useEffect, useRef } from "react"
import { useSearchParams } from "react-router-dom"

/** Consume ?path= query once and reveal it in a JSON editor. */
export function useConfigPathReveal(
  reveal: (path: string) => boolean,
  options: { ready?: boolean; section?: string } = {},
) {
  const [searchParams, setSearchParams] = useSearchParams()
  const handled = useRef(false)
  const ready = options.ready ?? true
  const path = searchParams.get("path")?.trim() || ""

  useEffect(() => {
    if (!ready || !path || handled.current) return
    const candidates = [path]
    const section = options.section?.trim()
    if (section) {
      if (path.startsWith(`${section}.`)) candidates.push(path.slice(section.length + 1))
      if (path.startsWith(`${section}[`)) candidates.push(path.slice(section.length))
      if (!path.startsWith(section)) candidates.push(`${section}.${path.replace(/^\./, "")}`)
    }
    const ok = candidates.some((candidate) => reveal(candidate))
    handled.current = true
    if (!ok) return
    const next = new URLSearchParams(searchParams)
    next.delete("path")
    setSearchParams(next, { replace: true })
  }, [options.section, path, ready, reveal, searchParams, setSearchParams])
}
