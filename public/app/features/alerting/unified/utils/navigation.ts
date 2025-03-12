import { RuleGroupIdentifierV2 } from 'app/types/unified-alerting';

import { createReturnTo } from '../hooks/useReturnTo';

import { createRelativeUrl } from './url';

export const createListFilterLink = (values: Array<[string, string]>) => {
  const params = new URLSearchParams([['search', values.map(([key, value]) => `${key}:"${value}"`).join(' ')]]);
  return createRelativeUrl(`/alerting/list`, params);
};

export const alertListPageLink = createRelativeUrl(`/alerting/list`);

export const groups = {
  detailsPageLink: (dsUid: string, namespaceId: string, groupName: string, options?: { includeReturnTo: boolean }) => {
    const params: Record<string, string> = options?.includeReturnTo ? { returnTo: createReturnTo() } : {};

    return createRelativeUrl(
      `/alerting/${dsUid}/namespaces/${encodeURIComponent(namespaceId)}/groups/${encodeURIComponent(groupName)}/view`,
      params
    );
  },
  detailsPageLinkFromGroupIdentifier: (groupIdentifier: RuleGroupIdentifierV2) => {
    const { groupOrigin, namespace, groupName } = groupIdentifier;
    const isGrafanaOrigin = groupOrigin === 'grafana';

    return isGrafanaOrigin
      ? groups.detailsPageLink('grafana', namespace.uid, groupName)
      : groups.detailsPageLink(groupIdentifier.rulesSource.uid, namespace.name, groupName);
  },
  editPageLink: (dsUid: string, namespaceId: string, groupName: string, options?: { includeReturnTo: boolean }) => {
    const params: Record<string, string> = options?.includeReturnTo ? { returnTo: createReturnTo() } : {};
    return createRelativeUrl(
      `/alerting/${dsUid}/namespaces/${encodeURIComponent(namespaceId)}/groups/${encodeURIComponent(groupName)}/edit`,
      params
    );
  },
};
