---
title: AI Operations Workspace
---

# AI Operations Workspace

The AI workspace is the primary natural-language operating surface in
KubeVision. It combines the current page context with live Kubernetes tools so
answers can be grounded in the selected cluster instead of relying only on
general model knowledge. It is available as a side panel or a full-tab chat
workspace.

## What It Can Do

- Read one Kubernetes resource or list resources by kind and label
- Read bounded recent Pod logs, including a previous container instance
- Summarize cluster health and identify Pods outside Running/Succeeded states
- Query a configured Prometheus instance
- Prepare create, update, patch, and delete operations for approval
- Stream explanations and tool results while a multi-step investigation runs

## Configure a Provider

Open **Settings > AI Assistant**. Administrators can enter an endpoint and
credentials, fetch available models, and test the configuration. Credentials
are encrypted at rest and are not returned by the API after saving.

KubeVision uses OpenAI-compatible chat and model APIs. This supports providers
such as OpenAI, OpenRouter, DeepSeek, Qwen, and compatible self-hosted gateways.
Compatibility depends on the provider implementing streaming and tool calling
correctly.

Only configure a service you trust. User prompts, page context, resource data,
log excerpts, metrics results, and tool outputs used in a conversation are sent
to that provider. A self-hosted compatible endpoint can be used when cluster
data must remain on your network.

## Guarded Tool Actions

Read operations are restricted to resources the current user can access.
Mutating tools use the same RBAC rules as the regular UI and pause for explicit
confirmation before execution. The server re-checks permission when the user
confirms, so an expired approval or changed role cannot reuse the earlier
authorization decision. Approved mutations generate audit entries.

Review the target cluster, namespace, resource, manifest or patch, and generated
arguments before confirming. Model output is not a substitute for a rollout,
backup, disruption, or data-retention plan.

When a request requires several Kubernetes steps, the assistant may continue
after tool results are returned. Keep the chat open until it reports completion
or a concrete failure; interrupted confirmation or expired authentication can
stop the sequence.

## Display Modes

Use the layout control to switch between the side panel and full-tab workspace.
The conversation remains available when switching modes. Full-tab mode is
better for long investigations; the panel keeps the current resource visible.

Conversations and drafts are stored in the browser, scoped to the signed-in
user, and restored after logout or a later login on the same browser. They are
not synchronized through the KubeVision server or shared with another browser.
Clear the site's browser data to remove them. Avoid leaving sensitive cluster
content on a shared device.

## Current Boundaries

- The assistant does not run autonomous background remediation.
- It operates on one selected cluster per conversation turn.
- Only the tools listed above are available; terminal execution and arbitrary
  network access are not exposed to the model.
- A conversation run has bounded tool iterations and bounded log/tool output to
  prevent runaway loops and oversized model context.
