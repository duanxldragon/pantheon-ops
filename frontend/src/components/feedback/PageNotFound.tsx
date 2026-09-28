import React from 'react';
import { Result } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';

const PageNotFound: React.FC = () => {
  const { t } = useTranslation();

  return (
    <Result className="page-result" status="404" title="404" subTitle={t('common.notFound')} />
  );
};

export default PageNotFound;
