import React from 'react';
import { SideRailItem, SideRailNote, SideRailPanel, SideRailStack } from './SideRail';
import type { RailSummaryItem, RailSummaryTone } from './RailSummary';

interface StandardRailSummaryProps {
  title?: React.ReactNode;
  items: RailSummaryItem[];
  className?: string;
  panelClassName?: string;
}

interface StandardRailNotePanelProps {
  title?: React.ReactNode;
  noteTitle?: React.ReactNode;
  noteDescription?: React.ReactNode;
  noteTone?: RailSummaryTone;
  className?: string;
  panelClassName?: string;
}

export const StandardRailSummary: React.FC<StandardRailSummaryProps> = ({
  title,
  items,
  className,
  panelClassName,
}) => (
  <SideRailPanel title={title} className={panelClassName}>
    <SideRailStack className={className}>
      {items.map((item, index) => (
        <SideRailItem
          key={index}
          label={item.label}
          value={item.value}
          description={item.description}
          tone={item.tone}
          className={item.className}
        />
      ))}
    </SideRailStack>
  </SideRailPanel>
);

export const StandardRailNotePanel: React.FC<StandardRailNotePanelProps> = ({
  title,
  noteTitle,
  noteDescription,
  noteTone = 'neutral',
  className,
  panelClassName,
}) => (
  <SideRailPanel title={title} className={panelClassName}>
    <SideRailNote
      title={noteTitle}
      description={noteDescription}
      tone={noteTone}
      className={className}
    />
  </SideRailPanel>
);
