import { ShieldAlertIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Separator } from "@/components/ui/separator"
import {
  certificateSourceFields,
  certificateStoreFields,
  hasCertificateSources,
} from "@/features/advanced/certificate-form-model"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import type { JsonObject } from "@/features/policy/policy-form-model"

export interface CertificateVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

function TrustStoreWarning({ hasSources }: { hasSources: boolean }) {
  const { t } = useTranslation()
  return <Alert variant={hasSources ? "default" : "destructive"}>
    <ShieldAlertIcon />
    <AlertTitle>{t(hasSources ? "advanced.certificate.customOnlyTitle" : "advanced.certificate.emptyTrustTitle")}</AlertTitle>
    <AlertDescription>{t(hasSources ? "advanced.certificate.customOnlyDescription" : "advanced.certificate.emptyTrustDescription")}</AlertDescription>
  </Alert>
}

export function CertificateVisualEditor(props: CertificateVisualEditorProps) {
  const { t } = useTranslation()
  const { object, revision, onChange, onFieldValidityChange } = props
  const hasSources = hasCertificateSources(object)
  const shared = { object, revision, onChange, onFieldValidityChange }
  return <div className="flex flex-col gap-4">
    <section className="flex flex-col gap-3">
      <header className="flex flex-col gap-1"><h2 className="text-base font-semibold">{t("advanced.certificate.storeTitle")}</h2>
        <p className="text-sm text-muted-foreground">{t("advanced.certificate.storeDescription")}</p></header>
      <PolicyFormFields fields={certificateStoreFields} namespace="advanced.certificate" {...shared} />
      {object.store === "none" ? <TrustStoreWarning hasSources={hasSources} /> : null}
    </section>
    <Separator />
    <section className="flex flex-col gap-3">
      <header className="flex flex-col gap-1"><h2 className="text-base font-semibold">{t("advanced.certificate.sourcesTitle")}</h2>
        <p className="text-sm text-muted-foreground">{t("advanced.certificate.sourcesDescription")}</p></header>
      <PolicyFormFields fields={certificateSourceFields} namespace="advanced.certificate" {...shared} />
    </section>
  </div>
}
