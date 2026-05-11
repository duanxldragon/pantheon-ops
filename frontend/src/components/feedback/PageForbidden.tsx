import React from 'react';
import { Result } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';

const PageForbidden: React.FC = () => {
  const { t } = useTranslation();

  return (
    <Result className="page-result" status="403" title="403" subTitle={t('common.forbidden')} />
  );
};

export default PageForbidden;
