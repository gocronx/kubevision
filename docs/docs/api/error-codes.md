---
sidebar_position: 5
title: Error Codes
---

# Error Codes

All API responses use HTTP 200. The `code` field in the response envelope determines the outcome. `0` always means success. Every non-zero code signals a specific failure that the client should handle.

## Code Reference

| Code | Category | Meaning |
|------|----------|---------|
| `0` | Success | Request succeeded |
| **4000x — Parameter errors** | | |
| `40001` | Param | Required parameter is missing |
| `40002` | Param | Parameter format is invalid (wrong type, malformed value) |
| **4010x — Authentication errors** | | |
| `40100` | Auth | Not logged in — no token provided |
| `40101` | Auth | Access token has expired — use the refresh endpoint |
| `40102` | Auth | 2FA verification required — provide TOTP code |
| `40103` | Auth | 2FA verification failed — incorrect or expired TOTP code |
| **4030x — Permission errors** | | |
| `40300` | Permission | User does not have permission for this action |
| `40301` | Permission | User does not have access to the requested cluster |
| `40302` | Permission | User does not have access to the requested namespace |
| **4040x — Not found errors** | | |
| `40400` | Not Found | Kubernetes resource not found in the cluster |
| `40401` | Not Found | KubeVision user not found |
| `40402` | Not Found | Cluster configuration not found |
| **4090x — Conflict errors** | | |
| `40900` | Conflict | Resource already exists (create collision) |
| `40901` | Conflict | Resource version conflict — re-fetch and retry the update |
| **4220x — Validation errors** | | |
| `42200` | Validation | YAML is syntactically invalid or cannot be parsed |
| `42201` | Validation | Dry-run rejected by the Kubernetes API Server |
| **5000x / 5020x — Server errors** | | |
| `50000` | Server | Internal server error — check backend logs with the `requestId` |
| `50200` | Server | Kubernetes API Server is unreachable for the requested cluster |

## Handling Errors in the Frontend

The recommended pattern is a single Axios response interceptor that maps `code` values to user-facing actions. The example below is taken from the KubeVision frontend source:

```typescript
import axios from "axios";
import { useAuthStore } from "@/store/auth";
import { toast } from "@/components/ui/use-toast";

axios.interceptors.response.use(
  (response) => {
    const { code, message, data } = response.data;

    if (code === 0) return response;

    // Token expired — attempt silent refresh, then retry
    if (code === 40101) {
      return useAuthStore.getState().refreshAndRetry(response.config);
    }

    // 2FA required — redirect to the 2FA prompt
    if (code === 40102) {
      window.location.href = "/auth/2fa";
      return Promise.reject(new Error(message));
    }

    // Not logged in — redirect to login
    if (code === 40100) {
      useAuthStore.getState().logout();
      return Promise.reject(new Error(message));
    }

    // All other errors — show a toast and reject
    toast({ variant: "destructive", title: `Error ${code}`, description: message });
    return Promise.reject(new Error(message));
  }
);
```

:::tip
Always include the `requestId` from `meta.requestId` when reporting a `50000` error. It allows the backend team to locate the exact log entry immediately.
:::

:::warning
Never surface raw `50000` error details to end users. The `message` field for server errors is intentionally generic. Full error context is only available in the backend structured logs.
:::

## Related

- [API Overview](/docs/api/overview) — Unified response format and the `meta` object
- [Authentication](/docs/api/authentication) — `401xx` auth flow details
- [RBAC](/docs/admin-guide/rbac) — How `403xx` permission errors map to role definitions
