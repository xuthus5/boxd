import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  classifyStreamErrorMessage,
  streamErrorClipboardText,
  streamErrorHintKey,
} from "@/features/observability/stream-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import { cn } from "@/lib/utils"

function StreamErrorCopyButton({ payload }: { payload: string }) {
  const { t } = useTranslation()
  return (
    <Button
      type="button"
      variant="ghost"
      size="xs"
      className="h-6 shrink-0 px-1.5 text-destructive"
      aria-label={t("observability.copyStreamError")}
      onClick={() => {
        void copyText(payload).then(
          () => toast.success(t("observability.streamErrorCopied")),
          () => toast.error(t("observability.streamErrorCopyFailed")),
        )
      }}
    >
      <CopyIcon className="size-3.5" aria-hidden="true" />
    </Button>
  )
}

function StreamErrorActions({
  href,
  actionLabel,
  onReconnect,
}: {
  href?: string
  actionLabel?: string
  onReconnect?: () => void
}) {
  const { t } = useTranslation()
  if (!href && !onReconnect) return null
  return (
    <div className="mt-1.5 flex flex-wrap gap-1.5">
      {onReconnect ? (
        <Button type="button" size="sm" variant="outline" className="h-7" onClick={onReconnect}>
          {t("observability.reconnectStream")}
        </Button>
      ) : null}
      {href ? (
        <Link
          to={href}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-7")}
        >
          {actionLabel || t("observability.viewConnections")}
        </Link>
      ) : null}
    </div>
  )
}

export function HealthStreamErrorBlock({
  error,
  status,
  path,
  href,
  actionLabel,
  onReconnect,
}: {
  error?: string
  status?: string
  path?: string
  href?: string
  actionLabel?: string
  onReconnect?: () => void
}) {
  const { t } = useTranslation()
  if (status !== "reconnecting" && !error) return null
  const code = error ? classifyStreamErrorMessage(error) : undefined
  const payload = error ? streamErrorClipboardText({ path, status, error, code }) : ""
  const reconnect = code === "unauthorized" ? undefined : onReconnect

  return (
    <div
      className="rounded-md border border-destructive/40 bg-destructive/5 px-2.5 py-2"
      data-slot="health-stream-error"
      data-error-code={code || (status === "reconnecting" ? "reconnecting" : "unknown")}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex flex-col gap-1">
          <div className="flex flex-wrap items-center gap-1.5">
            <p className="text-xs font-medium text-destructive">
              {status === "reconnecting"
                ? t("observability.streamReconnecting")
                : t("observability.streamDisconnected")}
            </p>
            {code && code !== "unknown" ? (
              <Badge variant="outline" className="font-mono text-[10px]">{code}</Badge>
            ) : null}
          </div>
          {error ? (
            <p className="break-words text-sm text-destructive" title={error}>{error}</p>
          ) : null}
          {error ? (
            <p className="text-[11px] text-muted-foreground">{t(streamErrorHintKey(code))}</p>
          ) : null}
        </div>
        {payload ? <StreamErrorCopyButton payload={payload} /> : null}
      </div>
      <StreamErrorActions href={href} actionLabel={actionLabel} onReconnect={reconnect} />
    </div>
  )
}
