import { RefreshCwIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Button, buttonVariants } from "@/components/ui/button"
import { CardFooter } from "@/components/ui/card"
import { ruleSetHealthHref } from "@/features/dashboard/rule-set-health"
import { cn } from "@/lib/utils"

export interface RuleSetUpdateAction {
  isPending: boolean
  mutate: () => void
}

export function RuleSetHealthFooter({ updatable, updateAction }: { updatable: number; updateAction: RuleSetUpdateAction }) {
  const { t } = useTranslation()
  return <CardFooter className="flex flex-wrap gap-1.5">
    <Button type="button" variant="outline" size="sm" className="h-8" disabled={updateAction.isPending || updatable === 0} onClick={updateAction.mutate}>
      <RefreshCwIcon data-icon="inline-start" className={updateAction.isPending ? "animate-spin" : undefined} />
      {t(updateAction.isPending ? "ruleSetHealth.updating" : "ruleSetHealth.updateAll")}
    </Button>
    <Link to={ruleSetHealthHref()} className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}>
      {t("ruleSetHealth.openRoute")}
    </Link>
    <Link to="/observability/logs?tab=application" className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-8")}>
      {t("ruleSetHealth.openLogs")}
    </Link>
  </CardFooter>
}
