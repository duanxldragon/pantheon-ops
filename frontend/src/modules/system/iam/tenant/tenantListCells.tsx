import React from 'react';
import { Tag, Typography } from '@arco-design/web-react';
import { formatDateTime } from '../../../../core/format/dateTime';
import QuotaRatio from './QuotaRatio';

/**
 * The cell renderers both tenant-ops list pages share.
 *
 * Keeping them here is what makes the two pages provably agree on how a
 * suspended tenant, an unmanaged plan code or an over-limit quota looks; a
 * column re-styled in one page cannot silently disagree with the other.
 */

export const tenantStatusTone: Record<string, string> = {
  active: 'green',
  suspended: 'orange',
  archived: 'gray',
  deleted: 'red',
};

/**
 * A plan code carried by tenants that resolves to no definition is not the free
 * tier: it is unmanaged data. Reporting it as free would hide the fact that the
 * ceiling shown next to it is only a fallback, so it gets its own tone.
 */
export const planTone: Record<string, string> = {
  free: 'gray',
  basic: 'blue',
  professional: 'arcoblue',
  enterprise: 'orange',
};

type Translate = (key: string, options?: Record<string, unknown>) => string;

/** A status/role tag: the tone comes from the caller's table, the label from i18n. */
export function renderStatusTag(
  t: Translate,
  tone: Record<string, string>,
  keyPrefix: string,
): (value: string) => React.ReactElement {
  return (value) => (
    <Tag color={tone[value] ?? 'gray'}>{t(`${keyPrefix}.${value}`, { defaultValue: value })}</Tag>
  );
}

/** The plan tag, shared because "no name, no code" has to read as unmanaged here too. */
export function renderPlanTag(
  t: Translate,
): (value: string, row: { planName?: string }) => React.ReactElement {
  return (value, row) => (
    <Tag color={planTone[value] ?? 'gray'}>
      {row.planName || value || t('system.tenant.overview.plan.unmanaged')}
    </Tag>
  );
}

export function renderDateTime(): (value: unknown) => React.ReactElement {
  return (value) => (
    <Typography.Text className="system-list__datetime-text">
      {formatDateTime(value as string, { withSeconds: true })}
    </Typography.Text>
  );
}

/** Usage against the row's own ceiling; `limitOf` picks which ceiling applies. */
export function renderQuota<T>(
  unlimitedText: string,
  limitOf: (row: T) => number,
): (value: number, row: T) => React.ReactElement {
  return (value, row) => (
    <QuotaRatio usage={value} limit={limitOf(row)} unlimitedText={unlimitedText} />
  );
}
