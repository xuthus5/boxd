export async function copyText(text: string, clipboard?: Pick<Clipboard, "writeText">): Promise<void> {
  const api = clipboard ?? (typeof navigator !== "undefined" ? navigator.clipboard : undefined)
  if (!api?.writeText) throw new Error("clipboard unavailable")
  await api.writeText(text)
}
