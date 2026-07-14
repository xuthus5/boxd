import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useAuth } from "@/features/auth/auth-context"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import type { LogEvent } from "@/lib/api/types"

export function LogPanel({ path, title }: { path: string; title: string }) {
  const { t } = useTranslation()
  const stream = useStreamBuffer<LogEvent>(path, useAuth().session!.token)
  const [filter, setFilter] = useState("")
  const items = useMemo(() => stream.items.filter((item) => `${item.level} ${item.message}`.toLowerCase().includes(filter.toLowerCase())), [filter, stream.items])
  return <Card><CardHeader><CardTitle>{title}</CardTitle><CardDescription>{t("observability.logDescription")}</CardDescription></CardHeader><CardContent className="flex flex-col gap-3">{stream.error ? <Alert variant="destructive"><AlertTitle>{t("observability.streamError")}</AlertTitle><AlertDescription>{stream.error}</AlertDescription></Alert> : null}<Input aria-label={t("observability.searchLogs")} placeholder={t("observability.searchLogs")} value={filter} onChange={(event) => setFilter(event.target.value)} /><ScrollArea className="h-[32rem]">{items.length === 0 ? <Empty><EmptyHeader><EmptyTitle>{t("observability.noLogs")}</EmptyTitle><EmptyDescription>{t("observability.waitLogs")}</EmptyDescription></EmptyHeader></Empty> : <Table><TableHeader><TableRow><TableHead>{t("dashboard.level")}</TableHead><TableHead>{t("dashboard.message")}</TableHead></TableRow></TableHeader><TableBody>{items.map((item, index) => <TableRow key={`${item.level}-${index}`}><TableCell><Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge></TableCell><TableCell>{item.message}</TableCell></TableRow>)}</TableBody></Table>}</ScrollArea></CardContent><CardFooter className="flex gap-2"><Button variant="outline" onClick={() => stream.setPaused(!stream.paused)}>{stream.paused ? t("observability.resume") : t("observability.pause")}</Button><Button variant="outline" onClick={stream.clear}>{t("observability.clear")}</Button></CardFooter></Card>
}
