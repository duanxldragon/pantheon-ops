import React from 'react';

export type RailSummaryTone = 'neutral' | 'warning' | 'danger';

export interface RailSummaryItem {
  label?: React.ReactNode;
  value?: React.ReactNode;
  description?: React.ReactNode;
  tone?: RailSummaryTone;
  className?: string;
}

export function withRailSummaryKeys(
  items: RailSummaryItem[],
): { key: string; item: RailSummaryItem }[] {
  const seen = new Map<string, number>();
  return items.map((item) => {
    const base =
      typeof item.label === 'string' || typeof item.label === 'number'
        ? `rail-item-${item.label}`
        : 'rail-item';
    const used = seen.get(base) ?? 0;
    seen.set(base, used + 1);
    return { key: used === 0 ? base : `${base}-${used}`, item };
  });
}
