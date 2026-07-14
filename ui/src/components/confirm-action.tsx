import { useState, type ComponentProps, type ReactElement } from "react"
import { useTranslation } from "react-i18next"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"

interface ConfirmActionProps {
  trigger: ReactElement
  title: string
  description: string
  confirmLabel: string
  onConfirm: () => void
  confirmVariant?: ComponentProps<typeof Button>["variant"]
}

export function ConfirmAction({
  trigger,
  title,
  description,
  confirmLabel,
  onConfirm,
  confirmVariant = "default",
}: ConfirmActionProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const confirm = () => {
    setOpen(false)
    onConfirm()
  }

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={trigger} />
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction variant={confirmVariant} onClick={confirm}>
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
