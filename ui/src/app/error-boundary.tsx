import { Component, type ErrorInfo, type ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"

interface ErrorBoundaryProps { children: ReactNode; title: string; description: string; retryLabel: string }
interface ErrorBoundaryState { error: Error | null }

class ErrorBoundaryImpl extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }
  static getDerivedStateFromError(error: Error): ErrorBoundaryState { return { error } }
  componentDidCatch(error: Error, info: ErrorInfo) { console.error("Unhandled UI error", error, info) }
  reset = () => this.setState({ error: null })
  render() {
    if (!this.state.error) return this.props.children
    return <main className="flex min-h-svh items-center justify-center p-6"><Card className="w-full max-w-lg"><CardHeader><CardTitle>{this.props.title}</CardTitle></CardHeader><CardContent><Alert variant="destructive"><AlertTitle>{this.props.description}</AlertTitle><AlertDescription>{this.state.error.message}</AlertDescription></Alert></CardContent><CardFooter><Button onClick={this.reset}>{this.props.retryLabel}</Button></CardFooter></Card></main>
  }
}

export function ErrorBoundary({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  return <ErrorBoundaryImpl title={t("common.unexpectedError")} description={t("common.unexpectedErrorDescription")} retryLabel={t("common.retry")}>{children}</ErrorBoundaryImpl>
}
