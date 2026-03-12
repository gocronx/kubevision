import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebars: SidebarsConfig = {
  docs: [
    "intro",
    {
      type: "category",
      label: "Getting Started",
      collapsed: false,
      items: [
        "getting-started/installation",
        "getting-started/quick-start",
        "getting-started/configuration",
      ],
    },
    {
      type: "category",
      label: "Architecture",
      items: [
        "architecture/overview",
        "architecture/data-flow",
        "architecture/directory-structure",
        "architecture/tech-stack",
      ],
    },
    {
      type: "category",
      label: "User Guide",
      items: [
        "user-guide/cluster-management",
        "user-guide/resource-crud",
        "user-guide/pod-terminal",
        "user-guide/pod-logs",
        "user-guide/global-search",
        "user-guide/resource-topology",
        "user-guide/cross-cluster-diff",
        "user-guide/dry-run",
        "user-guide/favorites",
        "user-guide/kubectl-hints",
      ],
    },
    {
      type: "category",
      label: "Admin Guide",
      items: [
        "admin-guide/rbac",
        "admin-guide/two-factor-auth",
        "admin-guide/audit-logging",
        "admin-guide/api-keys",
        "admin-guide/webhooks",
        "admin-guide/terminal-recording",
        "admin-guide/resource-quotas",
      ],
    },
    {
      type: "category",
      label: "Plugins",
      items: [
        "plugins/prometheus",
        "plugins/grafana",
        "plugins/argocd",
      ],
    },
    {
      type: "category",
      label: "API Reference",
      items: [
        "api/overview",
        "api/authentication",
        "api/resources",
        "api/websocket",
        "api/error-codes",
      ],
    },
    {
      type: "category",
      label: "Development",
      items: [
        "development/contributing",
        "development/building",
        "development/testing",
      ],
    },
    "comparison",
    "faq",
    "roadmap",
  ],
};

export default sidebars;
