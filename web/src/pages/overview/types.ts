export interface ResourceMetric {
  allocatable: number
  usage: number
  requests: number
  limits: number
}

export interface ResourceUsage {
  cpu: ResourceMetric
  memory: ResourceMetric
  metricsAvailable: boolean
}

export interface EventSummary {
  type: string
  reason: string
  message: string
  objectKind: string
  objectName: string
  namespace?: string
  timestamp: string
}

export interface PodStatusDist {
  running: number
  pending: number
  succeeded: number
  failed: number
  unknown: number
}

export interface OverviewData {
  pods: number
  runningPods: number
  deployments: number
  readyDeployments: number
  services: number
  nodes: number
  readyNodes: number
  namespaces: number
  activeNamespaces: number
  resources: ResourceUsage
  recentEvents: EventSummary[]
  statefulSets: number
  readyStatefulSets: number
  daemonSets: number
  readyDaemonSets: number
  jobs: number
  succeededJobs: number
  failedJobs: number
  cronJobs: number
  activeCronJobs: number
  ingresses: number
  persistentVolumes: number
  boundPVs: number
  availablePVs: number
  releasedPVs: number
  persistentVolumeClaims: number
  boundPVCs: number
  pendingPVCs: number
  totalStorageBytes: number
  allocatedStorageBytes: number
  usedStorageBytes: number
  podStatusDistribution: PodStatusDist
}

export const EMPTY_RESOURCES: ResourceUsage = {
  cpu: { allocatable: 0, usage: 0, requests: 0, limits: 0 },
  memory: { allocatable: 0, usage: 0, requests: 0, limits: 0 },
  metricsAvailable: false,
}

export const EMPTY_POD_STATUS: PodStatusDist = {
  running: 0,
  pending: 0,
  succeeded: 0,
  failed: 0,
  unknown: 0,
}
