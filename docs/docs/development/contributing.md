---
sidebar_position: 1
title: Contributing
---

# Contributing

KubeVision welcomes contributions of all kinds — bug reports, documentation improvements, feature implementations, and code reviews. All contributions are governed by the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0).

## Fork → Branch → PR Workflow

```bash
# 1. Fork the repository on GitHub, then clone your fork
git clone https://github.com/<your-username>/kubevision.git
cd kubevision

# 2. Add the upstream remote
git remote add upstream https://github.com/kubevision/kubevision.git

# 3. Create a feature branch from main
git checkout -b feat/my-feature

# 4. Make changes, commit, and push
git push origin feat/my-feature

# 5. Open a Pull Request on GitHub targeting main
```

:::tip
Keep branches focused on a single concern. Smaller PRs are reviewed faster and are easier to revert if needed.
:::

## Code Style

### Go (Backend)

| Tool | Usage |
|------|-------|
| `gofmt` | Canonical formatting — run before every commit |
| `go vet` | Catches common mistakes |
| `golangci-lint` | Full lint suite (run via `make lint`) |

```bash
# Format and vet all packages
gofmt -w ./...
go vet ./...

# Full lint (requires golangci-lint installed)
make lint
```

### Frontend (React + TypeScript)

| Tool | Config file |
|------|-------------|
| ESLint | `web/.eslintrc.cjs` |
| Prettier | `web/.prettierrc` |

```bash
cd web
pnpm lint        # ESLint check
pnpm lint:fix    # Auto-fix ESLint issues
pnpm format      # Prettier format
```

:::warning
PRs with lint failures will not be merged. Run `make lint` and `pnpm lint` locally before pushing.
:::

## Commit Message Convention

KubeVision follows [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>

[optional body]

[optional footer]
```

| Type | When to use |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `refactor` | Code change that is not a fix or feature |
| `test` | Adding or updating tests |
| `chore` | Build, tooling, dependency updates |

**Examples:**

```
feat(rbac): add custom role permission editor
fix(websocket): reconnect on 1006 close code
docs(contributing): add commit message guide
```

## Issue Templates

Use the appropriate GitHub issue template when filing a report:

- **Bug Report** — Steps to reproduce, expected vs actual behavior, KubeVision version, and Kubernetes version.
- **Feature Request** — Motivation, proposed solution, and any alternatives considered.
- **Security Vulnerability** — Do **not** open a public issue. Email `security@kubevision.io` instead.

## Code Review Process

1. A maintainer will be assigned within **2 business days** of PR submission.
2. Address all review comments with either code changes or a written explanation.
3. Once approved by one maintainer, a second maintainer merges.
4. Squash-merge is the default strategy to keep `main` history clean.

:::note
First-time contributors must sign the CLA (Contributor License Agreement) via the automated GitHub bot before a PR can be merged.
:::

## License

By contributing to KubeVision you agree that your contributions are licensed under the **Apache License, Version 2.0**. See [LICENSE](https://github.com/kubevision/kubevision/blob/main/LICENSE) for the full text.

## Related

- [Building](/docs/development/building) — Set up your local build environment
- [Testing](/docs/development/testing) — Run and write tests
