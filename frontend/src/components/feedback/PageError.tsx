import React from 'react';
import { Button, Result } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';

interface PageErrorProps {
  description?: React.ReactNode;
  onRetry?: () => void;
}

const PageError: React.FC<PageErrorProps> = ({ description, onRetry }) => {
  const { t } = useTranslation();

  return (
    <Result
      className="page-result"
      status="error"
      title={t('common.loadFailed')}
      subTitle={description || t('common.loadFailedDesc')}
      extra={onRetry ? <Button onClick={onRetry}>{t('common.retry')}</Button> : null}
    />
  );
};

export default PageError;
