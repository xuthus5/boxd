type WailsRuntime = typeof import("@wailsio/runtime")

const WINDOW_MAXIMISE_EVENT = "common:WindowMaximise"
const WINDOW_UNMAXIMISE_EVENT = "common:WindowUnMaximise"
const WINDOW_RESTORE_EVENT = "common:WindowRestore"

async function loadWailsRuntime(): Promise<WailsRuntime> {
  return import("@wailsio/runtime")
}

function reportDesktopWindowError(action: string, error: unknown): void {
  console.error(`[desktop] window ${action} failed`, error)
}

/** minimiseDesktopWindow minimises the native Wails window. */
export async function minimiseDesktopWindow(): Promise<void> {
  try {
    const runtime = await loadWailsRuntime()
    await runtime.Window.Minimise()
  } catch (error) {
    reportDesktopWindowError("minimise", error)
  }
}

/** toggleDesktopWindowMaximise switches the native Wails window between maximised and restored. */
export async function toggleDesktopWindowMaximise(): Promise<void> {
  try {
    const runtime = await loadWailsRuntime()
    await runtime.Window.ToggleMaximise()
  } catch (error) {
    reportDesktopWindowError("maximise", error)
  }
}

/** closeDesktopWindow requests the native Wails window to close. */
export async function closeDesktopWindow(): Promise<void> {
  try {
    const runtime = await loadWailsRuntime()
    await runtime.Window.Close()
  } catch (error) {
    reportDesktopWindowError("close", error)
  }
}

/** readDesktopWindowMaximised reads the current maximised state. */
export async function readDesktopWindowMaximised(): Promise<boolean | null> {
  try {
    const runtime = await loadWailsRuntime()
    return await runtime.Window.IsMaximised()
  } catch (error) {
    reportDesktopWindowError("maximise-state", error)
    return null
  }
}

/** subscribeDesktopWindowMaximised listens for native maximise state changes. */
export async function subscribeDesktopWindowMaximised(
  listener: (maximised: boolean) => void,
): Promise<() => void> {
  try {
    const runtime = await loadWailsRuntime()
    const unsubscribe = [
      runtime.Events.On(WINDOW_MAXIMISE_EVENT, () => listener(true)),
      runtime.Events.On(WINDOW_UNMAXIMISE_EVENT, () => listener(false)),
      runtime.Events.On(WINDOW_RESTORE_EVENT, () => listener(false)),
    ]
    return () => {
      for (const off of unsubscribe) {
        off()
      }
    }
  } catch (error) {
    reportDesktopWindowError("maximise-events", error)
    return () => undefined
  }
}
