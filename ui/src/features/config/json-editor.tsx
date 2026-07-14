import { json } from "@codemirror/lang-json"
import { EditorView } from "@codemirror/view"
import CodeMirror from "@uiw/react-codemirror"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { isValidJSON } from "@/features/config/json-utils"

interface JsonEditorProps {
  value: string
  onChange: (value: string) => void
  ariaLabel: string
  readOnly?: boolean
}

export function JsonEditor({ value, onChange, ariaLabel, readOnly = false }: JsonEditorProps) {
  const { t } = useTranslation()
  const valid = isValidJSON(value)
  const attributes = EditorView.contentAttributes.of({ "aria-label": ariaLabel })
  return (
    <div className="flex flex-col gap-3">
      <CodeMirror
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
}
