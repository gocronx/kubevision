import type { LucideIcon } from "lucide-react"
import {
  Box,
  Server,
  Layers,
  Radio,
  Play,
  Clock,
  Network,
  Globe,
  FileText,
  KeyRound,
  HardDrive,
  Database,
  Boxes,
  Monitor,
  FolderOpen,
  Activity,
  BarChart3,
  SlidersHorizontal,
} from "lucide-react"

export interface ColumnConfig {
  key: string
  label: string
  sortable?: boolean
}

export interface ResourceUIConfig {
  icon: LucideIcon
  category: string
  displayName: string
  columns: ColumnConfig[]
}

export const resourceUIConfig: Record<string, ResourceUIConfig> = {
  pods: {
    icon: Box,
    category: "workloads",
    displayName: "Pods",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "status", label: "Status", sortable: true },
      { key: "restarts", label: "Restarts", sortable: true },
      { key: "node", label: "Node", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  deployments: {
    icon: Server,
    category: "workloads",
    displayName: "Deployments",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "ready", label: "Ready", sortable: true },
      { key: "upToDate", label: "Up-to-date", sortable: true },
      { key: "available", label: "Available", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  statefulsets: {
    icon: Layers,
    category: "workloads",
    displayName: "StatefulSets",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "ready", label: "Ready", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  daemonsets: {
    icon: Radio,
    category: "workloads",
    displayName: "DaemonSets",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "desired", label: "Desired", sortable: true },
      { key: "current", label: "Current", sortable: true },
      { key: "ready", label: "Ready", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  jobs: {
    icon: Play,
    category: "workloads",
    displayName: "Jobs",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "completions", label: "Completions", sortable: true },
      { key: "duration", label: "Duration", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  cronjobs: {
    icon: Clock,
    category: "workloads",
    displayName: "CronJobs",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "schedule", label: "Schedule", sortable: false },
      { key: "suspend", label: "Suspend", sortable: true },
      { key: "lastSchedule", label: "Last Schedule", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  services: {
    icon: Network,
    category: "network",
    displayName: "Services",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "type", label: "Type", sortable: true },
      { key: "clusterIP", label: "Cluster IP", sortable: false },
      { key: "ports", label: "Ports", sortable: false },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  ingresses: {
    icon: Globe,
    category: "network",
    displayName: "Ingresses",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "hosts", label: "Hosts", sortable: false },
      { key: "address", label: "Address", sortable: false },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  configmaps: {
    icon: FileText,
    category: "config",
    displayName: "ConfigMaps",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "dataCount", label: "Data", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  secrets: {
    icon: KeyRound,
    category: "config",
    displayName: "Secrets",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "type", label: "Type", sortable: true },
      { key: "dataCount", label: "Data", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  persistentvolumes: {
    icon: HardDrive,
    category: "storage",
    displayName: "Persistent Volumes",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "capacity", label: "Capacity", sortable: true },
      { key: "accessModes", label: "Access Modes", sortable: false },
      { key: "reclaimPolicy", label: "Reclaim Policy", sortable: true },
      { key: "status", label: "Status", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  persistentvolumeclaims: {
    icon: Database,
    category: "storage",
    displayName: "PVCs",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "status", label: "Status", sortable: true },
      { key: "volume", label: "Volume", sortable: true },
      { key: "capacity", label: "Capacity", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  storageclasses: {
    icon: Boxes,
    category: "storage",
    displayName: "Storage Classes",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "provisioner", label: "Provisioner", sortable: true },
      { key: "reclaimPolicy", label: "Reclaim Policy", sortable: true },
      { key: "volumeBindingMode", label: "Binding Mode", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  nodes: {
    icon: Monitor,
    category: "cluster",
    displayName: "Nodes",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "status", label: "Status", sortable: true },
      { key: "roles", label: "Roles", sortable: false },
      { key: "version", label: "Version", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  namespaces: {
    icon: FolderOpen,
    category: "cluster",
    displayName: "Namespaces",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "status", label: "Status", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  events: {
    icon: Activity,
    category: "cluster",
    displayName: "Events",
    columns: [
      { key: "type", label: "Type", sortable: true },
      { key: "reason", label: "Reason", sortable: true },
      { key: "object", label: "Object", sortable: true },
      { key: "message", label: "Message", sortable: false },
      { key: "count", label: "Count", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  resourcequotas: {
    icon: BarChart3,
    category: "policy",
    displayName: "Resource Quotas",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "quotaStatus", label: "Hard / Used", sortable: false },
      { key: "age", label: "Age", sortable: true },
    ],
  },
  limitranges: {
    icon: SlidersHorizontal,
    category: "policy",
    displayName: "Limit Ranges",
    columns: [
      { key: "name", label: "Name", sortable: true },
      { key: "namespace", label: "Namespace", sortable: true },
      { key: "limitsCount", label: "Limits", sortable: true },
      { key: "age", label: "Age", sortable: true },
    ],
  },
}
