import type { ReactElement } from "react"
import { render } from "@testing-library/react"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { i18n } from "@/i18n"

export function renderApp(ui: ReactElement, route = "/") {
  return render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter></I18nextProvider>)
}
