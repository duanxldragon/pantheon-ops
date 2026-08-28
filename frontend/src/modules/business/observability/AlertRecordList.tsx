import { useTranslation } from 'react-i18next';
import { Card } from '@arco-design/web-react';
import { PageContainer, PageEmpty } from '../../../components';
import '../../system/components/shared/list-page.css';

export default function AlertRecordList() {
  const { t } = useTranslation();
  return (
    <PageContainer>
      <Card className="page-panel system-list__table-card">
        <PageEmpty description={t('business.observability.planning')} />
      </Card>
    </PageContainer>
  );
}
