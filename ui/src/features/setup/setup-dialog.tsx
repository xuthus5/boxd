import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { SetupCapabilities } from "@/features/setup/setup-capabilities"
import { SetupOptions } from "@/features/setup/setup-options"
import { SetupPreview } from "@/features/setup/setup-preview"
import { useSetupActions } from "@/features/setup/use-setup-actions"
import type { SetupStatus } from "@/lib/api/setup"

export function SetupDialog({ status, onClose }: { status: SetupStatus; onClose: () => void }) {
  const { t } = useTranslation()
  const state = useSetupActions(status, onClose)
  return <Dialog open onOpenChange={(open) => { if (!open && !state.busy) onClose() }}>
    <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl" showCloseButton={!state.busy}>
      <DialogHeader>
        <DialogTitle>{t("setup.dialogTitle")}</DialogTitle>
        <DialogDescription>{t("setup.dialogDescription")}</DialogDescription>
      </DialogHeader>
      <SetupCapabilities capabilities={status.capabilities} />
      <SetupOptions status={status} input={state.input} disabled={state.busy} onChange={state.change} />
      {state.preview ? <SetupPreview preview={state.preview} /> : null}
      <ConfigSaveErrorAlert error={state.errors.saveError} onDismiss={state.errors.clearSaveError} />
      <DialogFooter>
        <Button variant="outline" disabled={state.busy} onClick={onClose}>{t("common.cancel")}</Button>
        <Button variant="outline" disabled={state.busy || !state.canPreview} onClick={state.makePreview}>
          {t(state.previewing ? "setup.previewing" : "setup.preview")}
        </Button>
        <Button disabled={state.busy || !state.preview} onClick={state.apply}>{t(state.applying ? "setup.applying" : "setup.apply")}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
}
