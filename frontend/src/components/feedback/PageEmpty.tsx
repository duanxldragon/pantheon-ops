import React from 'react';
import { Empty } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';

interface PageEmptyProps {
  description?: React.ReactNode;
}

const PageEmpty: React.FC<PageEmptyProps> = ({ description }) => {
  const { t } = useTranslation();

  return (
    <div className="page-empty">
      <Empty className="page-empty__inner" description={description || t('common.noDataDesc')} />
    </div>
  );
};

export default PageEmpty;
