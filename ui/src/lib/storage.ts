const PREFERENCES_KEY = "boxui.preferences.v1"

export type Theme = "light" | "dark" | "system"
export type Language = "zh" | "en"
export interface Preferences { theme: Theme; language: Language }

const defaults: Preferences = { theme: "system", language: "zh" }

function isPreferences(value: unknown): value is Preferences {
  if (!value || typeof value !== "object") return false
  const item = value as Partial<Preferences>
  return ["light", "dark", "system"].includes(item.theme ?? "")
    && ["zh", "en"].includes(item.language ?? "")
}

export const preferencesStore = {
  get(): Preferences {
    try {
      const raw = localStorage.getItem(PREFERENCES_KEY)
      if (!raw) return defaults
      const value: unknown = JSON.parse(raw)
      return isPreferences(value) ? value : defaults
    } catch {
      return defaults
    }
  },
  set(preferences: Preferences) {
    try {
      localStorage.setItem(PREFERENCES_KEY, JSON.stringify(preferences))
    } catch {
      return
    }
  },
}
