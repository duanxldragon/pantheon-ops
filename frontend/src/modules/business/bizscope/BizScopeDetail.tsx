import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { Button, Card, Descriptions, Space } from '@arco-design/web-react';
import { IconLeft } from '@arco-design/web-react/icon';
import { PageContainer, PageError, PageLoading } from '../../../components';
import PageHeader from '../../../components/patterns/PageHeader';
import { getBizScopeDetail, type BizScopeDetail as BizScopeDetailType } from './api';
import '../../system/list-page.css';

export default function BizScopeDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [detail, setDetail] = useState<BizScopeDetailType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const loadDetail = useCallback(async () => {
    if (!id) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await getBizScopeDetail(Number(id));
      setDetail(result);
    } catch (requestError) {
      setError(requestError);
      setDetail(null);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadDetail();
    });
  }, [loadDetail]);

  if (loading) {
    return (
      <PageContainer>
        <PageLoading />
      </PageContainer>
    );
  }

  if (error || !detail) {
    return (
      <PageContainer>
        <PageError description={t('common.loadFailedDesc')} onRetry={loadDetail} />
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <PageHeader
        title={detail.name}
        subtitle={t('operations.bizscope.detail')}
        extra={
          <Button icon={<IconLeft />} onClick={() => navigate('/operations/business-scope')}>
            {t('common.back')}
          </Button>
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel">
          <Descriptions
            column={2}
            data={[
              { label: t('business.bizscope.field.code'), value: detail.code },
              { label: t('business.bizscope.field.name'), value: detail.name },
              { label: t('business.bizscope.field.owner'), value: detail.owner || '-' },
              {
                label: t('business.bizscope.field.environment'),
                value: t(`business.bizscope.environment.${detail.environment}`),
              },
              {
                label: t('business.bizscope.field.status'),
                value: t(`business.bizscope.status.${detail.status}`),
              },
              { label: t('business.bizscope.field.remark'), value: detail.remark || '-' },
            ]}
          />
        </Card>
      </Space>
    </PageContainer>
  );
}
