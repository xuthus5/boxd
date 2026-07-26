import { ListPlusIcon, ServerIcon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { ServiceCard } from "@/features/advanced/service-card"
import { ServiceEditorDialog } from "@/features/advanced/service-editor-dialog"
import { createServiceDraft } from "@/features/advanced/services-form-model"
import { cloneJsonObject, type JsonObject } from "@/features/policy/policy-form-model"

export interface ServicesVisualEditorProps {
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

export function ServicesVisualEditor({ items, onChange }: ServicesVisualEditorProps) {
  const { t } = useTranslation()
  const [selection, setSelection] = useState<Selection | null>(null)
  const edit = (index: number | null) => setSelection({
    index,
    item: index === null ? createServiceDraft() : cloneJsonObject(items[index]),
  })
  const remove = (index: number) => onChange(items.filter((_, itemIndex) => itemIndex !== index))
  const save = (item: JsonObject) => {
    if (!selection) return
    onChange(replaceOrAppend(items, selection.index, item))
    setSelection(null)
  }

  return (
    <div className="flex min-w-0 flex-col gap-2 sm:gap-3">
      <header className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="truncate text-base font-semibold">{t("advanced.services.listTitle")}</h2>
          <p className="text-sm text-muted-foreground">{t("advanced.services.listDescription")}</p>
        </div>
        <Button size="sm" className="h-8 w-full shrink-0 sm:w-auto" onClick={() => edit(null)}>
          <ListPlusIcon data-icon="inline-start" />{t("advanced.services.add")}
        </Button>
      </header>
      {items.length === 0
        ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon"><ServerIcon /></EmptyMedia>
              <EmptyTitle>{t("advanced.services.empty")}</EmptyTitle>
              <EmptyDescription>{t("advanced.services.emptyDescription")}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button size="sm" className="h-8" onClick={() => edit(null)}>
                <ListPlusIcon data-icon="inline-start" />{t("advanced.services.add")}
              </Button>
            </EmptyContent>
          </Empty>
        )
        : (
          <div className="grid gap-2 sm:grid-cols-2 sm:gap-3 xl:grid-cols-3">
            {items.map((item, index) => (
              <ServiceCard
                key={`${String(item.tag)}-${String(item.type)}-${index}`}
                item={item}
                onEdit={() => edit(index)}
                onDelete={() => remove(index)}
              />
            ))}
          </div>
        )}
      <p className="text-xs text-muted-foreground sm:text-sm">{t("advanced.services.count", { count: items.length })}</p>
      {selection
        ? (
          <ServiceEditorDialog
            key={`${selection.index}:${JSON.stringify(selection.item)}`}
            open
            item={selection.item}
            title={selection.index === null ? t("advanced.services.addTitle") : t("advanced.services.editTitle")}
            onOpenChange={(open) => { if (!open) setSelection(null) }}
            onSave={save}
          />
        )
        : null}
    </div>
  )
}
