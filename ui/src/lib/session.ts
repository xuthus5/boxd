const SESSION_KEY = "boxui.session.v1"

export interface Session {
  token: string
  expiresAt: string
}

function isSession(value: unknown): value is Session {
  if (!value || typeof value !== "object") return false
  const session = value as Partial<Session>
  return typeof session.token === "string" && typeof session.expiresAt === "string"
}

function removeSession() {
  try {
    sessionStorage.removeItem(SESSION_KEY)
  } catch {
    return
  }
}

export const sessionStore = {
  get(): Session | null {
    try {
      const raw = sessionStorage.getItem(SESSION_KEY)
      if (!raw) return null
      const session: unknown = JSON.parse(raw)
      if (!isSession(session) || Date.parse(session.expiresAt) <= Date.now()) {
        removeSession()
        return null
      }
      return session
    } catch {
      removeSession()
      return null
    }
  },
  set(session: Session) {
    try {
      sessionStorage.setItem(SESSION_KEY, JSON.stringify(session))
    } catch {
      return
    }
  },
  clear: removeSession,
  isValid() {
    return this.get() !== null
  },
}
