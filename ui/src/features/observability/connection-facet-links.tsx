import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import {
  facetHref,
  type ConnectionFacetLinkField,
} from "@/features/observability/connection-facets"
import { cn } from "@/lib/utils"

export function FacetLink({
  field,
  value,
  label,
  className,
}: {
  field: ConnectionFacetLinkField
  value?: string
  label: string
  className?: string
}) {
  const href = facetHref(field, value)
  const textValue = value?.trim() || "—"
  if (!href) return <span className={className} title={textValue}>{textValue}</span>
  return (
    <Link
      to={href}
      className={cn("underline-offset-4 hover:underline", className)}
      title={textValue}
      aria-label={`${label}: ${textValue}`}
    >
      {textValue}
    </Link>
  )
}

export function MetaChip({
  label,
  value,
  field,
}: {
  label: string
  value: string
  field?: ConnectionFacetLinkField
}) {
  if (!value || value === "—") return null
  const href = field ? facetHref(field, value) : ""
  const content = <>{label}: {value}</>
  if (!href) {
    return <Badge variant="outline" className="max-w-full truncate font-normal" title={value}>{content}</Badge>
  }
  return (
    <Badge variant="outline" className="max-w-full truncate font-normal p-0" title={value}>
      <Link to={href} className="block max-w-full truncate px-2 py-0.5" aria-label={`${label}: ${value}`}>
        {content}
      </Link>
    </Badge>
  )
}
