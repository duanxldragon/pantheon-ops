import React from 'react';
import { Button } from '@arco-design/web-react';
import { IconStorage } from '@arco-design/web-react/icon';
import AppDrawer from './AppDrawer';
import { SideRailItem, SideRailNote, SideRailPanel, SideRailStack } from './SideRail';
import type { RailSummaryItem, RailSummaryTone } from './RailSummary';

interface GovernanceRailToggleButtonProps {
  expanded: boolean;
  onToggle: () => void;
  children: React.ReactNode;
  disabled?: boolean;
}

interface GovernanceRailCloseButtonProps {
  onClose: () => void;
  children: React.ReactNode;
}

interface GovernanceRailPanelProps {
  title: React.ReactNode;
  onClose: () => void;
  closeText: React.ReactNode;
  children: React.ReactNode;
  noteTitle?: React.ReactNode;
  noteDescription?: React.ReactNode;
  noteTone?: RailSummaryTone;
}

interface GovernanceRailSummaryProps {
  items: RailSummaryItem[];
  className?: string;
}

interface GovernanceInsightDrawerProps {
  visible: boolean;
  title: React.ReactNode;
  onClose: () => void;
  children: React.ReactNode;
  noteTitle?: React.ReactNode;
  noteDescription?: React.ReactNode;
  noteTone?: RailSummaryTone;
  footer?: React.ReactNode;
}

export const GovernanceRailToggleButton: React.FC<GovernanceRailToggleButtonProps> = ({
  expanded,
  onToggle,
  children,
  disabled = false,
}) => (
  <Button
    icon={<IconStorage />}
    type={expanded ? 'primary' : 'secondary'}
    onClick={onToggle}
    disabled={disabled}
  >
    {children}
  </Button>
);

export const GovernanceRailCloseButton: React.FC<GovernanceRailCloseButtonProps> = ({
  onClose,
  children,
}) => (
  <Button type="text" size="mini" onClick={onClose}>
    {children}
  </Button>
);

export const GovernanceRailPanel: React.FC<GovernanceRailPanelProps> = ({
  title,
  onClose,
  closeText,
  children,
  noteTitle,
  noteDescription,
  noteTone = 'neutral',
}) => (
  <>
    <SideRailPanel
      title={title}
      extra={<GovernanceRailCloseButton onClose={onClose}>{closeText}</GovernanceRailCloseButton>}
    >
      {children}
    </SideRailPanel>
    {noteTitle || noteDescription ? (
      <SideRailPanel>
        <SideRailNote title={noteTitle} description={noteDescription} tone={noteTone} />
      </SideRailPanel>
    ) : null}
  </>
);

export const GovernanceRailSummary: React.FC<GovernanceRailSummaryProps> = ({
  items,
  className,
}) => (
  <SideRailStack className={className}>
    {items.map((item, index) => (
      <SideRailItem
        key={index}
        label={item.label}
        value={item.value}
        tone={item.tone}
        className={item.className}
      />
    ))}
  </SideRailStack>
);

export const GovernanceInsightDrawer: React.FC<GovernanceInsightDrawerProps> = ({
  visible,
  title,
  onClose,
  children,
  noteTitle,
  noteDescription,
  noteTone = 'neutral',
  footer,
}) => (
  <AppDrawer
    title={title}
    visible={visible}
    onCancel={onClose}
    size="sm"
    className="governance-insight-drawer"
    footer={footer ?? null}
  >
    <div className="governance-insight-drawer__body">
      <SideRailPanel className="governance-insight-drawer__summary">{children}</SideRailPanel>
      {noteTitle || noteDescription ? (
        <SideRailPanel className="governance-insight-drawer__note-panel">
          <SideRailNote title={noteTitle} description={noteDescription} tone={noteTone} />
        </SideRailPanel>
      ) : null}
    </div>
  </AppDrawer>
);
