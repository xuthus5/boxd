import { useState } from "react"
import { ArrowDownIcon, ArrowUpIcon, CopyIcon, EllipsisIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { routeMatchFields, summarizeRouteRule } from "@/features/policy/route-form-model"

interface RouteRuleCardProps {
  index: number
  item: JsonObject
  first: boolean
  last: boolean
  onEdit: () => void
  onCopy: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  onDelete: () => void
}

function DesktopActions({ index, first, last, onCopy, onMoveUp, onMoveDown, onDelete }: Omit<RouteRuleCardProps, "item" | "onEdit">) {
  const { t } = useTranslation()
  const number = index + 1
  return <div className="hidden gap-1 sm:flex">
    <Button variant="outline" size="icon-xs" aria-label={t("policy.route.copyRule", { index: number })} onClick={onCopy}><CopyIcon data-icon="inline-start" /></Button>
    <Button variant="outline" size="icon-xs" aria-label={t("policy.route.moveRuleUp", { index: number })} disabled={first} onClick={onMoveUp}><ArrowUpIcon data-icon="inline-start" /></Button>
    <Button variant="outline" size="icon-xs" aria-label={t("policy.route.moveRuleDown", { index: number })} disabled={last} onClick={onMoveDown}><ArrowDownIcon data-icon="inline-start" /></Button>
    <Button variant="destructive" size="icon-xs" aria-label={t("policy.route.deleteRule", { index: number })} onClick={onDelete}><Trash2Icon data-icon="inline-start" /></Button>
  </div>
}

function MobileActions(props: Omit<RouteRuleCardProps, "item" | "onEdit">) {
  const { t } = useTranslation()
  const { index, first, last, onCopy, onMoveUp, onMoveDown, onDelete } = props
  return <div className="sm:hidden"><DropdownMenu>
    <DropdownMenuTrigger render={<Button variant="outline" size="icon-xs" aria-label={t("policy.route.moreRuleActions", { index: index + 1 })} />}><EllipsisIcon data-icon="inline-start" /></DropdownMenuTrigger>
    <DropdownMenuContent align="end"><DropdownMenuGroup>
      <DropdownMenuItem onClick={onCopy}><CopyIcon />{t("policy.route.copy")}</DropdownMenuItem>
      <DropdownMenuItem disabled={first} onClick={onMoveUp}><ArrowUpIcon />{t("policy.route.moveUp")}</DropdownMenuItem>
      <DropdownMenuItem disabled={last} onClick={onMoveDown}><ArrowDownIcon />{t("policy.route.moveDown")}</DropdownMenuItem>
      <DropdownMenuItem variant="destructive" onClick={onDelete}><Trash2Icon />{t("policy.route.delete")}</DropdownMenuItem>
    </DropdownMenuGroup></DropdownMenuContent>
  </DropdownMenu></div>
}

export function RouteRuleCard(props: RouteRuleCardProps) {
  const { t } = useTranslation()
  const { index, item, first, last, onEdit, onCopy, onMoveUp, onMoveDown, onDelete } = props
  const [deleting, setDeleting] = useState(false)
  const number = index + 1
  const matchLabels = new Map<string, string>(
    routeMatchFields.map((field) => [field.path, t(`policy.route.${field.label}`)]),
  )
  const summary = summarizeRouteRule(item, { matchLabel: (path) => matchLabels.get(path) ?? path })
  const type = String(item.type ?? "default")
  const confirmDelete = () => { setDeleting(false); onDelete() }
  return <>
    <Card size="sm">
      <CardHeader className="min-w-0"><CardTitle>{t("policy.route.ruleCardTitle", { index: number })}</CardTitle><CardDescription>{type}</CardDescription>
        <CardAction><Button variant="outline" size="xs" aria-label={t("policy.route.editRule", { index: number })} onClick={onEdit}><PencilIcon data-icon="inline-start" />{t("policy.route.edit")}</Button></CardAction>
      </CardHeader>
      <CardContent><div className="flex flex-wrap gap-2">
        {summary.matches.slice(0, 3).map((match, matchIndex) => <Badge key={`${match}:${matchIndex}`} variant="secondary">{match}</Badge>)}
        <Badge>{summary.action}</Badge>
      </div></CardContent>
      <CardFooter className="justify-between gap-2">
        <DesktopActions index={index} first={first} last={last} onCopy={onCopy} onMoveUp={onMoveUp} onMoveDown={onMoveDown} onDelete={() => setDeleting(true)} />
        <MobileActions index={index} first={first} last={last} onCopy={onCopy} onMoveUp={onMoveUp} onMoveDown={onMoveDown} onDelete={() => setDeleting(true)} />
      </CardFooter>
    </Card>
    <AlertDialog open={deleting} onOpenChange={setDeleting}><AlertDialogContent>
      <AlertDialogHeader><AlertDialogTitle>{t("policy.route.deleteRuleTitle", { index: number })}</AlertDialogTitle><AlertDialogDescription>{t("policy.route.deleteDescription")}</AlertDialogDescription></AlertDialogHeader>
      <AlertDialogFooter><AlertDialogCancel>{t("policy.route.cancel")}</AlertDialogCancel><AlertDialogAction variant="destructive" onClick={confirmDelete}>{t("policy.route.confirmDelete")}</AlertDialogAction></AlertDialogFooter>
    </AlertDialogContent></AlertDialog>
  </>
}
