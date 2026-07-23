import { json } from "@codemirror/lang-json"
import { EditorView } from "@codemirror/view"
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror"
import { forwardRef, useImperativeHandle, useRef } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { locateJsonPath } from "@/features/config/json-path"
import { isValidJSON } from "@/features/config/json-utils"

interface JsonEditorProps {
  value: string
  onChange: (value: string) => void
  ariaLabel: string
  readOnly?: boolean
}

export interface JsonEditorHandle {
  revealPath: (path: string) => boolean
  focus: () => void
}

export const JsonEditor = forwardRef<JsonEditorHandle, JsonEditorProps>(function JsonEditor(
  { value, onChange, ariaLabel, readOnly = false },
  ref,
) {
  const { t } = useTranslation()
  const valid = isValidJSON(value)
  const cmRef = useRef<ReactCodeMirrorRef>(null)
  const attributes = EditorView.contentAttributes.of({ "aria-label": ariaLabel })

  useImperativeHandle(ref, () => ({
    revealPath(path: string) {
      const view = cmRef.current?.view
      if (!view) return false
      const location = locateJsonPath(view.state.doc.toString(), path)
      if (!location) return false
      const pos = Math.min(location.index, view.state.doc.length)
      view.dispatch({
        selection: { anchor: pos, head: Math.min(pos + 1, view.state.doc.length) },
        scrollIntoView: true,
      })
      view.focus()
      return true
    },
    focus() {
      cmRef.current?.view?.focus()
    },
  }), [])

  return (
    <div className="flex flex-col gap-3">
      <CodeMirror
        ref={cmRef}
        value={value}
        onChange={onChange}
        extensions={[json(), attributes]}
        readOnly={readOnly}
        basicSetup={{ foldGutter: false, highlightActiveLine: !readOnly }}
      />
      {valid ? null : (
        <Alert variant="destructive">
          <AlertTitle>{t("config.invalidJSON")}</AlertTitle>
          <AlertDescription>{t("config.invalidJSONDescription")}</AlertDescription>
        </Alert>
      )}
    </div>
  )
})
