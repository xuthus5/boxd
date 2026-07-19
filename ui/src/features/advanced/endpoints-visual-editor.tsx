import { ListPlusIcon, NetworkIcon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { EndpointCard } from "@/features/advanced/endpoint-card"
import { EndpointEditorDialog } from "@/features/advanced/endpoint-editor-dialog"
import { createEndpointDraft } from "@/features/advanced/endpoints-form-model"
import { cloneJsonObject, type JsonObject } from "@/features/policy/policy-form-model"

export interface EndpointsVisualEditorProps {
  items: JsonObject[]
  onChange: (items: JsonObject[]) => void
}

interface Selection {
  index: number | null
  item: JsonObject
}

function replaceOrAppend(items: readonly JsonObject[], index: number | null, item: JsonObject) {
  if (index === null) return [...items, item]
  return items.map((current, currentIndex) => currentIndex === index ? item : current)
}

export function EndpointsVisualEditor({ items, onChange }: EndpointsVisualEditorProps) {
  const { t } = useTranslation()
  const [selection, setSelection] = useState<Selection | null>(null)
  const edit = (index: number | null) => setSelection({
    index,
    item: index === null ? createEndpointDraft() : cloneJsonObject(items[index]),
  })
  const remove = (index: number) => onChange(items.filter((_, itemIndex) => itemIndex !== index))
  const save = (item: JsonObject) => {
    if (!selection) return
    onChange(replaceOrAppend(items, selection.index, item))
    setSelection(null)
  }

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <Card>
        <CardHeader className="min-w-0 grid-cols-1 has-data-[slot=card-action]:grid-cols-1 sm:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
          <CardTitle>{t("advanced.endpoints.listTitle")}</CardTitle>
          <CardDescription>{t("advanced.endpoints.listDescription")}</CardDescription>
          <CardAction className="col-start-1 row-start-auto w-full justify-self-start sm:col-start-2 sm:row-start-1 sm:w-auto sm:justify-self-end">
            <Button className="w-full sm:w-auto" onClick={() => edit(null)}>
              <ListPlusIcon data-icon="inline-start" />{t("advanced.endpoints.add")}
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          {items.length === 0
            ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon"><NetworkIcon /></EmptyMedia>
                  <EmptyTitle>{t("advanced.endpoints.empty")}</EmptyTitle>
                  <EmptyDescription>{t("advanced.endpoints.emptyDescription")}</EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button onClick={() => edit(null)}>
                    <ListPlusIcon data-icon="inline-start" />{t("advanced.endpoints.add")}
                  </Button>
                </EmptyContent>
              </Empty>
            )
            : (
              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {items.map((item, index) => (
                  <EndpointCard
                    key={`${String(item.tag)}-${index}`}
                    item={item}
                    onEdit={() => edit(index)}
                    onDelete={() => remove(index)}
                  />
                ))}
              </div>
            )}
        </CardContent>
        <CardFooter>
          <p className="text-muted-foreground">{t("advanced.endpoints.count", { count: items.length })}</p>
        </CardFooter>
      </Card>
      {selection
        ? (
          <EndpointEditorDialog
            key={`${selection.index}:${JSON.stringify(selection.item)}`}
            open
            item={selection.item}
            title={selection.index === null ? t("advanced.endpoints.addTitle") : t("advanced.endpoints.editTitle")}
            onOpenChange={(open) => { if (!open) setSelection(null) }}
            onSave={save}
          />
        )
        : null}
    </div>
  )
}
