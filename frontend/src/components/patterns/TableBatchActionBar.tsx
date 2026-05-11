import React from 'react';
import { Button, Typography } from '@arco-design/web-react';
import { message } from '../feedback/message';

interface TableBatchActionBarProps {
  selectedCount: number;
  selectedText: string;
  clearText: string;
  clearSuccessText?: string;
  onClear: () => void;
  actions?: React.ReactNode;
  hint?: React.ReactNode;
  className?: string;
}

const TableBatchActionBar: React.FC<TableBatchActionBarProps> = ({
  selectedCount,
  selectedText,
  clearText,
  clearSuccessText,
  onClear,
  actions,
  hint,
  className,
}) => {
  const handleClear = () => {
    if (selectedCount === 0) {
      return;
    }
    onClear();
    if (clearSuccessText) {
      message.success(clearSuccessText);
    }
  };

  return (
    <div className={className ? `table-batch-action-bar ${className}` : 'table-batch-action-bar'}>
      <div className="table-batch-action-bar__main">
        <div className="table-batch-action-bar__meta">
          <Typography.Text type="secondary" className="table-batch-action-bar__summary">
            {selectedText}
          </Typography.Text>
          <Button type="text" size="small" disabled={selectedCount === 0} onClick={handleClear}>
            {clearText}
          </Button>
        </div>
        {actions ? <div className="table-batch-action-bar__actions">{actions}</div> : null}
      </div>
      {hint ? (
        <Typography.Text type="secondary" className="table-batch-action-bar__hint">
          {hint}
        </Typography.Text>
      ) : null}
    </div>
  );
};

export default TableBatchActionBar;
