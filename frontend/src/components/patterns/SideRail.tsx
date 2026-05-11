import React from 'react';
import { Card } from '@arco-design/web-react';

type SideRailTone = 'neutral' | 'warning' | 'danger';

interface SideRailPanelProps {
  title?: React.ReactNode;
  extra?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}

interface SideRailStackProps {
  className?: string;
  children: React.ReactNode;
}

interface SideRailItemProps {
  label?: React.ReactNode;
  value?: React.ReactNode;
  description?: React.ReactNode;
  tone?: SideRailTone;
  className?: string;
}

interface SideRailNoteProps {
  title?: React.ReactNode;
  description?: React.ReactNode;
  tone?: SideRailTone;
  className?: string;
}

export const SideRailPanel: React.FC<SideRailPanelProps> = ({
  title,
  extra,
  className,
  children,
}) => (
  <Card
    className={className ? `page-panel side-rail-panel ${className}` : 'page-panel side-rail-panel'}
  >
    {title || extra ? (
      <div className="side-rail-panel__header">
        {title ? <span className="side-rail-panel__title">{title}</span> : <span />}
        {extra ? <div className="side-rail-panel__extra">{extra}</div> : null}
      </div>
    ) : null}
    {children}
  </Card>
);

export const SideRailStack: React.FC<SideRailStackProps> = ({ className, children }) => (
  <div className={className ? `side-rail-stack ${className}` : 'side-rail-stack'}>{children}</div>
);

export const SideRailItem: React.FC<SideRailItemProps> = ({
  label,
  value,
  description,
  tone = 'neutral',
  className,
}) => (
  <div
    className={
      className
        ? `side-rail-item side-rail-item--${tone} ${className}`
        : `side-rail-item side-rail-item--${tone}`
    }
  >
    {label ? <span className="side-rail-item__label">{label}</span> : null}
    {value ? <span className="side-rail-item__value">{value}</span> : null}
    {description ? <span className="side-rail-item__desc">{description}</span> : null}
  </div>
);

export const SideRailNote: React.FC<SideRailNoteProps> = ({
  title,
  description,
  tone = 'neutral',
  className,
}) => (
  <div
    className={
      className
        ? `side-rail-note side-rail-note--${tone} ${className}`
        : `side-rail-note side-rail-note--${tone}`
    }
  >
    {title ? <span className="side-rail-note__title">{title}</span> : null}
    {description ? <span className="side-rail-note__desc">{description}</span> : null}
  </div>
);
