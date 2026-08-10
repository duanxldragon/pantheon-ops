import React from 'react';
import { Button, Space } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';

interface SubmitBarProps {
  onCancel?: () => void;
  onSubmit?: () => void;
  loading?: boolean;
  submitDisabled?: boolean;
  submitText?: React.ReactNode;
  cancelText?: React.ReactNode;
}

const SubmitBar: React.FC<SubmitBarProps> = ({
  onCancel,
  onSubmit,
  loading,
  submitDisabled,
  submitText,
  cancelText,
}) => {
  const { t } = useTranslation();

  return (
    <div className="submit-bar">
      <Space size={10}>
        <Button onClick={onCancel}>{cancelText || t('common.cancel')}</Button>
        <Button
          type="primary"
          htmlType="submit"
          loading={loading}
          disabled={submitDisabled}
          onClick={onSubmit}
        >
          {submitText || t('common.save')}
        </Button>
      </Space>
    </div>
  );
};

export default SubmitBar;
