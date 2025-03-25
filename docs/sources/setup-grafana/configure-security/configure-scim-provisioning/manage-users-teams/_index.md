---
description: Learn how to implement SCIM provisioning in Grafana for automated user and team synchronization. SCIM integrates with identity providers like Okta and Azure AD to streamline user management, automate team provisioning, and replace Team Sync.
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
menuTitle: Manage users and teams with SCIM
title: Manage users and teams with SCIM
weight: 310
---

# Manage users and teams with SCIM

{{< admonition type="note" >}}
Available in [Grafana Enterprise](../../../introduction/grafana-enterprise/) and [Grafana Cloud Advanced](/docs/grafana-cloud/).
{{< /admonition >}}

SCIM streamlines identity management in Grafana by automating user lifecycle and team membership operations. This guide explains how SCIM works with existing Grafana setups, handles user provisioning, and manages team synchronization.

With SCIM, you can:

- **Automate user lifecycle** from creation to deactivation
- **Manage existing users** by linking them with identity provider identities
- **Synchronize team memberships** based on identity provider group assignments
- **Maintain security** through automated deprovisioning
- **Replace Team Sync** with more robust SCIM group synchronization

## User provisioning with SCIM

SCIM provisioning works in conjunction with existing user management methods in Grafana. While SCIM automates user provisioning from the identity provider, users can still be created through SAML just-in-time provisioning when they log in, manually through the Grafana UI, or via automation tools like Terraform and the Grafana API. For the most consistent user management experience, we recommend centralizing user provisioning through SCIM.

For detailed configuration steps specific to the identity provider, see:

- [Configure SCIM with Azure AD](../configure-scim-azure/)
- [Configure SCIM with Okta](../configure-scim-okta/)

### How SCIM identifies users

SCIM uses a specific process to establish and maintain user identity between the identity provider and Grafana:

1. Initial user lookup:
   - The identity provider looks up users in Grafana using the user's login and the Unique identifier field (configurable at IdP)
   - The identity provider expects a single result from Grafana for each user

2. Identity linking:
   - The identity provider learns the relationship between the found Grafana user and Grafana's internal ID
   - The identity provider updates Grafana with the External ID
   - Grafana updates its authentication validations to expect this External ID

3. Authentication validation:
   - Grafana expects the SAML integration to return the same External ID in SAML assertions
   - This External ID is used to validate that the logged-in user matches the provisioned user

This process ensures secure and consistent user identification across both systems, preventing security issues that could arise from email changes or other user attribute modifications.

### Existing Grafana users

{{< admonition type="note" >}}
Existing users must be assigned to the Grafana app in the identity provider to maintain access once SCIM is enabled.
{{< /admonition >}}

For users who already exist in the Grafana instance:

- SCIM establishes the relationship through the External ID matching process
- Creates a secure link with the identity provider identity
- Preserves all existing settings and access
- Keeps the account active and unchanged until assigned in the identity provider

#### Handling users from other provisioning methods

To prevent conflicts and maintain consistent user management, disable or restrict other provisioning methods when implementing SCIM. This ensures that all new users are created through SCIM and prevents duplicate or conflicting user records.

- SAML Just-in-Time (JIT) provisioning:

  - Disable `allow_sign_up` in SAML settings to prevent automatic user creation
  - Existing JIT-provisioned users will continue to work but should be migrated to SCIM

- Terraform or API provisioning:

  - Stop creating new users through these methods
  - Existing users will continue to work but should be migrated to SCIM
  - Consider removing or archiving Terraform user creation resources

- Manual user creation:
  - Restrict UI-based user creation to administrators only
  - Plan to migrate manually created users to SCIM

### New users

For users who don't yet exist in Grafana:

- SCIM creates accounts when users are assigned to Grafana in the identity provider
- Sets up initial access based on identity provider group memberships and SAML role mapping
- No manual Grafana account creation needed

### Role management

SCIM handles user synchronization but not role assignments. Role management is handled through [Role Sync](../../configure-authentication/saml#configure-role-sync), and any role changes take effect during user authentication.

## Team provisioning with SCIM

SCIM provides automated team management capabilities that go beyond what Team Sync offers. While Team Sync only maps identity provider groups to existing Grafana teams, SCIM can automatically create and delete teams based on group changes in the identity provider.

For detailed configuration steps specific to the identity provider, see:

- [Configure SCIM with Azure AD](../configure-scim-azure/)
- [Configure SCIM with Okta](../configure-scim-okta/)

### SCIM vs Team Sync

{{< admonition type="warning" >}}
Do not enable both SCIM and Team Sync simultaneously as these methods can conflict with each other.
{{< /admonition >}}

Choose one synchronization method:

- If you enable SCIM, disable Team Sync and use SCIM for team management
- If you prefer Team Sync, do not enable SCIM provisioning

### Key differences

SCIM Group Sync provides several advantages over Team Sync:

- **Automatic team creation:** SCIM automatically creates Grafana teams when new groups are added to the identity provider
- **Automatic team deletion:** SCIM removes teams when their corresponding groups are deleted from the identity provider
- **Real-time updates:** Team memberships are updated immediately when group assignments change
- **Simplified management:** No need to manually create teams in Grafana before mapping them

### How team synchronization works

SCIM manages teams through the following process:

Group assignment:

- User is assigned to groups in the identity provider
- SCIM detects group membership changes

Team creation and mapping:

- Creates Grafana teams for new identity provider groups
- Maps users to appropriate teams
- Removes users from teams when group membership changes

Team membership maintenance:

- Continuously syncs team memberships
- Removes users from teams when removed from groups
- Updates team memberships when groups change

### Migrating from Team Sync to SCIM

When transitioning from Team Sync to SCIM, consider the following important points:

{{< admonition type="warning" >}}
Team names must be unique in Grafana. You cannot have two teams with the same name, even if one is managed by Team Sync and the other by SCIM.
{{< /admonition >}}

#### Existing teams and permissions

When migrating from Team Sync to SCIM:

- Existing teams must be deleted before SCIM can create new teams with the same names
- Team memberships will be managed by SCIM after migration
- Team permissions must be manually reassigned through Terraform, Grafana API or Grafana UI
- Document current team permissions before migration
- Plan the migration to minimize service disruption

#### Migration process

While SCIM manages team memberships automatically, team permissions must be managed separately through your existing provisioning methods.

1. Document current team structure:

   - List all existing teams
   - Record current team memberships
   - Document team permissions and access levels

2. Prepare for migration:

   - Disable Team Sync
   - Delete existing teams (after documenting their configuration)
   - Enable SCIM

3. Restore team permissions:
   - Use your preferred method (Terraform, API, or UI) to reassign permissions
   - Verify access levels are correctly restored
   - Test team access for key users
