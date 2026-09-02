// TableBatchActionBar - Shell visual contract compliance
import React from 'react';

export interface TableBatchActionBarProps {
  prefixActions?: React.ReactNode;
  selectedCount?: number;
  children?: React.ReactNode;
}

export const TableBatchActionBar: React.FC<TableBatchActionBarProps> = ({
  prefixActions,
  selectedCount = 0,
  children,
}) => {
  return (
    <div className="table-batch-action-bar">
      <div className="table-batch-action-bar__main">
        {prefixActions}
        <span className="table-batch-action-bar__select">
          已选择 {selectedCount} 项
        </span>
      </div>
      <div className="table-batch-action-bar__actions">{children}</div>
    </div>
  );
};

export {};
