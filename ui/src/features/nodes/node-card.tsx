import { GaugeIcon } from "lucide-react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { nodeTestInput, nodeTestTypes, type NodeTestType } from "@/features/nodes/node-test-inputs"
import { api } from "@/lib/api/endpoints"
import type { Outbound, TestResult } from "@/lib/api/types"

type TestChoice = NodeTestType | "all"

function ResultBadge({ result }: { result?: TestResult }) {
  const { t } = useTranslation()
  if (!result) return <Badge variant="outline">—</Badge>
  if (!result.success) return <Badge variant="destructive">{result.error || t("nodes.testFailed")}</Badge>
  return <Badge variant="secondary">{result.latency_ms === undefined ? t("common.normal") : `${result.latency_ms.toFixed(0)} ms`}</Badge>
}

function TestResults({ results }: { results?: Record<string, TestResult> }) {
  return <dl className="grid gap-2">{nodeTestTypes.map((type) => (
    <div key={type} className="flex items-center justify-between gap-3">
      <dt className="text-sm text-muted-foreground">{type.toUpperCase()}</dt>
      <dd><ResultBadge result={results?.[type]} /></dd>
    </div>
  ))}</dl>
}

function TestControls({ node }: { node: Outbound }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const available = Boolean(node.server && node.port)
  const mutation = useMutation({
    mutationFn: async (choice: TestChoice) => {
      if (choice === "all") return api.nodes.testBatch(nodeTestTypes.map((type) => nodeTestInput(node, type)!), 3)
      return api.nodes.test(nodeTestInput(node, choice)!)
    },
    onSuccess: () => client.invalidateQueries({ queryKey: ["nodes", "results"] }),
    onError: (error: Error) => toast.error(error.message),
  })
  return <DropdownMenu><DropdownMenuTrigger render={<Button variant="outline" size="xs" disabled={!available || mutation.isPending} />}>
    <GaugeIcon data-icon="inline-start" />{t("nodes.test")}
  </DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuGroup>
    <DropdownMenuItem onClick={() => mutation.mutate("all")}>{t("nodes.testAll")}</DropdownMenuItem>
    {nodeTestTypes.map((type) => <DropdownMenuItem key={type} onClick={() => mutation.mutate(type)}>{type.toUpperCase()}</DropdownMenuItem>)}
  </DropdownMenuGroup></DropdownMenuContent></DropdownMenu>
}

export function NodeCard({ node, results }: { node: Outbound; results?: Record<string, TestResult> }) {
  const { t } = useTranslation()
  const titleId = useId()
  const subscription = node.source === "subscription"
  const source = subscription ? node.source_name || t("nodes.subscription") : t("nodes.imported")
  return <article aria-labelledby={titleId}><Card size="sm" className="h-full">
    <CardHeader><CardTitle><h3 id={titleId}>{node.tag}</h3></CardTitle><CardDescription>{node.server ?? "—"}:{node.port ?? "—"}</CardDescription><CardAction><div className="flex items-center gap-2"><Badge variant="outline">{node.type}</Badge><TestControls node={node} /></div></CardAction></CardHeader>
    <CardContent className="flex flex-col gap-3"><Badge variant="secondary">{source}</Badge><TestResults results={results} /></CardContent>
  </Card></article>
}
