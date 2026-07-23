import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { nodeTestErrorClipboardText, nodeTestErrorLabel } from "@/features/nodes/node-test-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import { api } from "@/lib/api/endpoints"
import type { TestResult } from "@/lib/api/types"

function flatten(results: Record<string, Record<string, TestResult>> = {}) {
  return Object.values(results).flatMap((byType) => Object.values(byType))
}

function copyFailedResult(result: TestResult, t: (key: string) => string) {
  const payload = nodeTestErrorClipboardText(result)
  if (!payload) return
  void copyText(payload).then(
    () => toast.success(t("nodes.testErrorCopied")),
    () => toast.error(t("nodes.testErrorCopyFailed")),
  )
}

function ResultStatusBadge({ result }: { result: TestResult }) {
  const { t } = useTranslation()
  if (result.success) {
    return <Badge variant="secondary">{t("common.normal")}</Badge>
  }
  const label = nodeTestErrorLabel(result, t("nodes.testFailed"))
  return (
    <Badge
      variant="destructive"
      className="max-w-[12rem] cursor-pointer truncate"
      title={label}
      role="button"
      tabIndex={0}
      aria-label={`${t("nodes.copyTestError")}: ${result.tag} ${result.test_type.toUpperCase()} ${label}`}
      onClick={() => copyFailedResult(result, t)}
      onKeyDown={(event) => {
        if (event.key !== "Enter" && event.key !== " ") return
        event.preventDefault()
        copyFailedResult(result, t)
      }}
    >
      {label}
    </Badge>
  )
}

export function NodeResultsCard({ visibleTags }: { visibleTags: ReadonlySet<string> }) {
  const { t } = useTranslation()
  const query = useQuery({ queryKey: ["nodes", "results"], queryFn: api.nodes.results })
  const results = flatten(query.data).filter((result) => visibleTags.has(result.tag))
  if (!results.length) return null
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("nodes.results")}</CardTitle>
        <CardDescription>{t("nodes.resultsDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Tag</TableHead>
              <TableHead>{t("nodes.testType")}</TableHead>
              <TableHead>{t("nodes.latency")}</TableHead>
              <TableHead>{t("common.status")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {results.map((result) => (
              <TableRow key={`${result.tag}-${result.test_type}`}>
                <TableCell className="max-w-[10rem] truncate" title={result.tag}>{result.tag}</TableCell>
                <TableCell>{result.test_type.toUpperCase()}</TableCell>
                <TableCell>
                  {result.latency_ms === undefined ? "—" : `${result.latency_ms.toFixed(0)} ms`}
                </TableCell>
                <TableCell>
                  <ResultStatusBadge result={result} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}
