import { useId, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useAuth } from "@/features/auth/auth-context"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import type { LogEvent } from "@/lib/api/types"

const levelRanks = { debug: 0, info: 1, warn: 2, error: 3 } as const
type LogThreshold = "all" | keyof typeof levelRanks

function meetsThreshold(level: string, threshold: LogThreshold) {
  if (threshold === "all") return true
  const rank = levelRanks[level.toLowerCase() as keyof typeof levelRanks]
  return rank !== undefined && rank >= levelRanks[threshold]
}

function formatLogTimestamp(timestamp?: string) {
  if (!timestamp) return "—"
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString()
}

function LogFilters({ filter, minimum, onFilterChange, onMinimumChange }: { filter: string; minimum: LogThreshold; onFilterChange: (value: string) => void; onMinimumChange: (value: LogThreshold) => void }) {
  const { t } = useTranslation()
  const searchId = useId()
  const levelId = useId()
  const levelDescriptionId = useId()
  const levels = [
    { label: t("observability.allLevels"), value: "all" },
    { label: "Debug", value: "debug" },
    { label: "Info", value: "info" },
    { label: "Warn", value: "warn" },
    { label: "Error", value: "error" },
  ]
  return <FieldGroup className="gap-3 @md/field-group:flex-row"><Field><FieldLabel htmlFor={searchId}>{t("observability.searchLogs")}</FieldLabel><Input id={searchId} aria-label={t("observability.searchLogs")} placeholder={t("observability.searchLogs")} value={filter} onChange={(event) => onFilterChange(event.target.value)} /></Field><Field><FieldLabel htmlFor={levelId}>{t("observability.minimumLogLevel")}</FieldLabel><Select items={levels} value={minimum} onValueChange={(value) => onMinimumChange(String(value) as LogThreshold)}><SelectTrigger id={levelId} aria-label={t("observability.minimumLogLevel")} aria-describedby={levelDescriptionId} className="w-full"><SelectValue /></SelectTrigger><SelectContent><SelectGroup>{levels.map((level) => <SelectItem key={level.value} value={level.value}>{level.label}</SelectItem>)}</SelectGroup></SelectContent></Select><FieldDescription id={levelDescriptionId}>{t("observability.minimumLogLevelDescription")}</FieldDescription></Field></FieldGroup>
}

export function LogPanel({ path, title }: { path: string; title: string }) {
  const { t } = useTranslation()
  const stream = useStreamBuffer<LogEvent>(path, useAuth().session!.token)
  const [filter, setFilter] = useState("")
  const [minimum, setMinimum] = useState<LogThreshold>("all")
  const items = useMemo(() => stream.items.filter((item) => meetsThreshold(item.level, minimum) && `${item.level} ${item.message}`.toLowerCase().includes(filter.toLowerCase())), [filter, minimum, stream.items])
  return <Card><CardHeader><CardTitle>{title}</CardTitle><CardDescription>{t("observability.logDescription")}</CardDescription></CardHeader><CardContent className="flex flex-col gap-3">{stream.error ? <Alert variant="destructive"><AlertTitle>{t("observability.streamError")}</AlertTitle><AlertDescription>{stream.error}</AlertDescription></Alert> : null}<LogFilters filter={filter} minimum={minimum} onFilterChange={setFilter} onMinimumChange={setMinimum} /><ScrollArea className="h-[32rem]">{items.length === 0 ? <Empty><EmptyHeader><EmptyTitle>{t("observability.noLogs")}</EmptyTitle><EmptyDescription>{t("observability.waitLogs")}</EmptyDescription></EmptyHeader></Empty> : <Table><TableHeader><TableRow><TableHead>{t("observability.time")}</TableHead><TableHead>{t("dashboard.level")}</TableHead><TableHead>{t("dashboard.message")}</TableHead></TableRow></TableHeader><TableBody>{items.map((item, index) => <TableRow key={`${item.timestamp}-${item.level}-${index}`}><TableCell className="whitespace-nowrap text-muted-foreground"><time dateTime={item.timestamp || undefined}>{formatLogTimestamp(item.timestamp)}</time></TableCell><TableCell><Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge></TableCell><TableCell className="min-w-64 whitespace-normal break-words">{item.message}</TableCell></TableRow>)}</TableBody></Table>}</ScrollArea></CardContent><CardFooter className="flex gap-2"><Button variant="outline" onClick={() => stream.setPaused(!stream.paused)}>{stream.paused ? t("observability.resume") : t("observability.pause")}</Button><Button variant="outline" onClick={stream.clear}>{t("observability.clear")}</Button></CardFooter></Card>
}
