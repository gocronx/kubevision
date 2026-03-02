import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"

export function NotFoundPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-4 bg-background">
      <h1 className="text-6xl font-bold">{t("notFound.title")}</h1>
      <p className="text-lg text-muted-foreground">{t("notFound.description")}</p>
      <Button onClick={() => navigate("/overview")}>{t("notFound.back")}</Button>
    </div>
  )
}
