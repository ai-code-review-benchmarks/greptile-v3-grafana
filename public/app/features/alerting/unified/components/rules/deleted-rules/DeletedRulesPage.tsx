import { withPageErrorBoundary } from '../../../withPageErrorBoundary';
import { AlertingPageWrapper } from '../../AlertingPageWrapper';

import { DeletedRules } from './DeletedRules';

function DeletedrulesPage() {
  return (
    <AlertingPageWrapper navId="alerts/recently-deleted" isLoading={false}>
      <DeletedRules />
    </AlertingPageWrapper>
  );
}

export default withPageErrorBoundary(DeletedrulesPage);
