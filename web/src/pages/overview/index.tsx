import { useTranslation } from "react-i18next"
import { Box, Server, Network, Monitor } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import type { LucideIcon } from "lucide-react"

interface StatCardProps {
  title: string
  icon: LucideIcon
}

function StatCard({ title, icon: Icon }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <Skeleton className="h-8 w-20" />
        <Skeleton className="mt-2 h-4 w-32" />
      </CardContent>
    </Card>
  )
}

export function OverviewPage() {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-bold tracking-tight">{t("overview.title")}</h1>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard title={t("overview.pods")} icon={Box} />
        <StatCard title={t("overview.deployments")} icon={Server} />
        <StatCard title={t("overview.services")} icon={Network} />
        <StatCard title={t("overview.nodes")} icon={Monitor} />
      </div>
    </div>
  )
}
