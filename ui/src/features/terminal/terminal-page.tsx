import { useQuery } from "@tanstack/react-query"
import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import { buildTerminalCommands, resolveProxyEndpoint } from "@/features/terminal/terminal-commands"
import { copyText } from "@/lib/clipboard"
import { setupAPI } from "@/lib/api/setup"

function CommandBlock({ label, command }: { label: string; command: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <p className="text-sm font-medium">{label}</p>
      <div className="flex items-center gap-2 rounded-md border p-2">
        <code className="min-w-0 flex-1 break-all text-xs">{command}</code>
        <Button
          variant="outline"
          size="icon-sm"
          aria-label={`${t("terminal.copy")}: ${label}`}
          onClick={() => {
            void copyText(command).then(
              () => toast.success(t("terminal.copied")),
              () => toast.error(t("terminal.copyFailed")),
            )
          }}
        >
          <CopyIcon data-icon="inline-start" />
        </Button>
      </div>
    </div>
  )
}

export function TerminalPage() {
  const { t } = useTranslation()
  const query = useQuery({ queryKey: ["config", "setup"], queryFn: setupAPI.status, retry: false })
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return <PageLoadErrorAlert error={query.error} scope="terminal" onRetry={() => { void query.refetch() }} />
  }
  const endpoint = resolveProxyEndpoint(query.data)
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle role="heading" aria-level={1} className="truncate">{t("pages.terminal")}</CardTitle>
        <CardDescription>{t("terminal.description")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        {endpoint ? (
          <>
            <p className="text-xs text-muted-foreground">{t("terminal.detected", { address: endpoint.url })}</p>
            <div className="grid items-start gap-2 sm:gap-3 md:grid-cols-2">
              {buildTerminalCommands(endpoint).map((item) => (
                <CommandBlock key={item.id} label={t(item.labelKey)} command={item.command} />
              ))}
            </div>
          </>
        ) : (
          <Alert>
            <AlertTitle>{t("terminal.noListenerTitle")}</AlertTitle>
            <AlertDescription className="flex flex-col items-start gap-2">
              <span>{t("terminal.noListenerDescription")}</span>
              <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/proxy/inbounds">
                {t("terminal.noListenerAction")}
              </Link>
            </AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  )
}
