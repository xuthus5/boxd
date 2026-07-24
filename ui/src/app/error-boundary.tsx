import { Component, type ErrorInfo, type ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  error: Error | null
}

function ErrorBoundaryFallback({
  error,
  onRetry,
}: {
  error: Error
  onRetry: () => void
}) {
  const { t } = useTranslation()

  return (
    <main className="flex min-h-svh items-center justify-center p-6">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>{t("common.unexpectedError")}</CardTitle>
        </CardHeader>
        <CardContent>
          <PageLoadErrorAlert
            error={error}
            scope="error-boundary"
            titleKey="common.unexpectedErrorDescription"
          />
        </CardContent>
        <CardFooter>
          <Button type="button" onClick={onRetry}>
            {t("common.retry")}
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}

class ErrorBoundaryImpl extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Unhandled UI error", error, info)
  }

  reset = () => {
    this.setState({ error: null })
  }

  render() {
    if (!this.state.error) {
      return this.props.children
    }
    return <ErrorBoundaryFallback error={this.state.error} onRetry={this.reset} />
  }
}

export function ErrorBoundary({ children }: ErrorBoundaryProps) {
  return <ErrorBoundaryImpl>{children}</ErrorBoundaryImpl>
}
