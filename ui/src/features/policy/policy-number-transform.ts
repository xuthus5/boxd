import {
  setPolicyPath,
  type PolicyFieldKind,
  type PolicyFieldTransform,
} from "@/features/policy/policy-form-model"

export interface PolicyNumberConstraint {
  kind: "integer" | "integer-list" | "mark"
  maximum: number
  allowed?: readonly number[]
  fieldKind?: PolicyFieldKind
}

export type PolicyNumberConstraints = Record<string, PolicyNumberConstraint>

function decimalInteger(raw: string, constraint: PolicyNumberConstraint): number | null {
  if (!/^\d+$/.test(raw)) return null
  const value = Number(raw)
  if (!Number.isSafeInteger(value) || value > constraint.maximum) return null
  return !constraint.allowed || constraint.allowed.includes(value) ? value : null
}

function integerList(raw: string, constraint: PolicyNumberConstraint): number[] | null {
  const tokens = raw.split(/[\n,]/).map((token) => token.trim()).filter(Boolean)
  if (tokens.length === 0) return []
  const values = tokens.map((token) => decimalInteger(token, constraint))
  return values.some((value) => value === null) ? null : values as number[]
}

function stringMark(raw: string, maximum: number): boolean {
  const normalized = /^0[0-7]+$/.test(raw) ? `0o${raw.slice(1)}` : raw
  if (!/^0(?:[xX][\da-fA-F]+|[bB][01]+|[oO][0-7]+)$/.test(normalized)) return false
  try {
    return BigInt(normalized) <= BigInt(maximum)
  } catch (error) {
    void error
    return false
  }
}

export function createPolicyNumberTransform(constraints: PolicyNumberConstraints): PolicyFieldTransform {
  return (object, field, raw) => {
    const constraint = constraints[field.path]
    if (!constraint || constraint.fieldKind && constraint.fieldKind !== field.kind) return undefined
    const token = raw.trim()
    if (!token) return setPolicyPath(object, field.path, undefined)
    if (constraint.kind === "mark" && stringMark(token, constraint.maximum)) {
      return setPolicyPath(object, field.path, token)
    }
    if (constraint.kind === "integer-list") {
      const values = integerList(raw, constraint)
      return values === null ? null : setPolicyPath(object, field.path, values)
    }
    const value = decimalInteger(token, constraint)
    return value === null ? null : setPolicyPath(object, field.path, value)
  }
}
