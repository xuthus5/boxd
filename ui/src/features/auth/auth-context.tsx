import { createContext, useContext, useEffect, useMemo, useState } from "react"

import { setUnauthorizedHandler } from "@/lib/api/client"
import { api, type LoginInput } from "@/lib/api/endpoints"
import { sessionStore, type Session } from "@/lib/session"

interface AuthContextValue {
  session: Session | null
  login: (input: LoginInput) => Promise<void>
  logout: () => Promise<void>
  clear: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [session, setSession] = useState(() => sessionStore.get())

  const clear = () => {
    sessionStore.clear()
    setSession(null)
  }

  useEffect(() => {
    setUnauthorizedHandler(clear)
    return () => setUnauthorizedHandler(undefined)
  }, [])

  const value = useMemo<AuthContextValue>(() => ({
    session,
    clear,
    async login(input) {
      const response = await api.auth.login(input)
      const next = { token: response.token, expiresAt: response.expires_at }
      sessionStore.set(next)
      setSession(next)
    },
    async logout() {
      try {
        await api.auth.logout()
      } finally {
        clear()
      }
    },
  }), [session])

  return <AuthContext value={value}>{children}</AuthContext>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) throw new Error("useAuth must be used inside AuthProvider")
  return context
}
