import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import type { SetupStep } from "@/features/dashboard/setup-checklist"
import { SetupRunAction } from "@/features/setup/setup-run-action"

interface SetupStepsProps {
  steps: SetupStep[]
  available: boolean
  onModules: () => void
  onAccess: () => void
}

function SetupStepAction({ step, props }: { step: SetupStep; props: SetupStepsProps }) {
  const { t } = useTranslation()
  if (step.id === "nodes") return <Link className={buttonVariants({ variant: "outline", size: "sm" })} to={step.href}>{t("setup.import")}</Link>
  if (step.id === "kernel") return <SetupRunAction running={step.done} ready={props.available && props.steps.filter((item) => item.id !== "kernel").every((item) => item.done)} />
  return <Button variant="outline" size="sm" disabled={!props.available} onClick={step.id === "modules" ? props.onModules : props.onAccess}>
    {t(step.id === "modules" ? "setup.manage" : "setup.access")}
  </Button>
}

export function SetupSteps(props: SetupStepsProps) {
  const { t } = useTranslation()
  return <ol className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
    {props.steps.map((step, index) => <li key={step.id} className="flex flex-col gap-3 rounded-md border p-3" data-setup-step={step.id}>
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium">{index + 1}. {t(`setup.${step.id}Step`)}</p>
        <Badge variant={step.done ? "secondary" : "outline"}>{t(step.done ? (step.id === "kernel" ? "setup.running" : "setup.configured") : "setup.pending")}</Badge>
      </div>
      <SetupStepAction step={step} props={props} />
    </li>)}
  </ol>
}
