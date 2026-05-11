import React from 'react';
import { Avatar, Card, Descriptions, Grid, Space, Tag, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import type { UserDetail as UserDetailData } from './api';
import { formatDateTime } from '../../../core/format/dateTime';

const Row = Grid.Row;
const Col = Grid.Col;

interface UserDetailContentProps {
  detail: UserDetailData;
  orgEnabled?: boolean;
}

const UserDetailContent: React.FC<UserDetailContentProps> = ({ detail, orgEnabled = true }) => {
  const { t } = useTranslation();
  const summaryItems = [
    ...(orgEnabled
      ? [
          { label: t('system.user.dept'), value: detail.deptName || '-' },
          { label: t('system.user.post'), value: detail.postName || '-' },
        ]
      : []),
    {
      label: t('system.user.roles'),
      value: detail.roleKeys.length ? (
        <Space wrap>
          {detail.roleKeys.map((item) => (
            <Tag key={item}>{item}</Tag>
          ))}
        </Space>
      ) : (
        '-'
      ),
    },
    { label: t('system.user.createdAt'), value: formatDateTime(detail.createdAt) },
    { label: t('system.user.updatedAt'), value: formatDateTime(detail.updatedAt) },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Space size={16} align="start" style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space size={16} align="center">
            <Avatar size={56}>
              {detail.avatar ? (
                <img src={detail.avatar} alt={detail.nickname || detail.username} />
              ) : (
                detail.nickname?.charAt(0) || detail.username.charAt(0)
              )}
            </Avatar>
            <Space direction="vertical" size={4}>
              <Typography.Title heading={5} style={{ margin: 0 }}>
                {detail.nickname || detail.username}
              </Typography.Title>
              <Typography.Text type="secondary">{detail.username}</Typography.Text>
            </Space>
          </Space>
          <Tag color={detail.status === 1 ? 'green' : 'red'}>
            {detail.status === 1
              ? t('system.user.status.enabled')
              : t('system.user.status.disabled')}
          </Tag>
        </Space>
      </Card>

      <Row gutter={16}>
        <Col span={12}>
          <Card title={t('common.basicInfo')}>
            <Descriptions
              column={1}
              data={[
                { label: t('system.user.username'), value: detail.username || '-' },
                { label: t('system.user.nickname'), value: detail.nickname || '-' },
                { label: t('system.user.avatar'), value: detail.avatar || '-' },
                { label: t('system.user.email'), value: detail.email || '-' },
                { label: t('system.user.phone'), value: detail.phone || '-' },
              ]}
            />
          </Card>
        </Col>
        <Col span={12}>
          <Card title={t('system.user.summary')}>
            <Descriptions column={1} data={summaryItems} />
          </Card>
        </Col>
      </Row>
    </Space>
  );
};

export default UserDetailContent;
