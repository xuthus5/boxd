import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  formatLoginErrorTitle,
  loginErrorClipboardText,
  loginErrorHintKey,
  type LoginErrorState,
} from "@/features/auth/login-error"
import { copyText } from "@/lib/clipboard"

interface LoginErrorAlertProps {
  error: LoginErrorState | null
}

export function LoginErrorAlert({ error }: LoginErrorAlertProps) {
  const { t } = useTranslation()
  if (!error) return null
  const code = error.code !== "unknown" ? error.code : undefined
  const payload = loginErrorClipboardText(error)

  return (
    <Alert variant="destructive" data-testid="login-error" data-error-code={error.code || "unknown"}>
      <div className="flex items-start justify-between gap-2">
        <AlertTitle>{formatLoginErrorTitle(t, error)}</AlertTitle>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="h-6 shrink-0 px-1.5 text-destructive"
          aria-label={t("auth.copyLoginError")}
          onClick={() => {
            if (!payload) return
            void copyText(payload).then(
              () => toast.success(t("auth.loginErrorCopied")),
              () => toast.error(t("auth.loginErrorCopyFailed")),
            )
          }}
        >
          <CopyIcon className="size-3.5" aria-hidden="true" />
        </Button>
      </div>
      <AlertDescription className="mt-1 flex flex-col gap-2">
        {code ? <Badge variant="outline" className="w-fit font-mono text-[10px]">{code}</Badge> : null}
        <span>{error.message}</span>
        <span className="text-[11px] text-muted-foreground">{t(loginErrorHintKey(error.code))}</span>
      </AlertDescription>
    </Alert>
  )
}
