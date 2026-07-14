import { Navigate, Route, Routes } from "react-router-dom"
import { lazy, Suspense } from "react"

import { AppShell } from "@/app/app-shell"
import { ProtectedRoute } from "@/app/protected-route"
import { LoginPage } from "@/features/auth/login-page"
import { Skeleton } from "@/components/ui/skeleton"

const DashboardPage = lazy(() => import("@/features/dashboard/dashboard-page").then((module) => ({ default: module.DashboardPage })))
const InboundsPage = lazy(() => import("@/features/proxy/inbounds-page").then((module) => ({ default: module.InboundsPage })))
const OutboundsPage = lazy(() => import("@/features/proxy/outbounds-page").then((module) => ({ default: module.OutboundsPage })))
const RoutePage = lazy(() => import("@/features/policy/route-page").then((module) => ({ default: module.RoutePage })))
const DNSPage = lazy(() => import("@/features/policy/dns-page").then((module) => ({ default: module.DNSPage })))
const NodesPage = lazy(() => import("@/features/nodes/nodes-page").then((module) => ({ default: module.NodesPage })))
const SubscriptionsPage = lazy(() => import("@/features/subscriptions/subscriptions-page").then((module) => ({ default: module.SubscriptionsPage })))
const ConnectionsPage = lazy(() => import("@/features/observability/connections-page").then((module) => ({ default: module.ConnectionsPage })))
const LogsPage = lazy(() => import("@/features/observability/logs-page").then((module) => ({ default: module.LogsPage })))
const EndpointsPage = lazy(() => import("@/features/advanced/endpoints-page").then((module) => ({ default: module.EndpointsPage })))
const ExperimentalPage = lazy(() => import("@/features/advanced/experimental-page").then((module) => ({ default: module.ExperimentalPage })))
const RawConfigPage = lazy(() => import("@/features/advanced/raw-config-page").then((module) => ({ default: module.RawConfigPage })))
const SettingsPage = lazy(() => import("@/features/settings/settings-page").then((module) => ({ default: module.SettingsPage })))

export function AppRoutes() {
  return (
    <Suspense fallback={<Skeleton className="h-64 w-full" />}><Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<AppShell />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/proxy/inbounds" element={<InboundsPage />} />
          <Route path="/proxy/outbounds" element={<OutboundsPage />} />
          <Route path="/policy/route" element={<RoutePage />} />
          <Route path="/policy/dns" element={<DNSPage />} />
          <Route path="/nodes" element={<NodesPage />} />
          <Route path="/subscriptions" element={<SubscriptionsPage />} />
          <Route path="/observability/connections" element={<ConnectionsPage />} />
          <Route path="/observability/logs" element={<LogsPage />} />
          <Route path="/advanced/endpoints" element={<EndpointsPage />} />
          <Route path="/advanced/experimental" element={<ExperimentalPage />} />
          <Route path="/advanced/raw" element={<RawConfigPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes></Suspense>
  )
}
