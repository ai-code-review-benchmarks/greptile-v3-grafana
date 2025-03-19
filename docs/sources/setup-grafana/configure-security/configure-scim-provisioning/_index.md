---
description: Learn how to use SCIM provisioning to synchronize users and groups from your identity provider to Grafana. SCIM enables automated user management, team provisioning, and enhanced security through real-time synchronization with your identity provider.
keywords:
  - grafana
  - scim
  - provisioning
  - user-management
  - team-management
labels:
  products:
    - cloud
    - enterprise
menuTitle: Configure SCIM provisioning
title: Configure SCIM provisioning
weight: 300
---

# Configure SCIM provisioning

System for Cross-domain Identity Management (SCIM) is an open standard that allows automated user provisioning and management. With SCIM, you can automate the provisioning of users and groups from your identity provider to Grafana.

{{< admonition type="note" >}}
Available in [Grafana Enterprise](../../../introduction/grafana-enterprise/) and [Grafana Cloud Advanced](/docs/grafana-cloud/).
{{< /admonition >}}

{{< admonition type="note" >}}
This feature is behind the `enableSCIM` feature toggle.
You can enable feature toggles through configuration file or environment variables.

For more information, refer to the [feature toggles documentation](/docs/grafana/<GRAFANA_VERSION>/setup-grafana/configure-grafana/#feature_toggles).
{{< /admonition >}}

## Benefits

{{< admonition type="note" >}}
SCIM provisioning only works SAML authentication.
Other authentication methods aren't supported.
{{< /admonition >}}

SCIM offers several advantages for managing users and teams in Grafana:

- **Automated user provisioning**: Automatically create, update, and disable users in Grafana when changes occur in your identity provider
- **Automated team provisioning**: Automatically create, update, and delete teams in Grafana based on groups in your identity provider
- **Reduced administrative overhead**: Eliminate manual user management tasks and reduce the risk of human error
- **Enhanced security**: Automatically disable access when users leave your organization

## Supported identity providers

The following identity providers are supported:

- [Azure AD](../configure-authentication/azuread/)
- [Okta](../configure-authentication/saml/)

## How it works

The synchronization process works as follows:

1. Configure SCIM in both your identity provider and Grafana
2. Your identity provider sends SCIM requests to the Grafana SCIM API endpoint
3. Grafana processes these requests to create, update, or deactivate users and teams, and synchronize team memberships

## Comparison with other sync methods

Grafana offers several methods for synchronizing users, teams, and roles.
The following table compares SCIM with other synchronization methods to help you understand its advantages:

| Sync Method                                                                    | Users | Teams | Roles | Automation | Key Benefits                                                             | Limitations                                                  | On-Prem | Cloud |
| ------------------------------------------------------------------------------ | ----- | ----- | ----- | ---------- | ------------------------------------------------------------------------ | ------------------------------------------------------------ | ------- | ----- |
| SCIM                                                                           | ✅    | ✅    | ⚠️    | Partial    | Complete user and team lifecycle management with automatic team creation | Requires SAML authentication; uses Role Sync for basic roles | ✅      | ✅    |
| [Team Sync](../configure-team-sync/)                                           | ❌    | ✅    | ❌    | Partial    | Maps identity provider groups to Grafana teams                           | Requires manual team creation                                | ✅      | ✅    |
| [Active LDAP Sync](../configure-authentication/enhanced-ldap/)                 | ✅    | ❌    | ❌    | Full       | Background synchronization of LDAP users                                 | Limited to LDAP environments                                 | ✅      | ❌    |
| [Group Attribute Sync](../configure-group-attribute-sync/)                     | ❌    | ❌    | ✅    | Partial    | Maps identity provider group attributes to permissions                   | Limited to identity provider attributes                      | ✅      | ✅    |
| [Role Sync](../configure-authentication/saml#configure-role-sync)              | ❌    | ❌    | ✅    | Partial    | Maps basic roles to users                                                | Limited to basic roles only                                  | ✅      | ✅    |
| [Org Mapping](../configure-authentication/saml#configure-organization-mapping) | ❌    | ❌    | ✅    | Partial    | Maps basic roles per organization                                        | Only available for on-premises deployments                   | ✅      | ❌    |

### Key advantages

- **Complete automation**: SCIM is the only method that fully automates user and team provisioning
- **Dynamic team creation**: Teams are created automatically based on identity provider groups
- **Near real-time synchronization**: Changes in your identity provider are reflected based on the provider's synchronization schedule
- **Enterprise-ready**: Designed for large organizations with complex user management needs

## Next steps

- [Manage users and teams with SCIM provisioning](manage-users-teams/)
- [Configure SCIM with Azure AD](azuread/)
- [Configure SCIM with Okta](okta/)
