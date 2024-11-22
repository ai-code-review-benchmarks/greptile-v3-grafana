import { config, getBackendSrv } from '@grafana/runtime';
import { DashboardDataDTO, DashboardDTO } from 'app/types';

import { Resource } from '../apiserver/types';

/**
 * Load a dashboard from repository
 */
export async function loadDashboardFromProvisioning(repo: string, path: string): Promise<DashboardDTO> {
  const params = new URLSearchParams(window.location.search);
  const ref = params.get('ref') ?? undefined; // commit hash or branch

  const url = `apis/provisioning.grafana.app/v0alpha1/namespaces/${config.namespace}/repositories/${repo}/files/${path}`;
  return getBackendSrv()
    .get(url, ref ? { ref } : undefined)
    .then((v) => {
      // Load the results from dryRun
      const dryRun = v.resource.dryRun as Resource<DashboardDataDTO, 'Dashboard'>;
      if (!dryRun) {
        return Promise.reject("failed to read provisioned dashboard")
      }

      if (!dryRun.apiVersion.startsWith("dashboard.grafana.app")) {
        return Promise.reject("unexpected resource type: "+dryRun.apiVersion)
      }

      return {
        meta: {
          canStar: false,
          isSnapshot: false,
          canShare: false,

          // Should come from the repo settings
          canDelete: true,
          canSave: true,
          canEdit: true,

          // Includes additional k8s metadata
          k8s: dryRun.metadata,

          // lookup info
          provisioning: {
            file: url,
            ref: ref,
            repo: repo,
          }
        },
        dashboard: dryRun.spec
      }
    });
}
