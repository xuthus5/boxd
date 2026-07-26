import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { DialogFooter } from "@/components/ui/dialog"
import { Spinner } from "@/components/ui/spinner"

interface ProxyEditorFooterProps {
  canSave: boolean
  canValidate: boolean
  validating: boolean
  onClose: () => void
  onSave: () => void
  onValidate: () => void
  saving?: boolean
}

/** Shared cancel / dry-run validate / save actions for proxy item dialogs. */
export function ProxyEditorFooter({
  canSave,
  canValidate,
  validating,
  onClose,
  onSave,
  onValidate,
  saving = false,
}: ProxyEditorFooterProps) {
  const { t } = useTranslation()
  return (
    <DialogFooter className="gap-2">
      <Button variant="outline" size="sm" className="h-8" disabled={saving} onClick={onClose}>
        {t("common.cancel")}
      </Button>
      <Button
        variant="outline"
        size="sm"
        className="h-8"
        disabled={!canValidate || validating || saving}
        onClick={() => { void onValidate() }}
      >
        {validating ? t("advanced.validating") : t("advanced.validate")}
      </Button>
      <Button
        size="sm"
        className="h-8"
        disabled={!canSave || validating || saving}
        onClick={onSave}
      >
        {saving ? <Spinner aria-hidden="true" data-icon="inline-start" /> : null}
        {t("common.save")}
      </Button>
    </DialogFooter>
  )
}
