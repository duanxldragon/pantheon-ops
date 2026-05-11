import React from 'react';

export type RailSummaryTone = 'neutral' | 'warning' | 'danger';

export interface RailSummaryItem {
  label?: React.ReactNode;
  value?: React.ReactNode;
  description?: React.ReactNode;
  tone?: RailSummaryTone;
  className?: string;
}
