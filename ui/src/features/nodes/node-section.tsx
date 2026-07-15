import { GaugeIcon } from "lucide-react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { NodeCard } from "@/features/nodes/node-card"
import { nodeTestInputs } from "@/features/nodes/node-test-inputs"
import { api } from "@/lib/api/endpoints"
import type { Outbound, TestResult } from "@/lib/api/types"

const groupTestConcurrency = 8

interface Props {
  title: string
  description: string
  nodes: Outbound[]
  results?: Record<string, Record<string, TestResult>>
}

export function NodeSection({ title, description, nodes, results }: Props) {
  const { t } = useTranslation()
  const titleId = useId()
  const client = useQueryClient()
  const inputs = nodeTestInputs(nodes)
  const mutation = useMutation({
    mutationFn: () => api.nodes.testBatch(inputs, groupTestConcurrency),
    onSuccess: () => { void client.invalidateQueries({ queryKey: ["nodes", "results"] }); toast.success(t("nodes.batchComplete")) },
    onError: (error: Error) => toast.error(error.message),
  })
  return <section aria-labelledby={titleId} className="flex flex-col gap-3">
    <div className="flex items-start justify-between gap-3"><div className="min-w-0"><h2 id={titleId} className="text-lg font-medium">{title}</h2><p className="text-sm text-muted-foreground">{description}</p></div><Button variant="outline" size="sm" disabled={!inputs.length || mutation.isPending} onClick={() => mutation.mutate()}><GaugeIcon data-icon="inline-start" />{t("nodes.batch")}</Button></div>
    {nodes.length ? <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {nodes.map((node) => <NodeCard key={node.tag} node={node} results={results?.[node.tag]} />)}
    </div> : <Empty><EmptyHeader><EmptyTitle>{t("nodes.empty")}</EmptyTitle><EmptyDescription>{t("nodes.emptyDescription")}</EmptyDescription></EmptyHeader></Empty>}
  </section>
}
