import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  classifyStreamErrorMessage,
  formatStreamErrorTitle,
  streamErrorClipboardText,
  streamErrorHintKey,
} from "@/features/observability/stream-error"
import { copyText } from "@/features/proxy/copy-tag-button"

export function StreamErrorAlert({
  error,
  path,
  status,
  paused,
  onReconnect,
}: {
  error: string
  path?: string
  status?: string
  paused?: boolean
  onReconnect?: () => void
}) {
  const { t } = useTranslation()
  const code = classifyStreamErrorMessage(error)
  const payload = streamErrorClipboardText({ path, status, paused, error, code })

  return (
    <Alert variant="destructive" data-stream-error-code={code} data-testid="stream-error-alert">
      <div className="flex items-start justify-between gap-2">
        <AlertTitle>{formatStreamErrorTitle(t, code)}</AlertTitle>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="h-6 shrink-0 px-1.5 text-destructive"
          aria-label={t("observability.copyStreamError")}
          onClick={() => {
            if (!payload) return
            void copyText(payload).then(
              () => toast.success(t("observability.streamErrorCopied")),
              () => toast.error(t("observability.streamErrorCopyFailed")),
            )
          }}
        >
          <CopyIcon className="size-3.5" aria-hidden="true" />
        </Button>
      </div>
      {code !== "unknown" ? (
        <Badge variant="outline" className="mt-1 font-mono text-[10px]">{code}</Badge>
      ) : null}
      <AlertDescription className="mt-1 flex flex-col gap-2">
        <p className="break-words">{error}</p>
        <p className="text-[11px] text-muted-foreground">{t(streamErrorHintKey(code))}</p>
        {onReconnect && !paused && code !== "unauthorized" ? (
          <div>
            <Button type="button" size="xs" variant="outline" onClick={onReconnect}>
              {t("observability.reconnectStream")}
            </Button>
          </div>
        ) : null}
      </AlertDescription>
    </Alert>
  )
}
