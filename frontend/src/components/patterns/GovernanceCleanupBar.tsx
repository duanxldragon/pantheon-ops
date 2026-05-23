import React from 'react';
import { Button, Input, Popconfirm, Select, Space, Typography } from '@arco-design/web-react';
import { IconDelete } from '@arco-design/web-react/icon';

export type GovernanceCleanupMode = 'retention' | 'range';

interface GovernanceCleanupBarProps {
  showCleanup?: boolean;
  retentionDays: number;
  retentionOptions: number[];
  onRetentionChange: (value: number) => void;
  retentionLabel: (value: number) => string;
  confirmTitle: string;
  actionLabel: string;
  onConfirm: () => void;
  cleanupMode?: GovernanceCleanupMode;
  onCleanupModeChange?: (value: GovernanceCleanupMode) => void;
  cleanupModeLabel?: string;
  cleanupModeOptions?: Array<{ label: string; value: GovernanceCleanupMode }>;
  rangeStart?: string;
  rangeEnd?: string;
  onRangeStartChange?: (value: string) => void;
  onRangeEndChange?: (value: string) => void;
  rangeStartLabel?: string;
  rangeEndLabel?: string;
  hint?: string;
  extraActions?: React.ReactNode;
  trailing?: React.ReactNode;
}

function normalizeDateTimeLocalValue(value?: string) {
  const normalized = String(value || '').trim();
  if (!normalized) {
    return '';
  }
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(normalized)) {
    return normalized;
  }
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/.test(normalized)) {
    return normalized.replace(' ', 'T');
  }
  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime())) {
    return '';
  }
  const year = parsed.getFullYear();
  const month = `${parsed.getMonth() + 1}`.padStart(2, '0');
  const day = `${parsed.getDate()}`.padStart(2, '0');
  const hours = `${parsed.getHours()}`.padStart(2, '0');
  const minutes = `${parsed.getMinutes()}`.padStart(2, '0');
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

const GovernanceCleanupBar: React.FC<GovernanceCleanupBarProps> = ({
  retentionDays,
  retentionOptions,
  onRetentionChange,
  retentionLabel,
  confirmTitle,
  actionLabel,
  onConfirm,
  cleanupMode = 'retention',
  onCleanupModeChange,
  cleanupModeLabel,
  cleanupModeOptions,
  rangeStart,
  rangeEnd,
  onRangeStartChange,
  onRangeEndChange,
  rangeStartLabel,
  rangeEndLabel,
  hint,
  extraActions,
  trailing,
  showCleanup = true,
}) => (
  <div className="table-batch-action-bar table-batch-action-bar--governance">
    <div className="table-batch-action-bar__main">
      {showCleanup ? (
        <div className="table-batch-action-bar__meta">
          {cleanupModeOptions && onCleanupModeChange ? (
            <Select
              className="table-batch-action-bar__select"
              value={cleanupMode}
              placeholder={cleanupModeLabel}
              onChange={(value) => onCleanupModeChange(value as GovernanceCleanupMode)}
              options={cleanupModeOptions}
            />
          ) : null}
          {cleanupMode === 'range' ? (
            <Space size={8}>
              <Input
                className="table-batch-action-bar__datetime"
                type="datetime-local"
                value={normalizeDateTimeLocalValue(rangeStart)}
                placeholder={rangeStartLabel}
                onChange={onRangeStartChange}
              />
              <Input
                className="table-batch-action-bar__datetime"
                type="datetime-local"
                value={normalizeDateTimeLocalValue(rangeEnd)}
                placeholder={rangeEndLabel}
                onChange={onRangeEndChange}
              />
            </Space>
          ) : (
            <Select
              className="table-batch-action-bar__select"
              value={retentionDays}
              onChange={(value) => onRetentionChange(Number(value))}
              options={retentionOptions.map((option) => ({
                label: retentionLabel(option),
                value: option,
              }))}
            />
          )}
          <Popconfirm title={confirmTitle} onOk={onConfirm}>
            <Button type="primary" status="danger" icon={<IconDelete />}>
              {actionLabel}
            </Button>
          </Popconfirm>
          {trailing}
        </div>
      ) : (
        trailing
      )}
      {extraActions ? <div className="table-batch-action-bar__actions">{extraActions}</div> : null}
    </div>
    {hint ? (
      <Typography.Text type="secondary" className="table-batch-action-bar__hint">
        {hint}
      </Typography.Text>
    ) : null}
  </div>
);

export default GovernanceCleanupBar;
