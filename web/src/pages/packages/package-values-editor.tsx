import { useMemo, useState } from "react"
import { Eye, EyeOff, RefreshCw, Search } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  collectHelmValueFields,
  generateHelmSecret,
  isSensitiveHelmValue,
  setHelmValue,
  type HelmValueField,
  type HelmValuePrimitive,
} from "./package-values"

interface PackageValuesEditorProps {
  values: Record<string, unknown>
  onChange: (values: Record<string, unknown>) => void
}

export function PackageValuesEditor({ values, onChange }: PackageValuesEditorProps) {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const { fields, truncated } = useMemo(() => collectHelmValueFields(values, query), [query, values])

  if (fields.length === 0 && !query) {
    return <p className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">{t("packages.noGuidedValues")}</p>
  }

  return (
    <div className="space-y-3">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          aria-label={t("packages.searchValues")}
          placeholder={t("packages.searchValues")}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          className="pl-9"
        />
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        {fields.map((field) => (
          <ValueField
            key={field.label}
            field={field}
            onChange={(value) => onChange(setHelmValue(values, field.path, value))}
          />
        ))}
      </div>
      {fields.length === 0 && <p className="text-sm text-muted-foreground">{t("packages.noMatchingValues")}</p>}
      {truncated && <p className="text-xs text-muted-foreground">{t("packages.guidedValuesTruncated")}</p>}
    </div>
  )
}

function ValueField({ field, onChange }: { field: HelmValueField; onChange: (value: HelmValuePrimitive) => void }) {
  const { t } = useTranslation()
  const [showSensitive, setShowSensitive] = useState(false)
  const sensitive = isSensitiveHelmValue(field.path)

  if (typeof field.value === "boolean") {
    return (
      <label className="flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm">
        <input type="checkbox" checked={field.value} onChange={(event) => onChange(event.target.checked)} />
        <span className="break-all font-mono text-xs">{field.label}</span>
      </label>
    )
  }

  return (
    <label className="space-y-1.5">
      <span className="block break-all font-mono text-xs text-muted-foreground">{field.label}</span>
      <div className="flex gap-1">
        <Input
          aria-label={field.label}
          type={typeof field.value === "number" ? "number" : sensitive && !showSensitive ? "password" : "text"}
          value={field.value ?? ""}
          onChange={(event) => onChange(typeof field.value === "number" ? Number(event.target.value) : event.target.value)}
        />
        {sensitive && (
          <>
            <Button type="button" variant="outline" size="icon" title={t("packages.toggleSecretVisibility")} onClick={() => setShowSensitive((value) => !value)}>
              {showSensitive ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </Button>
            <Button type="button" variant="outline" size="icon" title={t("packages.generateSecret")} onClick={() => onChange(generateHelmSecret())}>
              <RefreshCw className="size-4" />
            </Button>
          </>
        )}
      </div>
    </label>
  )
}
