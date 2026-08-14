import { Database, PackageOpen, Server } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import type { PackageChangeInput } from "@/hooks/use-package-releases"

export interface PackageTemplate {
  id: "postgresql" | "redis" | "gocron"
  releaseName: string
  source: PackageChangeInput["source"]
  values: Record<string, unknown>
}

const secret = () => {
  const bytes = new Uint8Array(32)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (value) => value.toString(16).padStart(2, "0")).join("")
}

function createTemplate(id: PackageTemplate["id"]): PackageTemplate {
  if (id === "postgresql") return {
    id,
    releaseName: "postgresql",
    source: { chart: "oci://registry-1.docker.io/bitnamicharts/postgresql" },
    values: { auth: { username: "app", password: secret(), database: "app" }, primary: { persistence: { enabled: true, size: "8Gi" } } },
  }
  if (id === "redis") return {
    id,
    releaseName: "redis",
    source: { chart: "oci://registry-1.docker.io/bitnamicharts/redis" },
    values: { architecture: "replication", auth: { enabled: true, password: secret() }, replica: { replicaCount: 2 }, master: { persistence: { enabled: true, size: "8Gi" } } },
  }
  return {
    id,
    releaseName: "gocron",
    source: { chart: "gocron", repoUrl: "https://gocronx-team.github.io/gocron" },
    values: {
      replicaCount: 2,
      db: { engine: "postgres", host: "postgresql.default.svc.cluster.local", port: 5432, user: "gocron", password: "", database: "gocron" },
      managed: { authSecret: secret(), encryptionKey: secret(), admin: { username: "admin", password: secret().slice(0, 32), email: "admin@example.com" } },
      service: { type: "ClusterIP", port: 5920 },
    },
  }
}

const templates = [
  { id: "postgresql" as const, icon: Database },
  { id: "redis" as const, icon: Server },
  { id: "gocron" as const, icon: PackageOpen },
]

export function PackageTemplatesPanel({ onInstall }: { onInstall: (template: PackageTemplate) => void }) {
  const { t } = useTranslation()
  return <div className="grid gap-3 md:grid-cols-3">
    {templates.map(({ id, icon: Icon }) => <div key={id} className="rounded-md border p-4">
      <div className="flex items-center gap-2"><Icon className="size-5 text-blue-600" /><h2 className="font-medium">{t(`packageTemplates.${id}.name`)}</h2></div>
      <p className="mt-2 min-h-10 text-sm text-muted-foreground">{t(`packageTemplates.${id}.description`)}</p>
      <Button className="mt-4 w-full" variant="outline" onClick={() => onInstall(createTemplate(id))}>{t("packageTemplates.configure")}</Button>
    </div>)}
  </div>
}
