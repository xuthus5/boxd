import { Badge } from "@/components/ui/badge"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import type { LogEvent } from "@/lib/api/types"

function formatLogTime(timestamp?: string) {
  if (!timestamp) return "—"
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleTimeString()
}

export function RecentLogs({ items }: { items: LogEvent[] }) {
  const { t } = useTranslation()
  return (
    <Card className="lg:col-span-3">
      <CardHeader><CardTitle>{t("dashboard.recentLogs")}</CardTitle><CardDescription>{t("dashboard.recentLogsDescription")}</CardDescription></CardHeader>
      <CardContent>
        {items.length === 0 ? <Empty><EmptyHeader><EmptyTitle>{t("dashboard.noLogs")}</EmptyTitle><EmptyDescription>{t("dashboard.logsWaiting")}</EmptyDescription></EmptyHeader></Empty> : <Table className="table-fixed"><TableHeader><TableRow><TableHead className="w-28">{t("observability.time")}</TableHead><TableHead className="w-20">{t("dashboard.level")}</TableHead><TableHead>{t("dashboard.message")}</TableHead></TableRow></TableHeader><TableBody>{items.map((item, index) => <TableRow key={`${item.timestamp}-${item.level}-${index}`}><TableCell className="whitespace-nowrap text-muted-foreground"><time dateTime={item.timestamp || undefined}>{formatLogTime(item.timestamp)}</time></TableCell><TableCell><Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge></TableCell><TableCell className="whitespace-normal break-words">{item.message}</TableCell></TableRow>)}</TableBody></Table>}
      </CardContent>
    </Card>
  )
}
