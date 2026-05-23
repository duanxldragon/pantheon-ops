import React from 'react';
import { Typography } from '@arco-design/web-react';

export interface GovernanceSummaryMetric {
  key: string;
  label: React.ReactNode;
  value: React.ReactNode;
  description?: React.ReactNode;
  className?: string;
  role?: React.AriaRole;
  tabIndex?: number;
  onClick?: React.MouseEventHandler<HTMLDivElement>;
  onKeyDown?: React.KeyboardEventHandler<HTMLDivElement>;
}

export interface GovernanceSummaryBarProps {
  eyebrow?: React.ReactNode;
  title?: React.ReactNode;
  description?: React.ReactNode;
  metrics: GovernanceSummaryMetric[];
  action?: React.ReactNode;
  className?: string;
}

const GovernanceSummaryBar: React.FC<GovernanceSummaryBarProps> = ({
  eyebrow,
  title,
  description,
  metrics,
  action,
  className,
}) => {
  const summaryLine = title || description;

  return (
    <section
      className={['page-panel', 'governance-summary-bar', className].filter(Boolean).join(' ')}
    >
      <div className="governance-summary-bar__copy">
        {eyebrow ? <span className="governance-summary-bar__eyebrow">{eyebrow}</span> : null}
        {summaryLine ? (
          <Typography.Text className="governance-summary-bar__title">
            {summaryLine}
          </Typography.Text>
        ) : null}
      </div>
      <div className="governance-summary-bar__metrics">
        {metrics.map((item) => (
          <div
            key={item.key}
            className={['governance-summary-bar__metric', item.className].filter(Boolean).join(' ')}
            role={item.role}
            tabIndex={item.tabIndex}
            onClick={item.onClick}
            onKeyDown={item.onKeyDown}
          >
            <span className="governance-summary-bar__metric-label">{item.label}</span>
            <strong className="governance-summary-bar__metric-value">{item.value}</strong>
          </div>
        ))}
      </div>
      {action ? <div className="governance-summary-bar__action">{action}</div> : null}
    </section>
  );
};

export default GovernanceSummaryBar;
