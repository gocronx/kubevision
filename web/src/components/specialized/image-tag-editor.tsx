import { useId } from "react"
import { RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useRegistryTags } from "@/hooks/use-registry-tags"
import { setContainerImage } from "@/lib/container-images"

interface ContainerImage {
  path: string[]
  name: string
  image: string
}

function containerImages(document: Record<string, unknown>): ContainerImage[] {
  const spec = document.spec as Record<string, unknown> | undefined
  const podSpec = (spec?.template as Record<string, unknown> | undefined)?.spec as Record<string, unknown> | undefined
    ?? spec
  const result: ContainerImage[] = []
  for (const group of ["containers", "initContainers"] as const) {
    const containers = podSpec?.[group]
    if (!Array.isArray(containers)) continue
    containers.forEach((value, index) => {
      if (!value || typeof value !== "object") return
      const container = value as Record<string, unknown>
      if (typeof container.image !== "string") return
      const prefix = podSpec === spec ? ["spec"] : ["spec", "template", "spec"]
      result.push({
        path: [...prefix, group, String(index), "image"],
        name: typeof container.name === "string" ? container.name : `${group} ${index + 1}`,
        image: container.image,
      })
    })
  }
  return result
}

function withTag(image: string, tag: string): string {
  const withoutDigest = image.split("@", 1)[0]
  const slash = withoutDigest.lastIndexOf("/")
  const colon = withoutDigest.lastIndexOf(":")
  const repository = colon > slash ? withoutDigest.slice(0, colon) : withoutDigest
  return `${repository}:${tag}`
}

function ImageField({ container, onChange }: { container: ContainerImage; onChange: (image: string) => void }) {
  const listID = useId()
  const query = useRegistryTags(container.image)
  return (
    <div className="grid min-w-0 grid-cols-[minmax(7rem,0.35fr)_minmax(0,1fr)_2rem] items-center gap-2">
      <Label htmlFor={`${listID}-input`} className="truncate text-xs" title={container.name}>{container.name}</Label>
      <Input
        id={`${listID}-input`}
        list={listID}
        value={container.image}
        onChange={(event) => {
          const value = event.target.value
          const discovered = query.data?.tags.includes(value)
          onChange(discovered ? withTag(container.image, value) : value)
        }}
        className="h-8 font-mono text-xs"
      />
      <datalist id={listID}>
        {query.data?.tags.map((tag) => <option key={tag} value={tag} />)}
      </datalist>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        title={query.isError ? "Tag discovery unavailable" : "Refresh image tags"}
        aria-label="Refresh image tags"
        onClick={() => void query.refetch()}
        disabled={query.isFetching}
      >
        <RefreshCw className={query.isFetching ? "animate-spin" : ""} />
      </Button>
    </div>
  )
}

export function ImageTagEditor({ json, onChange }: { json: string; onChange: (json: string) => void }) {
  let containers: ContainerImage[]
  try {
    containers = containerImages(JSON.parse(json) as Record<string, unknown>)
  } catch {
    return null
  }
  if (containers.length === 0) return null
  return (
    <div className="flex flex-col gap-2 border-y py-3" data-testid="image-tag-editor">
      {containers.map((container) => (
        <ImageField
          key={container.path.join(".")}
          container={container}
          onChange={(image) => onChange(setContainerImage(json, container.path, image))}
        />
      ))}
    </div>
  )
}
