import React from 'react';
import { Button, Popconfirm, Select, Typography } from '@arco-design/web-react';
import { IconDelete } from '@arco-design/web-react/icon';

interface GovernanceCleanupBarProps {
  showCleanup?: boolean;
  retentionDays: number;
  retentionOptions: number[];
  onRetentionChange: (value: number) => void;
  retentionLabel: (value: number) => string;
  confirmTitle: string;
  actionLabel: string;
  onConfirm: () => void;
  hint?: string;
  extraActions?: React.ReactNode;
  trailing?: React.ReactNode;
}

const GovernanceCleanupBar: React.FC<GovernanceCleanupBarProps> = ({
  retentionDays,
  retentionOptions,
  onRetentionChange,
  retentionLabel,
  confirmTitle,
  actionLabel,
  onConfirm,
  hint,
  extraActions,
  trailing,
  showCleanup = true,
}) => (
  <div className="table-batch-action-bar table-batch-action-bar--governance">
    <div className="table-batch-action-bar__main">
      {showCleanup ? (
        <div className="table-batch-action-bar__meta">
          <Select
            className="table-batch-action-bar__select"
            value={retentionDays}
            onChange={(value) => onRetentionChange(Number(value))}
            options={retentionOptions.map((option) => ({
              label: retentionLabel(option),
              value: option,
            }))}
          />
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
