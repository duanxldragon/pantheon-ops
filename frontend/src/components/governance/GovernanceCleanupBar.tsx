import dayjs, { type Dayjs } from 'dayjs';
import React, { useState } from 'react';
import { Alert, Button, DatePicker, Radio, Select, Typography } from '@arco-design/web-react';
import { IconDelete } from '@arco-design/web-react/icon';
import AppModal from '../patterns/modals/AppModal';

const { RangePicker } = DatePicker;
const CLEANUP_RANGE_FORMAT = 'YYYY-MM-DD HH:mm';

export type GovernanceCleanupMode = 'retention' | 'range';

export interface GovernanceCleanupPayload {
  mode: GovernanceCleanupMode;
  retentionDays?: number;
  /** RFC3339 with local offset, e.g. 2026-07-18T09:00:00+08:00 */
  startedAt?: string;
  endedAt?: string;
}

interface GovernanceCleanupBarProps {
  showCleanup?: boolean;
  retentionDays: number;
  retentionOptions: number[];
  onRetentionChange: (value: number) => void;
  retentionLabel: (value: number) => string;
  /** Shown inside the dialog as the irreversible-action warning. */
  confirmTitle: string;
  actionLabel: string;
  confirmActionLabel?: string;
  cleanupModeLabel?: string;
  cleanupModeOptions?: Array<{ label: string; value: GovernanceCleanupMode }>;
  rangeStartLabel?: string;
  rangeEndLabel?: string;
  rangeRequiredMessage?: string;
  onConfirm: (payload: GovernanceCleanupPayload) => Promise<void> | void;
  hint?: string;
  extraActions?: React.ReactNode;
  trailing?: React.ReactNode;
}

function toRfc3339(value: Dayjs) {
  return value.second(0).millisecond(0).format('YYYY-MM-DDTHH:mm:ssZ');
}

const GovernanceCleanupBar: React.FC<GovernanceCleanupBarProps> = ({
  retentionDays,
  retentionOptions,
  onRetentionChange,
  retentionLabel,
  confirmTitle,
  actionLabel,
  confirmActionLabel,
  cleanupModeLabel,
  cleanupModeOptions,
  rangeStartLabel,
  rangeEndLabel,
  rangeRequiredMessage,
  onConfirm,
  hint,
  extraActions,
  trailing,
  showCleanup = true,
}) => {
  const [dialogVisible, setDialogVisible] = useState(false);
  const [mode, setMode] = useState<GovernanceCleanupMode>('retention');
  const [rangeValue, setRangeValue] = useState<Dayjs[]>([]);
  const [rangeError, setRangeError] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const rangeEnabled = Boolean(cleanupModeOptions?.some((item) => item.value === 'range'));

  const openDialog = () => {
    setMode('retention');
    setRangeValue([]);
    setRangeError(false);
    setDialogVisible(true);
  };

  const closeDialog = () => {
    if (!submitting) {
      setDialogVisible(false);
    }
  };

  const handleOk = async () => {
    let payload: GovernanceCleanupPayload;
    if (mode === 'range') {
      const [start, end] = rangeValue;
      if (!start || !end) {
        setRangeError(true);
        return;
      }
      payload = { mode, startedAt: toRfc3339(start), endedAt: toRfc3339(end) };
    } else {
      payload = { mode, retentionDays };
    }
    setSubmitting(true);
    try {
      await onConfirm(payload);
      setDialogVisible(false);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="table-batch-action-bar table-batch-action-bar--governance">
      <div className="table-batch-action-bar__main">
        {showCleanup ? (
          <div className="table-batch-action-bar__meta">
            <Button type="outline" status="danger" icon={<IconDelete />} onClick={openDialog}>
              {actionLabel}
            </Button>
            {trailing}
          </div>
        ) : (
          trailing
        )}
        {extraActions ? (
          <div className="table-batch-action-bar__actions">{extraActions}</div>
        ) : null}
      </div>
      {hint ? (
        <Typography.Text type="secondary" className="table-batch-action-bar__hint">
          {hint}
        </Typography.Text>
      ) : null}
      <AppModal
        title={actionLabel}
        visible={dialogVisible}
        size="sm"
        onCancel={closeDialog}
        onOk={() => {
          void handleOk();
        }}
        okText={confirmActionLabel || actionLabel}
        confirmLoading={submitting}
        okButtonProps={{ status: 'danger' }}
      >
        <div className="governance-cleanup-dialog">
          {rangeEnabled && cleanupModeOptions ? (
            <div className="governance-cleanup-dialog__field">
              {cleanupModeLabel ? (
                <Typography.Text type="secondary" className="governance-cleanup-dialog__label">
                  {cleanupModeLabel}
                </Typography.Text>
              ) : null}
              <Radio.Group
                type="button"
                value={mode}
                onChange={(value) => {
                  setMode(value as GovernanceCleanupMode);
                  setRangeError(false);
                }}
                options={cleanupModeOptions}
              />
            </div>
          ) : null}
          {mode === 'range' && rangeEnabled ? (
            <div className="governance-cleanup-dialog__field">
              <RangePicker
                className="governance-cleanup-bar__range-picker"
                style={{ width: '100%' }}
                allowClear
                dayStartOfWeek={1}
                showTime={{ format: 'HH:mm' }}
                format={CLEANUP_RANGE_FORMAT}
                placeholder={[rangeStartLabel || '', rangeEndLabel || '']}
                value={rangeValue}
                disabledDate={(current) => Boolean(current?.isAfter(dayjs(), 'day'))}
                // Inside AppModal the popup default-mounts into the dialog,
                // where `.app-dialog { overflow: hidden }` clips the calendar
                // (it opens above the dialog box). Mount it on body instead,
                // above the wrapper AppModal lifts to z-index 2500.
                triggerProps={{
                  getPopupContainer: () => document.body,
                  style: { zIndex: 2600 },
                }}
                onChange={(_, dates) => {
                  setRangeValue((dates || []).filter(Boolean) as Dayjs[]);
                  setRangeError(false);
                }}
              />
              {rangeError && rangeRequiredMessage ? (
                <Typography.Text type="error" className="governance-cleanup-dialog__error">
                  {rangeRequiredMessage}
                </Typography.Text>
              ) : null}
            </div>
          ) : (
            <div className="governance-cleanup-dialog__field">
              <Select
                value={retentionDays}
                onChange={(value) => onRetentionChange(Number(value))}
                options={retentionOptions.map((option) => ({
                  label: retentionLabel(option),
                  value: option,
                }))}
              />
            </div>
          )}
          <Alert type="warning" content={confirmTitle} />
        </div>
      </AppModal>
    </div>
  );
};

export default GovernanceCleanupBar;
