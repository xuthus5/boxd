/** Validation helpers for proxy form numeric fields. */

export function parseFiniteNumber(raw: string): number | undefined | null {
  const trimmed = raw.trim()
  if (!trimmed) return undefined
  const value = Number(trimmed)
  if (!Number.isFinite(value)) return null
  return value
}

export function parseFiniteNumberList(raw: string): number[] | undefined | null {
  const parts = raw
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
  if (parts.length === 0) return undefined
  const numbers = parts.map((item) => Number(item))
  if (!numbers.every(Number.isFinite)) return null
  return numbers
}

export function isNumericFieldKind(kind: string | undefined): kind is "number" | "number-list" {
  return kind === "number" || kind === "number-list"
}

export function isNumericFieldRawValid(kind: string | undefined, raw: string): boolean {
  if (kind === "number") return parseFiniteNumber(raw) !== null
  if (kind === "number-list") return parseFiniteNumberList(raw) !== null
  return true
}
