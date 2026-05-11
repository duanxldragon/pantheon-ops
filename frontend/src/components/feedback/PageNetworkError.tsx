import React from 'react';
import { Button, Result } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';

interface PageNetworkErrorProps {
  description?: React.ReactNode;
  onRetry?: () => void;
  timeout?: boolean;
}

const PageNetworkError: React.FC<PageNetworkErrorProps> = ({
  description,
  onRetry,
  timeout = false,
}) => {
  const { t } = useTranslation();

  return (
    <Result
      className="page-result"
      status="warning"
      title={t(timeout ? 'common.networkTimeoutTitle' : 'common.networkErrorTitle')}
      subTitle={description || t(timeout ? 'common.networkTimeout' : 'common.networkError')}
      extra={onRetry ? <Button onClick={onRetry}>{t('common.retry')}</Button> : null}
    />
  );
};

export default PageNetworkError;
