import { locationService } from '@grafana/runtime';
import { Badge, BadgeColor, IconName } from '@grafana/ui';

import { PROVISIONING_URL } from './constants';

interface StatusBadgeProps {
  state?: string;
  name: string;
}

export function StatusBadge({ state, name }: StatusBadgeProps) {
  if (state == null) {
    return null;
  }

  let tooltip: string | undefined = undefined;
  let color: BadgeColor = 'purple';
  let text = 'Unknown';
  let icon: IconName = 'exclamation-triangle';
  switch (state) {
    case 'success':
      icon = 'check';
      text = 'In sync';
      color = 'green';
      break;
    case null:
    case undefined:
    case '':
      color = 'orange';
      text = 'Pending';
      icon = 'spinner';
      tooltip = 'Waiting for health check to run';
      break;
    case 'working':
    case 'pending':
      color = 'orange';
      text = 'Syncing';
      icon = 'spinner';
      break;
    case 'error':
      color = 'red';
      text = 'Error';
      icon = 'exclamation-triangle';
      break;
    default:
      break;
  }
  return (
    <Badge
      color={color}
      icon={icon}
      text={text}
      style={{ cursor: 'pointer' }}
      tooltip={tooltip}
      onClick={() => {
        locationService.push(`${PROVISIONING_URL}/${name}/?tab=health`);
      }}
    />
  );
}
