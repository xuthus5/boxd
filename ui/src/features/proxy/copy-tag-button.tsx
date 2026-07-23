import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"

export async function copyText(value: string) {
  if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }
  throw new Error("clipboard unavailable")
}

export function CopyTagButton({ tag }: { tag: string }) {
  const { t } = useTranslation()
  return (
    <Button
      variant="outline"
      size="xs"
      aria-label={t("proxy.copyTag")}
      onClick={() => {
        void copyText(tag).then(
          () => toast.success(t("proxy.tagCopied")),
          () => toast.error(t("proxy.tagCopyFailed")),
        )
      }}
    >
      <CopyIcon data-icon="inline-start" />
      {t("proxy.copyTag")}
    </Button>
  )
}
