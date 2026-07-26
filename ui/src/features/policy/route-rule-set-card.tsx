import { useState } from "react"
import { CopyIcon, EllipsisIcon, PencilIcon, RefreshCwIcon, Trash2Icon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { headlessRules } from "@/features/policy/route-headless-rule-model"
import { summarizeRuleSet } from "@/features/policy/route-form-model"
import {
  resolveRuleSetErrorCode,
  ruleSetErrorHintKey,
  ruleSetUpdateErrorClipboardText,
} from "@/features/policy/ruleset-update-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import type { RuleSetStatusItem, RuleSetUpdateResult } from "@/lib/api/types"

interface RouteRuleSetCardProps {
  item: JsonObject
  status?: RuleSetStatusItem
  lastUpdate?: RuleSetUpdateResult
  updating?: boolean
  onEdit: () => void
  onCopy: () => void
  onDelete: () => void
  onUpdate?: () => void
}

function formatUpdatedAt(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function RuleSetUpdateError({
  result,
  tag,
}: {
  result: RuleSetUpdateResult
  tag: string
}) {
  const { t } = useTranslation()
  if (result.ok || !result.error?.trim()) return null
  const code = resolveRuleSetErrorCode(result)
  const copyError = () => {
    const payload = ruleSetUpdateErrorClipboardText(result)
    if (!payload) return
    void copyText(payload).then(
      () => toast.success(t("policy.route.ruleSetErrorCopied")),
      () => toast.error(t("policy.route.ruleSetErrorCopyFailed")),
    )
  }
  return (
    <div className="rounded-md border border-destructive/40 bg-destructive/5 px-2.5 py-1.5">
      {code && code !== "unknown" ? (
        <Badge variant="outline" className="mb-1 font-mono text-[10px]">{code}</Badge>
      ) : null}
      <p className="line-clamp-2 text-xs text-destructive" title={result.error}>{result.error}</p>
      <p className="mt-1 line-clamp-2 text-[11px] text-muted-foreground">{t(ruleSetErrorHintKey(code))}</p>
      <Button
        type="button"
        size="xs"
        variant="outline"
        className="mt-1.5 h-7"
        aria-label={`${t("policy.route.copyRuleSetError")}: ${tag}`}
        onClick={copyError}
      >
        <CopyIcon data-icon="inline-start" />
        {t("policy.route.copyRuleSetError")}
      </Button>
    </div>
  )
}

export function RouteRuleSetCard({
  item,
  status,
  lastUpdate,
  updating,
  onEdit,
  onCopy,
  onDelete,
  onUpdate,
}: RouteRuleSetCardProps) {
  const { t } = useTranslation()
  const [deleting, setDeleting] = useState(false)
  const tag = typeof item.tag === "string" && item.tag ? item.tag : t("policy.route.unnamed")
  const summary = summarizeRuleSet(item)
  const detail = String(item.type ?? "inline") === "inline"
    ? t("policy.route.inlineRuleCount", { count: headlessRules(item).length })
    : summary.detail || t("policy.route.ruleSetLocationMissing")
  const confirmDelete = () => {
    setDeleting(false)
    onDelete()
  }
  const updatable = Boolean(status?.updatable && onUpdate)
  const updatedLabel = formatUpdatedAt(status?.last_updated)
  return (
    <>
      <Card size="sm">
        <CardHeader className="min-w-0 gap-1.5">
          <CardTitle className="truncate" title={tag}>{tag}</CardTitle>
          <CardDescription className="min-w-0 line-clamp-2 break-words">
            {detail}
          </CardDescription>
          <CardAction className="flex flex-wrap items-center justify-end gap-1">
            {updatable ? (
              <Button
                variant="outline"
                size="xs"
                disabled={updating}
                aria-label={t("policy.route.updateRuleSet", { tag })}
                onClick={onUpdate}
              >
                <RefreshCwIcon data-icon="inline-start" className={updating ? "animate-spin" : undefined} />
                {t("policy.route.update")}
              </Button>
            ) : null}
            <Button variant="outline" size="xs" aria-label={t("policy.route.editRuleSet", { tag })} onClick={onEdit}>
              <PencilIcon data-icon="inline-start" />
              {t("policy.route.edit")}
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent className="flex flex-col gap-1.5">
          <div className="flex flex-wrap gap-1.5">
            <Badge variant="secondary">{summary.type}</Badge>
            {status?.builtin ? <Badge variant="outline">{t("policy.route.builtin")}</Badge> : null}
            {status?.update_interval ? (
              <Badge variant="outline">{t("policy.route.intervalBadge", { interval: status.update_interval })}</Badge>
            ) : null}
          </div>
          {updatedLabel ? (
            <p className="text-muted-foreground text-xs">{t("policy.route.lastUpdated", { time: updatedLabel })}</p>
          ) : null}
          {status?.note ? <p className="text-muted-foreground text-xs">{status.note}</p> : null}
          {lastUpdate ? <RuleSetUpdateError result={lastUpdate} tag={tag} /> : null}
        </CardContent>
        <CardFooter className="justify-between gap-2">
          <div className="hidden gap-1 sm:flex">
            <Button variant="outline" size="icon-xs" aria-label={t("policy.route.copyRuleSet", { tag })} onClick={onCopy}>
              <CopyIcon data-icon="inline-start" />
            </Button>
            <Button
              variant="destructive"
              size="icon-xs"
              aria-label={t("policy.route.deleteRuleSet", { tag })}
              onClick={() => setDeleting(true)}
            >
              <Trash2Icon data-icon="inline-start" />
            </Button>
          </div>
          <div className="sm:hidden">
            <DropdownMenu>
              <DropdownMenuTrigger
                render={<Button variant="outline" size="icon-xs" aria-label={t("policy.route.moreRuleSetActions", { tag })} />}
              >
                <EllipsisIcon data-icon="inline-start" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuGroup>
                  {updatable ? (
                    <DropdownMenuItem disabled={updating} onClick={onUpdate}>
                      <RefreshCwIcon />
                      {t("policy.route.update")}
                    </DropdownMenuItem>
                  ) : null}
                  <DropdownMenuItem onClick={onCopy}><CopyIcon />{t("policy.route.copy")}</DropdownMenuItem>
                  <DropdownMenuItem variant="destructive" onClick={() => setDeleting(true)}>
                    <Trash2Icon />
                    {t("policy.route.delete")}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </CardFooter>
      </Card>
      <AlertDialog open={deleting} onOpenChange={setDeleting}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("policy.route.deleteRuleSetTitle", { tag })}</AlertDialogTitle>
            <AlertDialogDescription>{t("policy.route.deleteDescription")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("policy.route.cancel")}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={confirmDelete}>
              {t("policy.route.confirmDelete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
