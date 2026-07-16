import { AppProviders } from "@/app/providers"
import { ErrorBoundary } from "@/app/error-boundary"
import { AppRoutes } from "@/app/router"

function App() {
  return (
    <AppProviders><ErrorBoundary><AppRoutes /></ErrorBoundary></AppProviders>
  )
}

export default App
