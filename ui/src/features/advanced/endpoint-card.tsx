import { PencilIcon, Trash2Icon } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { summarizeEndpoint } from "@/features/advanced/endpoints-form-model"
import type { JsonObject } from "@/features/policy/policy-form-model"

interface EndpointCardProps {
  item: JsonObject
  onEdit: () => void
  onDelete: () => void
}

export function EndpointCard({ item, onEdit, onDelete }: EndpointCardProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const tag = typeof item.tag === "string" && item.tag ? item.tag : t("advanced.endpoints.unnamed")
  const summary = summarizeEndpoint(item)
  return (
    <article aria-labelledby={titleId}>
      <Card size="sm" className="h-full">
        <CardHeader className="min-w-0">
          <CardTitle><h2 id={titleId} className="truncate">{tag}</h2></CardTitle>
          <CardDescription className="truncate">{summary.detail ?? summary.type}</CardDescription>
          <CardAction>
            <Button variant="outline" size="xs" onClick={onEdit}>
              <PencilIcon data-icon="inline-start" />{t("common.edit")}
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            <Badge>{summary.type}</Badge>
            {summary.type === "wireguard" && summary.meta > 0
              ? <Badge variant="secondary">{t("advanced.endpoints.peerCount", { count: summary.meta })}</Badge>
              : null}
            {summary.detail && summary.detail !== summary.type
              ? <Badge variant="outline">{summary.detail}</Badge>
              : null}
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <ConfirmAction
            trigger={<Button variant="destructive" size="xs"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>}
            title={t("advanced.endpoints.deleteTitle")}
            description={t("advanced.endpoints.deleteDescription", { tag })}
            confirmLabel={t("advanced.endpoints.confirmDelete")}
            confirmVariant="destructive"
            onConfirm={onDelete}
          />
        </CardFooter>
      </Card>
    </article>
  )
}
