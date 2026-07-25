import { useQuery } from "@tanstack/react-query"
import { RotateCcwIcon } from "lucide-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { CardQueryError } from "@/features/common/card-query-error"
import { ConfigDiffPanel } from "@/features/config/config-diff-panel"
import { type ConfigDiffItem, diffConfig } from "@/features/config/config-diff"
import { shortConfigHash } from "@/features/dashboard/config-apply-source"
import type { ConfigRestoreHandler } from "@/features/dashboard/use-config-restore"
import { api } from "@/lib/api/endpoints"
import type { ConfigApplyEvent, SingBoxConfig } from "@/lib/api/types"

interface ConfigRestoreActionProps {
  event: ConfigApplyEvent
  currentConfig?: SingBoxConfig
  currentConfigLoading: boolean
  restoring: boolean
  onRestore: ConfigRestoreHandler
}

interface RestorePreviewProps {
  currentConfigAvailable: boolean
  currentConfigLoading: boolean
  snapshotLoading: boolean
  snapshotError: unknown
  items?: readonly ConfigDiffItem[]
  onRetry: () => void
}

function RestorePreview({
  currentConfigAvailable,
  currentConfigLoading,
  snapshotLoading,
  snapshotError,
  items,
  onRetry,
}: RestorePreviewProps) {
  const { t } = useTranslation()
  if (snapshotError) {
    return <CardQueryError error={snapshotError} scope="config-restore-snapshot" fallback={t("configRestore.snapshotLoadFailed")} onRetry={onRetry} />
  }
  if (snapshotLoading) {
    return <div className="flex flex-col gap-2"><Skeleton className="h-32 w-full" /><p className="text-xs text-muted-foreground">{t("configRestore.snapshotLoading")}</p></div>
  }
  if (currentConfigLoading) {
    return <p className="text-sm text-muted-foreground">{t("configRestore.currentConfigLoading")}</p>
  }
  if (!currentConfigAvailable || !items) {
    return <p className="text-sm text-muted-foreground">{t("configRestore.currentConfigUnavailable")}</p>
  }
  return <ConfigDiffPanel items={items} />
}

function RestoreFooter({ canRestore, restoring, onCancel, onConfirm }: {
  canRestore: boolean
  restoring: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  const { t } = useTranslation()
  return (
    <DialogFooter>
      <Button type="button" variant="outline" disabled={restoring} onClick={onCancel}>{t("common.cancel")}</Button>
      <Button type="button" disabled={!canRestore} aria-busy={restoring} onClick={onConfirm}>
        {restoring ? <Spinner data-icon="inline-start" /> : <RotateCcwIcon data-icon="inline-start" />}
        {restoring ? t("configRestore.restoringConfig") : t("configRestore.confirmRestoreConfig")}
      </Button>
    </DialogFooter>
  )
}

function useRestorePreview(eventID: string, open: boolean, currentConfig?: SingBoxConfig) {
  const snapshot = useQuery({
    queryKey: ["config", "apply-history", eventID, "snapshot"],
    queryFn: () => api.config.applyHistorySnapshot(eventID),
    enabled: open && Boolean(eventID),
    staleTime: Number.POSITIVE_INFINITY,
  })
  const items = useMemo(() => (
    currentConfig && snapshot.data ? diffConfig(currentConfig, snapshot.data) : undefined
  ), [currentConfig, snapshot.data])
  return { items, snapshot }
}

export function ConfigRestoreAction({
  event,
  currentConfig,
  currentConfigLoading,
  restoring,
  onRestore,
}: ConfigRestoreActionProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const { items, snapshot } = useRestorePreview(event.id?.trim() ?? "", open, currentConfig)
  const canRestore = Boolean(items?.length) && !currentConfigLoading && !snapshot.isFetching && !restoring
  const confirmRestore = async () => {
    if (!canRestore) return
    if (await onRestore(event)) setOpen(false)
  }
  return (
    <div className="mt-1.5 flex justify-end">
      <Dialog open={open} onOpenChange={(nextOpen) => { if (!restoring) setOpen(nextOpen) }}>
        <DialogTrigger render={(
          <Button type="button" size="sm" variant="outline" className="h-7" disabled={restoring} aria-busy={restoring}>
            <RotateCcwIcon data-icon="inline-start" />
            {restoring ? t("configRestore.restoringConfig") : t("configRestore.restoreConfig")}
          </Button>
        )} />
        <DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("configRestore.restoreConfigTitle")}</DialogTitle>
            <DialogDescription>{t("configRestore.restoreConfigDescription", { hash: shortConfigHash(event.hash) })}</DialogDescription>
          </DialogHeader>
          <div className="min-h-0 overflow-y-auto pr-1">
            <RestorePreview
              currentConfigAvailable={currentConfig !== undefined}
              currentConfigLoading={currentConfigLoading}
              snapshotLoading={snapshot.isPending}
              snapshotError={snapshot.error}
              items={items}
              onRetry={() => { void snapshot.refetch() }}
            />
          </div>
          <RestoreFooter
            canRestore={canRestore}
            restoring={restoring}
            onCancel={() => setOpen(false)}
            onConfirm={() => { void confirmRestore() }}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
