import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function PagePlaceholder({ titleKey }: { titleKey: string }) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle role="heading" aria-level={1}>{t(titleKey)}</CardTitle>
        <CardDescription>{t("pages.pending")}</CardDescription>
      </CardHeader>
      <CardContent />
    </Card>
  )
}
