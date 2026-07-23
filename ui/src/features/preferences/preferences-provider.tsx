import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react"

import { useAuth } from "@/features/auth/auth-context"
import { isLogThreshold } from "@/features/observability/log-level"
import { i18n } from "@/i18n"
import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"
import { api } from "@/lib/api/endpoints"
import type { UIPreferences } from "@/lib/api/types"
import { preferencesStore, type Language, type LogThreshold, type Preferences, type Theme } from "@/lib/storage"

interface PreferencesContextValue {
  theme: Theme
  language: Language
  minimumLogLevel: LogThreshold
  setTheme: (theme: Theme) => void
  setLanguage: (language: Language) => void
  setMinimumLogLevel: (level: LogThreshold) => void
}

const PreferencesContext = createContext<PreferencesContextValue | null>(null)
const defaults: Preferences = { theme: "system", language: "zh", minimumLogLevel: "all" }

function applyTheme(theme: Theme) {
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches
  document.documentElement.classList.toggle("dark", theme === "dark" || (theme === "system" && prefersDark))
}

function samePreferences(left: Preferences, right: Preferences) {
  return left.theme === right.theme
    && left.language === right.language
    && left.minimumLogLevel === right.minimumLogLevel
}

function normalizePreferences(value: unknown): Preferences {
  if (!value || typeof value !== "object") return defaults
  const item = value as Partial<Preferences>
  const theme = ["light", "dark", "system"].includes(item.theme ?? "") ? item.theme as Theme : defaults.theme
  const language = ["zh", "en"].includes(item.language ?? "") ? item.language as Language : defaults.language
  const minimumLogLevel = isLogThreshold(item.minimumLogLevel) ? item.minimumLogLevel : defaults.minimumLogLevel
  return { theme, language, minimumLogLevel }
}

function persistRemote(prefs: UIPreferences) {
  return api.settings.setPreferences(prefs).catch((error: unknown) => {
    console.error("Failed to save preferences", error)
    reportSettingsRequestError(error, (key) => i18n.t(key), {
      scope: "preferences-save",
      fallback: i18n.t("settings.preferencesSaveFailed"),
    })
  })
}

export function PreferencesProvider({ children }: { children: React.ReactNode }) {
  const auth = useAuth()
  const [preferences, setPreferences] = useState(() => preferencesStore.get())

  useEffect(() => {
    applyTheme(preferences.theme)
    preferencesStore.set(preferences)
    i18n.changeLanguage(preferences.language).catch((error: unknown) => {
      console.error("Failed to change language", error)
    })
  }, [preferences])

  useEffect(() => {
    if (!auth.session) return
    let cancelled = false
    void (async () => {
      try {
        const remote = normalizePreferences(await api.settings.preferences())
        if (cancelled) return
        const local = preferencesStore.get()
        const remoteIsDefault = samePreferences(remote, defaults)
        const localIsCustom = !samePreferences(local, defaults)
        // One-time migration: push browser-local custom prefs when DB still has defaults.
        if (remoteIsDefault && localIsCustom) {
          await persistRemote(local)
          if (!cancelled) setPreferences(local)
          return
        }
        setPreferences(remote)
      } catch (error: unknown) {
        console.error("Failed to load preferences", error)
        reportSettingsRequestError(error, (key) => i18n.t(key), {
          scope: "preferences-load",
          fallback: i18n.t("settings.preferencesLoadFailed"),
        })
      }
    })()
    return () => {
      cancelled = true
    }
  }, [auth.session])

  const update = useCallback((patch: Partial<Preferences>) => {
    setPreferences((current) => {
      const next = { ...current, ...patch }
      preferencesStore.set(next)
      if (auth.session) void persistRemote(next)
      return next
    })
  }, [auth.session])

  const value = useMemo<PreferencesContextValue>(() => ({
    ...preferences,
    setTheme: (theme) => update({ theme }),
    setLanguage: (language) => update({ language }),
    setMinimumLogLevel: (minimumLogLevel) => update({ minimumLogLevel }),
  }), [preferences, update])

  return <PreferencesContext value={value}>{children}</PreferencesContext>
}

export function usePreferences() {
  const context = useContext(PreferencesContext)
  if (!context) throw new Error("usePreferences must be used inside PreferencesProvider")
  return context
}
