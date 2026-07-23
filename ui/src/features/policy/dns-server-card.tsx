import { useMutation } from "@tanstack/react-query"
import { CopyIcon, EllipsisIcon, GaugeIcon, PencilIcon, ScrollTextIcon, Trash2Icon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { formatLatency } from "@/features/nodes/node-format"
import { latencyBadgeVariant, latencyTone, latencyToneClass } from "@/features/nodes/latency-style"
import { buildDNSHref } from "@/features/policy/dns-filter"
import {
  dnsProbeErrorClipboardText,
  dnsProbeErrorHintKey,
  resolveDNSProbeErrorCode,
} from "@/features/policy/dns-probe-error"
import { dnsProbeInput, isDNSServerProbeable } from "@/features/policy/dns-probe"
import { copyText } from "@/features/proxy/copy-tag-button"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import { inferDNSServerType, summarizeDNSServer } from "@/features/policy/dns-form-model"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { api } from "@/lib/api/endpoints"
import type { DNSProbeResult } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface DNSServerCardProps {
  item: JsonObject
  onEdit: () => void
  onCopy: () => void
  onDelete: () => void
  probeResult?: DNSProbeResult
  onProbeResult?: (result: DNSProbeResult) => void
}

function copyProbeError(result: DNSProbeResult, t: (key: string) => string) {
  const payload = dnsProbeErrorClipboardText(result)
  if (!payload) return
  void copyText(payload).then(
    () => toast.success(t("policy.dns.probeErrorCopied")),
    () => toast.error(t("policy.dns.probeErrorCopyFailed")),
  )
}

function ProbeBadge({ result }: { result?: DNSProbeResult }) {
  const { t } = useTranslation()
  if (!result) return null
  if (!result.success) {
    const label = result.error?.trim() || t("policy.dns.probeFailed")
    const code = resolveDNSProbeErrorCode(result)
    const hint = t(dnsProbeErrorHintKey(code))
    const title = code ? `${code} · ${label}\n${hint}` : `${label}\n${hint}`
    return (
      <Badge
        variant="destructive"
        className="max-w-[10rem] cursor-pointer truncate"
        title={title}
        role="button"
        tabIndex={0}
        aria-label={`${t("policy.dns.copyProbeError")}: ${result.tag || ""} ${label}`}
        onClick={() => copyProbeError(result, t)}
        onKeyDown={(event) => {
          if (event.key !== "Enter" && event.key !== " ") return
          event.preventDefault()
          copyProbeError(result, t)
        }}
      >
        {code && code !== "unknown" && code !== label ? `${code}: ${label}` : label}
      </Badge>
    )
  }
  const tone = latencyTone(result.latency_ms, true)
  const label = result.latency_ms === undefined ? t("common.normal") : formatLatency(result.latency_ms)
  return <Badge variant={latencyBadgeVariant(tone)} className={cn(latencyToneClass(tone))}>{label}</Badge>
}

export function DNSServerCard({ item, onEdit, onCopy, onDelete, probeResult, onProbeResult }: DNSServerCardProps) {
  const { t } = useTranslation()
  const [deleting, setDeleting] = useState(false)
  const tag = typeof item.tag === "string" && item.tag ? item.tag : t("policy.dns.unnamed")
  const probeable = isDNSServerProbeable(item)
  const summary = summarizeDNSServer(item, {
    path: (value) => t("policy.dns.summaryPath", { value }), predefined: (count) => t("policy.dns.summaryPredefined", { count }),
    ipv4: (value) => t("policy.dns.summaryIPv4", { value }), ipv6: (value) => t("policy.dns.summaryIPv6", { value }),
    tag: (value) => t("policy.dns.summaryTag", { value }), detour: (value) => t("policy.dns.summaryDetour", { value }),
    strategy: (value) => t("policy.dns.summaryStrategy", { value }),
  })
  const probeMutation = useMutation({
    mutationFn: async () => {
      const input = dnsProbeInput(item)
      if (!input) throw new Error(t("policy.dns.probeUnsupported"))
      return api.runtime.probeDNS(input)
    },
    onSuccess: (result) => {
      onProbeResult?.(result)
      if (!result.success) {
        const code = resolveDNSProbeErrorCode(result)
        const detail = result.error || t("policy.dns.probeFailed")
        toast.error(code ? `${code}: ${detail}` : detail, {
          description: t(dnsProbeErrorHintKey(code)),
        })
      }
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const confirmDelete = () => { setDeleting(false); onDelete() }
  return <><Card size="sm"><CardHeader className="min-w-0 gap-1.5">
    <CardTitle className="truncate" title={tag}>{tag}</CardTitle>
    <CardDescription className="truncate">{inferDNSServerType(item)}</CardDescription>
    <CardAction className="flex flex-wrap items-center justify-end gap-1">
      <ProbeBadge result={probeResult} />
      <Button variant="outline" size="xs" disabled={!probeable || probeMutation.isPending}
        aria-label={t("policy.dns.probeServer", { tag })} onClick={() => probeMutation.mutate()}>
        <GaugeIcon data-icon="inline-start" />{probeMutation.isPending ? t("policy.dns.probing") : t("policy.dns.probe")}
      </Button>
      <Button variant="outline" size="xs" aria-label={t("policy.dns.editServer", { tag })} onClick={onEdit}>
        <PencilIcon data-icon="inline-start" />{t("policy.dns.edit")}
      </Button>
    </CardAction>
  </CardHeader>
    <CardContent className="flex flex-col gap-1.5">
      <div className="flex flex-wrap gap-1.5">
        <Badge>{summary.type}</Badge>
        {summary.detail ? <Badge variant="secondary" className="max-w-full truncate">{summary.detail}</Badge> : null}
      </div>
      {typeof item.tag === "string" && item.tag ? (
        <div className="flex flex-wrap gap-1.5">
          <Link
            to={buildDNSHref({ rules: item.tag })}
            aria-label={`${t("policy.dns.viewDNSRules")}: ${item.tag}`}
            className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
          >
            {t("policy.dns.viewDNSRules")}
          </Link>
          <Link
            to={buildLogsHref({ query: item.tag, preset: "dns" })}
            aria-label={`${t("policy.dns.viewLogs")}: ${item.tag}`}
            className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
          >
            <ScrollTextIcon data-icon="inline-start" />
            {t("policy.dns.viewLogs")}
          </Link>
        </div>
      ) : null}
    </CardContent>
    <CardFooter className="justify-between gap-2"><div className="hidden gap-1 sm:flex">
      <Button variant="outline" size="icon-xs" aria-label={t("policy.dns.copyServer", { tag })} onClick={onCopy}><CopyIcon data-icon="inline-start" /></Button>
      <Button variant="destructive" size="icon-xs" aria-label={t("policy.dns.deleteServer", { tag })} onClick={() => setDeleting(true)}><Trash2Icon data-icon="inline-start" /></Button>
    </div><div className="sm:hidden"><DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline" size="icon-xs" aria-label={t("policy.dns.moreServerActions", { tag })} />}><EllipsisIcon data-icon="inline-start" /></DropdownMenuTrigger>
      <DropdownMenuContent align="end"><DropdownMenuGroup>
        <DropdownMenuItem onClick={onCopy}><CopyIcon />{t("policy.dns.copy")}</DropdownMenuItem>
        <DropdownMenuItem variant="destructive" onClick={() => setDeleting(true)}><Trash2Icon />{t("policy.dns.delete")}</DropdownMenuItem>
      </DropdownMenuGroup></DropdownMenuContent>
    </DropdownMenu></div></CardFooter>
  </Card><AlertDialog open={deleting} onOpenChange={setDeleting}><AlertDialogContent>
    <AlertDialogHeader><AlertDialogTitle>{t("policy.dns.deleteServerTitle", { tag })}</AlertDialogTitle>
      <AlertDialogDescription>{t("policy.dns.deleteDescription")}</AlertDialogDescription></AlertDialogHeader>
    <AlertDialogFooter><AlertDialogCancel>{t("policy.dns.cancel")}</AlertDialogCancel>
      <AlertDialogAction variant="destructive" onClick={confirmDelete}>{t("policy.dns.confirmDelete")}</AlertDialogAction></AlertDialogFooter>
  </AlertDialogContent></AlertDialog></>
}
