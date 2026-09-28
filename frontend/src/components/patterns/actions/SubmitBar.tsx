import React from 'react';
import { Button, Dropdown, Menu, Space } from '@arco-design/web-react';
import { IconMore } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';

export type SubmitBarStatus = 'idle' | 'dirty' | 'submitting' | 'success' | 'error' | 'conflict';

export interface SubmitBarSecondaryAction {
  key: string;
  label: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
}

export interface SubmitBarProps {
  onCancel?: () => void;
  onSubmit?: () => void;
  loading?: boolean;
  submitDisabled?: boolean;
  submitText?: React.ReactNode;
  cancelText?: React.ReactNode;
  sticky?: boolean;
  status?: SubmitBarStatus;
  statusText?: React.ReactNode;
  errorSummary?: React.ReactNode;
  secondaryActions?: SubmitBarSecondaryAction[];
  moreActionsLabel?: React.ReactNode;
  primaryActionAriaLabel?: string;
  errorSummaryId?: string;
}

const SubmitBar: React.FC<SubmitBarProps> = ({
  onCancel,
  onSubmit,
  loading,
  submitDisabled,
  submitText,
  cancelText,
  sticky = false,
  status = loading ? 'submitting' : 'idle',
  statusText,
  errorSummary,
  secondaryActions = [],
  moreActionsLabel,
  primaryActionAriaLabel,
  errorSummaryId = 'submit-bar-error-summary',
}) => {
  const { t } = useTranslation();
  const hasStatus = Boolean(statusText) || status !== 'idle';
  const effectiveSubmitDisabled = submitDisabled || loading || status === 'submitting';
  const overflowMenu =
    secondaryActions.length > 0 ? (
      <Menu
        onClickMenuItem={(key) => {
          const action = secondaryActions.find((item) => item.key === key);
          if (!action?.disabled) {
            action?.onClick();
          }
        }}
      >
        {secondaryActions.map((action) => (
          <Menu.Item key={action.key} disabled={action.disabled}>
            {action.label}
          </Menu.Item>
        ))}
      </Menu>
    ) : null;

  return (
    <div
      className="submit-bar"
      data-status={status}
      data-sticky={sticky || undefined}
    >
      <div className="submit-bar__state" aria-live="polite">
        {hasStatus ? (
          <span className="submit-bar__status">
            {statusText || t(`common.submitBar.status.${status}`, { defaultValue: status })}
          </span>
        ) : null}
        {errorSummary ? (
          <div id={errorSummaryId} className="submit-bar__error" role="alert" tabIndex={-1}>
            {errorSummary}
          </div>
        ) : null}
      </div>
      <Space size={10} className="submit-bar__actions">
        {onCancel ? <Button onClick={onCancel}>{cancelText || t('common.cancel')}</Button> : null}
        {overflowMenu ? (
          <Dropdown trigger="click" position="br" droplist={overflowMenu}>
            <Button
              className="submit-bar__overflow"
              icon={<IconMore />}
              aria-label={String(moreActionsLabel || t('common.moreActions'))}
            />
          </Dropdown>
        ) : null}
        <Button
          type="primary"
          htmlType="submit"
          loading={loading}
          disabled={effectiveSubmitDisabled}
          onClick={onSubmit}
          aria-label={primaryActionAriaLabel}
          aria-describedby={errorSummary ? errorSummaryId : undefined}
        >
          {submitText || t('common.save')}
        </Button>
      </Space>
    </div>
  );
};

export default SubmitBar;
