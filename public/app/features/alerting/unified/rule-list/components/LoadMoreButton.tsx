import { Trans, useTranslate } from '@grafana/i18n';
import { Button } from '@grafana/ui';

interface LoadMoreButtonProps {
  onClick: () => void;
  loading?: boolean;
}

export function LoadMoreButton({ onClick, loading = false }: LoadMoreButtonProps) {
  const { t } = useTranslate();
  const label = t('alerting.rule-list.pagination.next-page', 'Show more…');

  return (
    <Button aria-label={label} fill="text" size="sm" variant="secondary" onClick={onClick} disabled={loading}>
      {loading ? <Trans i18nKey="alerting.rule-list.loading-more-groups">Loading more groups…</Trans> : label}
    </Button>
  );
}
