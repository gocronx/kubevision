---
sidebar_position: 6
title: Error Codes
---

# Error Codes

Most JSON API handlers return HTTP `200` with a business code in the response
envelope. Health checks and WebSocket handshake failures use conventional HTTP
status codes, so clients must handle both layers.

| Code | Meaning |
|------|---------|
| `0` | Success |
| `40001` | Required parameter missing |
| `40002` | Invalid parameter or request body |
| `40100` | Authentication required or invalid |
| `40101` | Token expired |
| `40102` | Two-factor verification required |
| `40103` | Two-factor verification failed |
| `40300` | Operation forbidden by authorization policy |
| `40400` | Requested record or Kubernetes resource not found |
| `40900` | Existing resource, version conflict, or concurrent operation |
| `42200` | Validation or Kubernetes dry-run failure |
| `50000` | Internal server error |
| `50200` | Kubernetes cluster unavailable |

Messages provide additional context but are not a stable machine-readable API.
Branch on `code`, and include the request ID from response metadata or headers
when reporting an internal error.

Do not retry validation, authentication, or authorization failures blindly.
Retry transient server and cluster failures with a bounded backoff, and re-fetch
a resource before retrying a version conflict.
