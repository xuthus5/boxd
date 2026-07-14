import { Badge } from "@/components/ui/badge"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import type { LogEvent } from "@/lib/api/types"

export function RecentLogs({ items }: { items: LogEvent[] }) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader><CardTitle>{t("dashboard.recentLogs")}</CardTitle><CardDescription>{t("dashboard.recentLogsDescription")}</CardDescription></CardHeader>
      <CardContent>
        {items.length === 0 ? <Empty><EmptyHeader><EmptyTitle>{t("dashboard.noLogs")}</EmptyTitle><EmptyDescription>{t("dashboard.logsWaiting")}</EmptyDescription></EmptyHeader></Empty> : <Table><TableHeader><TableRow><TableHead>{t("dashboard.level")}</TableHead><TableHead>{t("dashboard.message")}</TableHead></TableRow></TableHeader><TableBody>{items.map((item, index) => <TableRow key={`${item.level}-${index}`}><TableCell><Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge></TableCell><TableCell>{item.message}</TableCell></TableRow>)}</TableBody></Table>}
      </CardContent>
    </Card>
  )
}
