import { useMemo } from "react"
import { Activity } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useResourceList } from "@/hooks/use-resource"
import { formatAge } from "@/lib/k8s-utils"

/**
 * Resource kinds whose events are mostly generated for their child Pods rather
 * than the workload itself.  For these we fetch ALL events in the namespace and
 * filter client-side by name prefix (workload name → pods like "name-xxxxx").
 */
const WORKLOAD_KINDS = new Set([
  "deployments",
  "statefulsets",
  "daemonsets",
  "replicasets",
  "jobs",
  "cronjobs",
])

interface ResourceEventsProps {
  clusterID: string
  resource: string
  name: string
  namespace: string
}

export function ResourceEvents({
  clusterID,
  resource,
  name,
  namespace,
}: ResourceEventsProps) {
  const isWorkload = WORKLOAD_KINDS.has(resource)

  // For Pods and other simple resources we can use a precise fieldSelector.
  // For workloads (Deployment, DaemonSet, …) most events are on child Pods
  // whose names start with the workload name, so we fetch all namespace
  // events and filter client-side.
  const fieldSelector = isWorkload
    ? undefined
    : `involvedObject.name=${name}`

  const { data, isLoading } = useResourceList(clusterID, "events", {
    namespace,
    fieldSelector,
    enabled: !!clusterID && !!name,
  })

  const events = useMemo(() => {
    if (!data?.items) return []

    const namePrefix = `${name}-`

    const filtered = isWorkload
      ? data.items.filter((ev) => {
          const obj = ev.involvedObject as Record<string, unknown> | undefined
          const objName = (obj?.name as string) ?? ""
          // Match: the workload itself, or child objects whose name starts with "workload-"
          return objName === name || objName.startsWith(namePrefix)
        })
      : data.items

    return filtered.sort((a, b) => {
      const tsA = getEventTimestamp(a)
      const tsB = getEventTimestamp(b)
      return new Date(tsB).getTime() - new Date(tsA).getTime()
    })
  }, [data, name, isWorkload])

  if (isLoading) {
    return (
      <Card>
        <CardContent className="space-y-3 pt-6">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </CardContent>
      </Card>
    )
  }

  if (events.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center gap-2 py-12">
          <Activity className="size-8 text-muted-foreground/40" />
          <p className="text-sm text-muted-foreground">
            No events found for this resource
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardContent className="divide-y pt-4">
        {events.map((event, idx) => {
          const meta = event.metadata as Record<string, unknown> | undefined
          const obj = event.involvedObject as Record<string, unknown> | undefined
          const eventType = (event.type as string) ?? "Normal"
          const reason = (event.reason as string) ?? ""
          const message = (event.message as string) ?? ""
          const count = (event.count as number) ?? 1
          const lastTs = getEventTimestamp(event)
          const objKind = (obj?.kind as string) ?? ""
          const objName = (obj?.name as string) ?? ""

          return (
            <div key={(meta?.uid as string) ?? idx} className="flex gap-3 py-3">
              <Badge
                variant={eventType === "Warning" ? "destructive" : "secondary"}
                className="mt-0.5 shrink-0"
              >
                {eventType}
              </Badge>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 text-sm">
                  <span className="font-medium">{reason}</span>
                  {/* Show the involved object when it differs from the parent */}
                  {objName && objName !== name && (
                    <span className="text-xs text-muted-foreground">
                      {objKind}/{objName}
                    </span>
                  )}
                  {count > 1 && (
                    <span className="text-muted-foreground">×{count}</span>
                  )}
                  {lastTs && (
                    <span className="ml-auto shrink-0 text-xs text-muted-foreground">
                      {formatAge(lastTs)}
                    </span>
                  )}
                </div>
                <p className="mt-0.5 text-sm text-muted-foreground break-all">
                  {message}
                </p>
              </div>
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}

function getEventTimestamp(event: Record<string, unknown>): string {
  return (
    (event.lastTimestamp as string) ??
    (event.eventTime as string) ??
    ((event.metadata as Record<string, unknown> | undefined)
      ?.creationTimestamp as string) ??
    ""
  )
}
