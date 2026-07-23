import { useMutation } from "@tanstack/react-query"
import { CopyIcon, EllipsisIcon, GaugeIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { formatLatency } from "@/features/nodes/node-format"
import { latencyBadgeVariant, latencyTone, latencyToneClass } from "@/features/nodes/latency-style"
import { dnsProbeInput, isDNSServerProbeable } from "@/features/policy/dns-probe"
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

function ProbeBadge({ result }: { result?: DNSProbeResult }) {
  const { t } = useTranslation()
  if (!result) return null
  if (!result.success) {
    return <Badge variant="destructive">{result.error || t("policy.dns.probeFailed")}</Badge>
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
      if (!result.success) toast.error(result.error || t("policy.dns.probeFailed"))
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const confirmDelete = () => { setDeleting(false); onDelete() }
  return <><Card size="sm"><CardHeader className="min-w-0"><CardTitle>{tag}</CardTitle>
    <CardDescription>{inferDNSServerType(item)}</CardDescription><CardAction className="flex items-center gap-2">
      <ProbeBadge result={probeResult} />
      <Button variant="outline" size="xs" disabled={!probeable || probeMutation.isPending}
        aria-label={t("policy.dns.probeServer", { tag })} onClick={() => probeMutation.mutate()}>
        <GaugeIcon data-icon="inline-start" />{probeMutation.isPending ? t("policy.dns.probing") : t("policy.dns.probe")}
      </Button>
      <Button variant="outline" size="xs" aria-label={t("policy.dns.editServer", { tag })} onClick={onEdit}>
        <PencilIcon data-icon="inline-start" />{t("policy.dns.edit")}
      </Button></CardAction></CardHeader>
    <CardContent><div className="flex flex-wrap gap-2"><Badge>{summary.type}</Badge>
      {summary.detail ? <Badge variant="secondary">{summary.detail}</Badge> : null}</div></CardContent>
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
