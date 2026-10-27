import type { GovernanceCleanupMode } from './GovernanceCleanupBar';

export interface GovernanceCleanupLabels {
  retentionLabel: (option: number) => string;
  confirmTitle: string;
  confirmActionLabel: string;
  cleanupModeLabel: string;
  cleanupModeOptions: Array<{ label: string; value: GovernanceCleanupMode }>;
  rangeStartLabel: string;
  rangeEndLabel: string;
  rangeRequiredMessage: string;
}

export function buildGovernanceCleanupLabels(
  t: (key: string, options?: Record<string, unknown>) => string,
): GovernanceCleanupLabels {
  return {
    retentionLabel: (option) => t('common.keepRecentDays', { count: option }),
    confirmTitle: t('common.cleanupIrreversibleWarning'),
    confirmActionLabel: t('common.cleanup'),
    cleanupModeLabel: t('common.cleanupMode'),
    cleanupModeOptions: [
      { label: t('common.cleanupModeRetention'), value: 'retention' },
      { label: t('common.cleanupModeRange'), value: 'range' },
    ],
    rangeStartLabel: t('common.cleanupRangeStart'),
    rangeEndLabel: t('common.cleanupRangeEnd'),
    rangeRequiredMessage: t('common.cleanupRangeRequired'),
  };
}
