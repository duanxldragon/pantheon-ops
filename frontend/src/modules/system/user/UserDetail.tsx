import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button } from '@arco-design/web-react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PageContainer, PageEmpty, PageError, PageLoading } from '../../../components';
import { getUserDetail, type UserDetail as UserDetailData } from './api';
import UserDetailContent from './UserDetailContent';

const UserDetail: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [detail, setDetail] = useState<UserDetailData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  const userId = useMemo(() => Number(id), [id]);
  const invalidUserId = !Number.isInteger(userId) || userId <= 0;

  const loadDetail = useCallback(async () => {
    if (invalidUserId) {
      setDetail(null);
      setError(false);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(false);
    try {
      const result = await getUserDetail(userId);
      setDetail(result);
    } catch {
      setError(true);
      setDetail(null);
    } finally {
      setLoading(false);
    }
  }, [invalidUserId, userId]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadDetail();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadDetail]);

  if (loading) {
    return <PageLoading />;
  }

  if (invalidUserId) {
    return <PageEmpty description={t('system.user.detailInvalid')} />;
  }

  if (error) {
    return (
      <PageError
        onRetry={() => {
          void loadDetail();
        }}
      />
    );
  }

  if (!detail) {
    return <PageEmpty description={t('common.noData')} />;
  }

  return (
    <PageContainer>
      <div className="system-list__work-actions">
        <Button onClick={() => navigate('/system/user')}>{t('common.back')}</Button>
      </div>
      <UserDetailContent detail={detail} />
    </PageContainer>
  );
};

export default UserDetail;
