import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { isLoopbackListener, listenerCommands } from "@/features/setup/setup-access"
import { copyText } from "@/lib/clipboard"
import type { SetupListener, SetupStatus } from "@/lib/api/setup"

function CopyCommand({ command, title }: { command: string; title: string }) {
  const { t } = useTranslation()
  return <div className="flex flex-col gap-1.5">
    <p>{title}</p>
    <div className="flex items-center gap-2 rounded-md border p-2">
      <code className="min-w-0 flex-1 break-all text-xs">{command}</code>
      <Button variant="outline" size="icon-sm" aria-label={`${t("setup.copy")}: ${title}`} onClick={() => {
        void copyText(command).then(() => toast.success(t("setup.copied")), () => toast.error(t("setup.copyFailed")))
      }}><CopyIcon data-icon="inline-start" /></Button>
    </div>
  </div>
}

function ListenerGuide({ listener, container }: { listener: SetupListener; container: boolean }) {
  const { t } = useTranslation()
  const commands = listenerCommands(listener, container)
  return <li className="flex flex-col gap-2">
    <div className="flex flex-wrap items-center gap-1.5">
      <Badge variant="outline">{listener.tag}</Badge><Badge variant="secondary">{listener.type}</Badge>
      <span>{t("setup.listener")}: {listener.listen}{listener.listen_port ? `:${listener.listen_port}` : ""}</span>
    </div>
    {listener.type === "tun" ? <p>{t("setup.tunScope")}</p> : null}
    {container && isLoopbackListener(listener) ? <p>{t("setup.loopbackContainer")}</p> : null}
    {commands?.publication ? <>
      <CopyCommand command={commands.publication} title={t("setup.portPublication")} />
      <p>{t("setup.portPublicationHint")}</p>
    </> : null}
    {commands ? <CopyCommand command={commands.check} title={t("setup.clientCheck")} /> : null}
  </li>
}

export function SetupAccessGuide({ status }: { status: SetupStatus }) {
  const { t } = useTranslation()
  return <Alert>
    <AlertTitle>{t("setup.accessTitle")}</AlertTitle>
    <AlertDescription className="flex flex-col gap-3">
      <p>{t("setup.accessDescription")}</p>
      {status.listeners.length ? <ul className="flex flex-col gap-4">{status.listeners.map((listener, index) =>
        <ListenerGuide key={`${index}-${listener.tag}`} listener={listener} container={status.capabilities.container} />)}</ul>
        : <p>{t("setup.noListener")}</p>}
      <p>{t("setup.clientCheckHint")}</p><p>{t("setup.remoteHint")}</p>
      <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/proxy/inbounds">{t("setup.details")}</Link>
    </AlertDescription>
  </Alert>
}
