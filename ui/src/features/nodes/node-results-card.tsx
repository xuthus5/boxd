import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { api } from "@/lib/api/endpoints"
import type { TestResult } from "@/lib/api/types"

function flatten(results: Record<string, Record<string, TestResult>> = {}) {
  return Object.values(results).flatMap((byType) => Object.values(byType))
}

export function NodeResultsCard({ visibleTags }: { visibleTags: ReadonlySet<string> }) {
  const { t } = useTranslation()
  const query = useQuery({ queryKey: ["nodes", "results"], queryFn: api.nodes.results })
  const results = flatten(query.data).filter((result) => visibleTags.has(result.tag))
  if (!results.length) return null
  return <Card><CardHeader><CardTitle>{t("nodes.results")}</CardTitle><CardDescription>{t("nodes.resultsDescription")}</CardDescription></CardHeader><CardContent>
    <Table><TableHeader><TableRow><TableHead>Tag</TableHead><TableHead>{t("nodes.testType")}</TableHead><TableHead>{t("nodes.latency")}</TableHead><TableHead>{t("common.status")}</TableHead></TableRow></TableHeader>
      <TableBody>{results.map((result) => <TableRow key={`${result.tag}-${result.test_type}`}><TableCell>{result.tag}</TableCell><TableCell>{result.test_type.toUpperCase()}</TableCell><TableCell>{result.latency_ms === undefined ? "—" : `${result.latency_ms.toFixed(0)} ms`}</TableCell><TableCell><Badge variant={result.success ? "secondary" : "destructive"}>{result.success ? t("common.normal") : result.error}</Badge></TableCell></TableRow>)}</TableBody>
    </Table>
  </CardContent></Card>
}
