---
title: User Management
---

# User Management

Administrators manage local and externally linked users under
**Administration > Users**. The workspace supports creating users, changing
roles and status, resetting local passwords, deleting users, and revoking a
user's passkey or security-key credential.

Use the least-privileged role that covers the user's cluster and namespace
responsibilities. Directory users receive their mapped role at login; changing
directory policy revokes affected directory sessions so the new mapping is
applied on the next login.

Password resets and authentication changes should be communicated through a
trusted channel. Administrators cannot register a passkey for another user.
Avoid deleting the final usable administrator account or removing a user's last
authentication and recovery method.

User, password, role, and credential changes are mutating administrative
operations and are captured by audit logging when auditing is enabled.
