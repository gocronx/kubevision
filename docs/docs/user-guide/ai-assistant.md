---
title: AI Assistant
---

# AI Assistant

The AI assistant can inspect accessible Kubernetes state, explain failures,
summarize logs, generate manifests, and propose operational actions from the
dashboard. It is available as a side panel or a full-tab chat workspace.

## Configure a Provider

Open **Settings > AI Assistant**. Administrators can select a supported
provider, enter its endpoint and credentials, fetch available models, and test
the configuration. Credentials are encrypted at rest and are not returned by
the API after saving.

Only configure a service you trust. Resource details and log excerpts included
in a conversation are sent to that provider. A local OpenAI-compatible endpoint
can be used when cluster data must remain on your network.

## Tool Actions

Read operations are restricted to resources the current user can access.
Mutating tools use the same RBAC rules as the regular UI and pause for explicit
confirmation before execution. Review the target cluster, namespace, resource,
and generated arguments before confirming.

When a request requires several Kubernetes steps, the assistant may continue
after tool results are returned. Keep the chat open until it reports completion
or a concrete failure; interrupted confirmation or expired authentication can
stop the sequence.

## Display Modes

Use the layout control in the assistant to switch between the side panel and
the full-tab workspace. The conversation remains available when switching
modes. Full-tab mode is better for long investigations; the panel keeps the
current resource visible alongside the chat.
