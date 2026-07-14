import { createContext, useContext, useEffect, useMemo, useState } from "react"

import { i18n } from "@/i18n"
import { preferencesStore, type Language, type Theme } from "@/lib/storage"

interface PreferencesContextValue {
  theme: Theme
  language: Language
  setTheme: (theme: Theme) => void
  setLanguage: (language: Language) => void
}

const PreferencesContext = createContext<PreferencesContextValue | null>(null)

function applyTheme(theme: Theme) {
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches
  document.documentElement.classList.toggle("dark", theme === "dark" || (theme === "system" && prefersDark))
}

export function PreferencesProvider({ children }: { children: React.ReactNode }) {
  const [preferences, setPreferences] = useState(() => preferencesStore.get())

  useEffect(() => {
    applyTheme(preferences.theme)
    preferencesStore.set(preferences)
    i18n.changeLanguage(preferences.language).catch((error: unknown) => {
      console.error("Failed to change language", error)
    })
  }, [preferences])

  const value = useMemo<PreferencesContextValue>(() => ({
    ...preferences,
    setTheme: (theme) => setPreferences((current) => ({ ...current, theme })),
    setLanguage: (language) => setPreferences((current) => ({ ...current, language })),
  }), [preferences])

  return <PreferencesContext value={value}>{children}</PreferencesContext>
}

export function usePreferences() {
  const context = useContext(PreferencesContext)
  if (!context) throw new Error("usePreferences must be used inside PreferencesProvider")
  return context
}
