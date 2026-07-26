import { PencilIcon, Trash2Icon } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { summarizeService } from "@/features/advanced/services-form-model"
import type { JsonObject } from "@/features/policy/policy-form-model"

interface ServiceCardProps {
  item: JsonObject
  onEdit: () => void
  onDelete: () => void
}

export function ServiceCard({ item, onEdit, onDelete }: ServiceCardProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const tag = typeof item.tag === "string" && item.tag ? item.tag : t("advanced.services.unnamed")
  const summary = summarizeService(item)
  return (
    <article aria-labelledby={titleId}>
      <Card size="sm" className="h-full">
        <CardHeader className="min-w-0 gap-1.5">
          <CardTitle><h2 id={titleId} className="truncate" title={tag}>{tag}</h2></CardTitle>
          <CardDescription className="truncate" title={summary.detail ?? summary.type}>
            {summary.detail ?? summary.type}
          </CardDescription>
          <CardAction>
            <Button variant="outline" size="xs" className="h-7" onClick={onEdit}>
              <PencilIcon data-icon="inline-start" />{t("common.edit")}
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-1.5">
            <Badge className="max-w-full truncate">{summary.type}</Badge>
            {summary.type === "ssm-api" && summary.meta > 0
              ? <Badge variant="secondary">{t("advanced.services.serverCount", { count: summary.meta })}</Badge>
              : null}
            {summary.detail
              ? <Badge variant="outline" className="max-w-full truncate">{summary.detail}</Badge>
              : null}
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <ConfirmAction
            trigger={<Button variant="destructive" size="xs" className="h-7"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>}
            title={t("advanced.services.deleteTitle")}
            description={t("advanced.services.deleteDescription", { tag })}
            confirmLabel={t("advanced.services.confirmDelete")}
            confirmVariant="destructive"
            onConfirm={onDelete}
          />
        </CardFooter>
      </Card>
    </article>
  )
}
