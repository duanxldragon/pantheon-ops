import React from 'react';
import GovernanceSummaryBar, { type GovernanceSummaryMetric } from './GovernanceSummaryBar';
import { GovernanceRailToggleButton } from './GovernanceRail';

export interface GovernanceListSummaryProps {
  eyebrow: React.ReactNode;
  title: React.ReactNode;
  description: React.ReactNode;
  metrics: GovernanceSummaryMetric[];
  summaryTitle: React.ReactNode;
  rail: { expanded: boolean; toggle: () => void };
  railToggle?: React.ReactNode;
  className?: string;
}

const GovernanceListSummary: React.FC<GovernanceListSummaryProps> = ({
  eyebrow,
  title,
  description,
  metrics,
  summaryTitle,
  rail,
  railToggle,
  className,
}) => (
  <GovernanceSummaryBar
    className={className}
    eyebrow={eyebrow}
    title={title}
    description={description}
    metrics={metrics}
    action={
      railToggle ?? (
        <GovernanceRailToggleButton expanded={rail.expanded} onToggle={rail.toggle}>
          {summaryTitle}
        </GovernanceRailToggleButton>
      )
    }
  />
);

export default GovernanceListSummary;
