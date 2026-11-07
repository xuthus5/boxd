import { Navigate, Outlet, useLocation } from "react-router-dom"

import { useAuth } from "@/features/auth/auth-context"
import { isDesktop } from "@/lib/api/desktop"

export function ProtectedRoute() {
  const auth = useAuth()
  const location = useLocation()
  if (isDesktop()) return <Outlet />
  if (!auth.session) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return <Outlet />
}
