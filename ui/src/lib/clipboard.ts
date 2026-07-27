export async function copyText(
  text: string,
  clipboard?: Pick<Clipboard, "writeText">,
  documentRef: Document | undefined = typeof document !== "undefined" ? document : undefined,
): Promise<void> {
  const api = clipboard ?? (typeof navigator !== "undefined" ? navigator.clipboard : undefined)
  if (api?.writeText) {
    try {
      await api.writeText(text)
      return
    } catch (error) {
      if (!documentRef?.body || typeof documentRef.execCommand !== "function") throw error
    }
  }
  copyTextWithDocument(text, documentRef)
}

function copyTextWithDocument(text: string, documentRef?: Document): void {
  if (!documentRef?.body || typeof documentRef.execCommand !== "function") {
    throw new Error("clipboard unavailable")
  }
  const textarea = documentRef.createElement("textarea")
  textarea.value = text
  textarea.setAttribute("readonly", "")
  textarea.setAttribute("aria-hidden", "true")
  textarea.tabIndex = -1
  textarea.style.position = "fixed"
  textarea.style.left = "-9999px"
  textarea.style.opacity = "0"
  textarea.style.pointerEvents = "none"
  documentRef.body.appendChild(textarea)
  try {
    textarea.select()
    textarea.setSelectionRange(0, text.length)
    if (!documentRef.execCommand("copy")) throw new Error("clipboard unavailable")
  } finally {
    textarea.remove()
  }
}
