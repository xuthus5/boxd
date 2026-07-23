import { GaugeIcon } from "lucide-react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import {
  batchTestToastTone,
  formatBatchTestToastMessage,
  summarizeBatchTestResults,
} from "@/features/nodes/batch-test-summary"
import { NodeCard } from "@/features/nodes/node-card"
import { nodeTestInputs } from "@/features/nodes/node-test-inputs"
import { api } from "@/lib/api/endpoints"
import type { LatencyPoint, Outbound, TestResult } from "@/lib/api/types"

const groupTestConcurrency = 8

interface Props {
  title: string
  description: string
  nodes: Outbound[]
  results?: Record<string, Record<string, TestResult>>
  history?: Record<string, Record<string, LatencyPoint[]>>
  onBatchComplete?: () => void
}

function publishBatchToast(
  results: readonly TestResult[] | undefined,
  t: (key: string, values?: Record<string, string | number>) => string,
) {
  const summary = summarizeBatchTestResults(results)
  const message = formatBatchTestToastMessage(summary, t)
  const tone = batchTestToastTone(summary)
  if (tone === "error") toast.error(message)
  else if (tone === "warning") toast.warning(message)
  else toast.success(message)
}

export function NodeSection({ title, description, nodes, results, history, onBatchComplete }: Props) {
  const { t } = useTranslation()
  const titleId = useId()
  const client = useQueryClient()
  const inputs = nodeTestInputs(nodes)
  const mutation = useMutation({
    mutationFn: () => api.nodes.testBatch(inputs, groupTestConcurrency),
    onSuccess: (payload) => {
      void client.invalidateQueries({ queryKey: ["nodes", "results"] })
      void client.invalidateQueries({ queryKey: ["nodes", "history"] })
      publishBatchToast(payload.results, t)
      onBatchComplete?.()
    },
    onError: (error: Error) => toast.error(error.message),
  })
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-2.5 sm:gap-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h2 id={titleId} className="text-base font-medium sm:text-lg">{title}</h2>
          <p className="text-xs text-muted-foreground sm:text-sm">{description}</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          className="h-8 shrink-0"
          disabled={!inputs.length || mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          <GaugeIcon data-icon="inline-start" />{t("nodes.batch")}
        </Button>
      </div>
      {nodes.length ? (
        <div className="grid gap-2 sm:grid-cols-2 sm:gap-3 xl:grid-cols-3">
          {nodes.map((node) => (
            <NodeCard
              key={node.tag}
              node={node}
              results={results?.[node.tag]}
              history={history?.[node.tag]}
            />
          ))}
        </div>
      ) : (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{t("nodes.empty")}</EmptyTitle>
            <EmptyDescription>{t("nodes.emptyDescription")}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </section>
  )
}
