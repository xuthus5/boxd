import { CopyIcon, MinusIcon, SquareIcon, XIcon } from "lucide-react"
import { useEffect, useState, type MouseEvent, type ReactNode } from "react"
import { useTranslation } from "react-i18next"

import {
  closeDesktopWindow,
  minimiseDesktopWindow,
  readDesktopWindowMaximised,
  subscribeDesktopWindowMaximised,
  toggleDesktopWindowMaximise,
} from "@/lib/desktop-window"
import { cn } from "@/lib/utils"

interface WindowControlButtonProps {
  action: "minimise" | "maximise" | "close"
  label: string
  className?: string
  onClick: () => void
  children: ReactNode
}

function WindowControlButton({ action, label, className, onClick, children }: WindowControlButtonProps) {
  return (
    <button
      type="button"
      data-window-action={action}
      aria-label={label}
      title={label}
      onClick={onClick}
      className={cn(
        "window-no-drag flex size-8 items-center justify-center rounded-lg text-muted-foreground outline-hidden transition-[color,background-color,transform] duration-150 hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:scale-95 [&_svg]:size-3.5",
        className,
      )}
    >
      {children}
    </button>
  )
}

/** WindowControls renders the custom title bar controls used by the frameless desktop window. */
export function WindowControls() {
  const { t } = useTranslation()
  const [maximised, setMaximised] = useState(false)

  useEffect(() => {
    let disposed = false
    let unsubscribe: () => void = () => undefined
    const setup = async () => {
      unsubscribe = await subscribeDesktopWindowMaximised((value) => {
        if (!disposed) setMaximised(value)
      })
      if (disposed) {
        unsubscribe()
        return
      }
      const value = await readDesktopWindowMaximised()
      if (!disposed && value !== null) setMaximised(value)
    }
    void setup()
    return () => {
      disposed = true
      unsubscribe()
    }
  }, [])

  const maximiseLabel = maximised ? t("nav.restoreWindow") : t("nav.maximiseWindow")

  return (
    <div className="window-no-drag flex items-center gap-0.5" data-slot="window-controls">
      <WindowControlButton action="minimise" label={t("nav.minimiseWindow")} onClick={() => { void minimiseDesktopWindow() }}>
        <MinusIcon aria-hidden="true" />
      </WindowControlButton>
      <WindowControlButton action="maximise" label={maximiseLabel} onClick={() => { void toggleDesktopWindowMaximise() }}>
        {maximised ? <CopyIcon aria-hidden="true" /> : <SquareIcon aria-hidden="true" />}
      </WindowControlButton>
      <WindowControlButton
        action="close"
        label={t("nav.closeWindow")}
        className="hover:bg-destructive/10 hover:text-destructive"
        onClick={() => { void closeDesktopWindow() }}
      >
        <XIcon aria-hidden="true" />
      </WindowControlButton>
    </div>
  )
}

/** handleDesktopTitleBarDoubleClick maximises the window when the empty title bar is double-clicked. */
export function handleDesktopTitleBarDoubleClick(event: MouseEvent<HTMLElement>): void {
  const target = event.target
  if (target instanceof HTMLElement && target.closest(".window-no-drag")) {
    return
  }
  void toggleDesktopWindowMaximise()
}
