import { useMutation } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/endpoints"
import type { ImportResult } from "@/lib/api/types"

interface Props { onClose: () => void; onSaved: () => void }

export function NodeImportDialog({ onClose, onSaved }: Props) {
  const { t } = useTranslation()
  const [link, setLink] = useState("")
  const [preview, setPreview] = useState<ImportResult | null>(null)
  const parse = useMutation({ mutationFn: () => api.import.link(link), onSuccess: setPreview, onError: (error: Error) => toast.error(error.message) })
  const save = useMutation({
    mutationFn: () => api.import.save({ tag: preview!.tag, type: preview!.type, server: preview!.server, port: preview!.port, config: preview!.config }).then(() => api.nodes.sync()),
    onSuccess: () => { toast.success(t("nodes.saved")); onSaved() },
    onError: (error: Error) => toast.error(error.message),
  })
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-h-[calc(100dvh-1rem)] min-w-0 gap-3 overflow-y-auto p-3 sm:max-h-[calc(100dvh-2rem)] sm:max-w-lg sm:gap-4 sm:p-4">
        <DialogHeader>
          <DialogTitle className="truncate">{t("nodes.import")}</DialogTitle>
          <DialogDescription>{t("nodes.importDescription")}</DialogDescription>
        </DialogHeader>
        <FieldGroup className="gap-2 sm:gap-3">
          <Field>
            <FieldLabel htmlFor="node-link">{t("nodes.link")}</FieldLabel>
            <Input id="node-link" className="h-8" value={link} onChange={(event) => setLink(event.target.value)} />
          </Field>
          {preview ? (
            <Field>
              <FieldLabel>{t("nodes.parseResult")}</FieldLabel>
              <Card size="sm">
                <CardHeader className="gap-1.5">
                  <CardTitle className="truncate" title={preview.tag}>{preview.tag}</CardTitle>
                  <CardDescription className="truncate">{preview.type}</CardDescription>
                </CardHeader>
                <CardContent className="truncate text-sm tabular-nums" title={`${preview.server}:${preview.port}`}>
                  {preview.server}:{preview.port}
                </CardContent>
              </Card>
            </Field>
          ) : null}
        </FieldGroup>
        <DialogFooter className="gap-2">
          <Button size="sm" className="h-8" variant="outline" disabled={!link || parse.isPending} onClick={() => parse.mutate()}>
            {t("nodes.parse")}
          </Button>
          <Button size="sm" className="h-8" disabled={!preview || save.isPending} onClick={() => save.mutate()}>
            {t("nodes.saveNode")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
