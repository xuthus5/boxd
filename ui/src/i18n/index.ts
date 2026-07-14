import i18n from "i18next"
import { initReactI18next } from "react-i18next"

import { en } from "@/i18n/locales/en"
import { zh } from "@/i18n/locales/zh"
import { preferencesStore } from "@/lib/storage"

const initialization = i18n.use(initReactI18next).init({
  resources: { en, zh },
  lng: preferencesStore.get().language,
  fallbackLng: "zh",
  interpolation: { escapeValue: false },
})

initialization.catch((error: unknown) => {
  console.error("Failed to initialize translations", error)
})

export { i18n }
