import React from 'react';
import { Button, Result } from '@arco-design/web-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const PageNotFound: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Result
      className="page-result"
      status="404"
      title="404"
      subTitle={t('common.notFound')}
      extra={
        <Button type="primary" onClick={() => navigate('/')}>
          {t('common.backHome')}
        </Button>
      }
    />
  );
};

export default PageNotFound;
