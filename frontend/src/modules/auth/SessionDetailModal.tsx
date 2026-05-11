import React from 'react';
import { Card, Descriptions, Space, Tag, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { AppModal, PageEmpty } from '../../components';
import { formatDateTime } from '../../core/format/dateTime';
import { formatClientSummary } from './clientInfo';

type SessionDetailRecord = {
  sessionId: string;
  username?: string;
  nickname?: string;
  isCurrent?: boolean;
  lastIp?: string;
  browser?: string;
  os?: string;
  device?: string;
  userAgent?: string;
  refreshExpiresAt?: string;
  lastRefreshAt?: string;
  lastActivityAt?: string;
  revokedAt?: string;
  createdAt?: string;
};

interface SessionDetailModalProps {
  visible: boolean;
  session?: SessionDetailRecord | null;
  onCancel: () => void;
}

const SessionDetailModal: React.FC<SessionDetailModalProps> = ({ visible, session, onCancel }) => {
  const { t } = useTranslation();

  return (
    <AppModal
      title={t('common.detail')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size="detail"
    >
      {!session ? (
        <PageEmpty description={t('auth.session.empty')} />
      ) : (
        <Space direction="vertical" size={16} className="auth-detail-stack">
          <div className="auth-detail-summary">
            <div className="auth-detail-summary__copy">
              <Typography.Text className="auth-detail-summary__title">
                {formatClientSummary(session)}
              </Typography.Text>
              <Typography.Text className="auth-detail-summary__desc">
                {session.username
                  ? `${session.username}${session.nickname ? ` / ${session.nickname}` : ''}`
                  : t('auth.security.currentSessionSummary')}
              </Typography.Text>
            </div>
            <Space wrap>
              {session.isCurrent ? (
                <Tag color="arcoblue">{t('auth.session.currentDevice')}</Tag>
              ) : null}
              <Tag color={session.revokedAt ? 'red' : 'green'}>
                {session.revokedAt
                  ? t('auth.session.status.revoked')
                  : t('auth.session.status.active')}
              </Tag>
            </Space>
          </div>

          <Descriptions
            column={2}
            data={[
              ...(session.username
                ? [{ label: t('system.user.username'), value: session.username }]
                : []),
              ...(session.nickname
                ? [{ label: t('system.profile.nickname'), value: session.nickname }]
                : []),
              { label: t('auth.session.ip'), value: session.lastIp || '-' },
              {
                label: t('auth.session.lastActive'),
                value: formatDateTime(
                  session.lastActivityAt || session.lastRefreshAt || session.createdAt,
                ),
              },
              {
                label: t('auth.session.refreshExpiresAt'),
                value: formatDateTime(session.refreshExpiresAt),
              },
              { label: t('system.profile.createdAt'), value: formatDateTime(session.createdAt) },
              {
                label: t('auth.session.status'),
                value: session.revokedAt
                  ? t('auth.session.status.revoked')
                  : t('auth.session.status.active'),
              },
              { label: t('auth.session.sessionId'), value: session.sessionId || '-' },
            ]}
          />

          <div className="auth-detail-grid">
            <Card className="detail-panel-card" title={t('auth.session.userAgent')} size="small">
              <Descriptions
                column={1}
                data={[
                  { label: t('auth.session.deviceName'), value: session.device || '-' },
                  { label: t('auth.session.browserName'), value: session.browser || '-' },
                  { label: t('auth.session.osName'), value: session.os || '-' },
                ]}
              />
            </Card>
            <Card className="detail-panel-card" title={t('auth.session.userAgentRaw')} size="small">
              <Typography.Paragraph
                type="secondary"
                className="auth-detail-user-agent"
                ellipsis={{ rows: 6, expandable: true }}
              >
                {session.userAgent || '-'}
              </Typography.Paragraph>
            </Card>
          </div>
        </Space>
      )}
    </AppModal>
  );
};

export default SessionDetailModal;
