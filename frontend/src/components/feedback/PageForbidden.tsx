import React from 'react';
import { Button, Result } from '@arco-design/web-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const PageForbidden: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Result
      className="page-result"
      status="403"
      title="403"
      subTitle={t('common.forbidden')}
      extra={
        <Button type="primary" onClick={() => navigate('/')}>
          {t('common.backHome')}
        </Button>
      }
    />
  );
};

export default PageForbidden;
