<div align="center">
  <img src="docs/assets/logo.svg" alt="KubeVision Logo" width="128" height="128">
  <h1>KubeVision</h1>
  <p>A modern, real-time Kubernetes dashboard. See your clusters clearly, act on them instantly.</p>
  <p>
    <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
    <a href="https://react.dev"><img src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React"></a>
    <a href="https://typescriptlang.org"><img src="https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="License"></a>
  </p>
  <p>English · <a href="README_CN.md">简体中文</a></p>
</div>

<p align="center">
  <img src="assets/overview.png" alt="KubeVision Dashboard" width="100%">
</p>

<table>
  <tr>
    <td width="50%" align="center"><b>Pod Terminal & Logs</b></td>
    <td width="50%" align="center"><b>Resource Topology</b></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-terminal.png" alt="Pod Terminal" width="100%"></td>
    <td><img src="docs/assets/screenshot-topology.png" alt="Resource Topology" width="100%"></td>
  </tr>
  <tr>
    <td width="50%" align="center"><b>Global Search</b></td>
    <td width="50%" align="center"><b>Terminal Audit</b></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-search.png" alt="Global Search" width="100%"></td>
    <td><img src="docs/assets/screenshot-audit.png" alt="Terminal Audit" width="100%"></td>
  </tr>
</table>


## 📖 Documentation

Full documentation is available at: **[kubevision-docs](https://kubevision-docs.pages.dev/)**

- 🚀 [Quick Start](https://kubevision-docs.pages.dev/docs/getting-started/installation) - Installation, quick start, configuration
- 🏛️ [Architecture](https://kubevision-docs.pages.dev/docs/architecture/overview) - System design, data flow, component interactions
- 📘 [User Guide](https://kubevision-docs.pages.dev/docs/user-guide/cluster-management) - Features walkthrough and usage guides
- 🔌 [API Reference](https://kubevision-docs.pages.dev/docs/api/overview) - REST & WebSocket API documentation

## ✨ Features

- **Real-time Sync**: Sub-second updates with zero polling via Informer to WebSocket Push architecture
- **Multi-cluster**: Manage all your clusters efficiently from a single, unified dashboard
- **Pod Terminal & Logs**: Integrated terminal emulator with fully recorded and replayable sessions
- **Resource Topology**: Visual ownership graph displaying real-time relationships for workloads
- **Global Search**: `Cmd+K` fuzzy search across your entire cluster ecosystem
- **Deployment Ops**: Seamlessly scale, restart, rollback, and preview diffs before applying changes
- **AI Assistant**: Natural-language cluster inspection and mutation (Supports OpenAI, DeepSeek, Qwen, etc.)
- **Security & Access Control**: 5 built-in RBAC roles, TOTP 2FA, and asynchronous audit logging
- **Fully Extensible**: 26+ built-in resources with CRDs that are auto-discovered at runtime
- **Modern UI**: Full i18n localization and native dark mode support

## 🚀 Quick Start

The fastest way to deploy KubeVision in your cluster is via Helm:

```bash
# 1. Install KubeVision
helm install kubevision deploy/helm/kubevision

# 2. Port-forward to access the dashboard
kubectl port-forward svc/kubevision 8080:80

# 3. Access Web Interface
# http://localhost:8080
```

> Default login credentials: `admin` / `admin123`
> 
> For other deployment methods (Docker, Binary, Development), please refer to the [Installation Guide](https://kubevision-docs.pages.dev/docs/getting-started/installation).

## 🔷 CLI & AI Assistant

The `kubevision` binary doubles as a powerful administrative CLI. You can manage accounts natively or chat with your cluster from the shell:

```bash
# Account management
kubevision reset-password --username admin
kubevision create-user --username dev --role editor --email dev@example.com

# Terminal AI assistant (chat with your cluster)
export API_KEY=... MODEL_ID=gpt-4o-mini
kubevision ai "why is the web pod crashing in default?"   # one-shot query
kubevision ai                                              # interactive REPL
```

## 🤝 Contributing

Contributions are welcome! Please open an issue first to discuss what you would like to change.

One thing to note: commit messages are validated by a git hook
([commitlint](https://github.com/conventional-changelog/commitlint)), so use the
interactive commit tool instead of `git commit`:

```bash
pnpm install      # first-time setup (installs git hooks in the web directory)
pnpm run commit   # create a properly formatted commit
```

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
