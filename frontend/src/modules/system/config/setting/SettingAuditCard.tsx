import React from 'react';
import { Button, Card, Space, Tag, Typography } from '@arco-design/web-react';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import {
  AppTable,
  buildStandardPagination,
  TABLE_ACTION_COLUMN_WIDTH,
  withTableColumnPriority,
} from '../../../../components';
import { formatDateTime } from '../../../../core/format/dateTime';
import type { SettingAuditChange, SettingAuditRow } from './api';

interface SettingAuditCardProps {
  rows: SettingAuditRow[];
  total: number;
  loading: boolean;
  page: number;
  pageSize: number;
  canExportAudit: boolean;
  canViewOperationLog: boolean;
  onChange: TableProps<SettingAuditRow>['onChange'];
  onExport: () => void;
  onViewOperationLog: (id: number) => void;
}

const SettingAuditCard: React.FC<SettingAuditCardProps> = ({
  rows,
  total,
  loading,
  page,
  pageSize,
  canExportAudit,
  canViewOperationLog,
  onChange,
  onExport,
  onViewOperationLog,
}) => {
  const { t } = useTranslation();

  const renderAuditChange = (change: SettingAuditChange) => {
    const label = t(`system.setting.item.${change.settingKey}`, change.settingKey);
    if (change.isEncrypted === 1) {
      return (
        <Space size={6} wrap>
          <Typography.Text>{label}</Typography.Text>
          <Tag color="red">{t('system.setting.audit.sensitiveChanged')}</Tag>
        </Space>
      );
    }
    const oldValue = change.oldValue || '-';
    const newValue = change.newValue || '-';
    return (
      <Space size={6} wrap>
        <Typography.Text>{label}</Typography.Text>
        <Typography.Text type="secondary">{oldValue}</Typography.Text>
        <Typography.Text type="secondary">→</Typography.Text>
        <Typography.Text>{newValue}</Typography.Text>
      </Space>
    );
  };

  const columns: ColumnProps<SettingAuditRow>[] = [
    {
      title: t('system.setting.audit.operator'),
      dataIndex: 'operName',
      render: (value: string) => value || '-',
    },
    withTableColumnPriority(
      {
        title: t('system.setting.audit.ip'),
        dataIndex: 'operIp',
        render: (value: string) => value || '-',
      },
      'medium',
    ),
    {
      title: t('system.setting.audit.changes'),
      dataIndex: 'changes',
      render: (changes: SettingAuditChange[]) => (
        <Space direction="vertical" size={4}>
          {changes.length > 0 ? (
            changes.map((change) => <div key={change.settingKey}>{renderAuditChange(change)}</div>)
          ) : (
            <Typography.Text type="secondary">{t('system.setting.audit.noChange')}</Typography.Text>
          )}
        </Space>
      ),
    },
    {
      title: t('system.setting.audit.status'),
      dataIndex: 'status',
      render: (value: number) =>
        value === 1 ? (
          <Tag color="green">{t('auth.loginLog.status.success')}</Tag>
        ) : (
          <Tag color="red">{t('auth.loginLog.status.failed')}</Tag>
        ),
    },
    {
      title: t('system.setting.audit.operTime'),
      dataIndex: 'operTime',
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.single,
      render: (_, record) => (
        <Button
          type="text"
          size="small"
          disabled={!canViewOperationLog}
          onClick={() => onViewOperationLog(record.id)}
        >
          {t('system.setting.audit.viewOperationLog')}
        </Button>
      ),
    },
  ];

  return (
    <Card className="page-panel system-list__table-card setting-page__audit-card">
      <div className="setting-page__audit-header">
        <div>
          <Typography.Text style={{ fontWeight: 600 }}>
            {t('system.setting.audit.title')}
          </Typography.Text>
          <Typography.Paragraph type="secondary" style={{ margin: '4px 0 0' }}>
            {t('common.total', { count: total })}
          </Typography.Paragraph>
        </div>
        <Space>
          <Button
            size="small"
            onClick={() => {
              onExport();
            }}
            disabled={!canExportAudit}
          >
            {t('common.export')}
          </Button>
        </Space>
      </div>
      <AppTable<SettingAuditRow>
        className="system-list__table"
        rowKey="id"
        data={rows}
        columns={columns}
        loading={loading}
        scroll={{ x: 'max-content' }}
        onChange={onChange}
        pagination={buildStandardPagination(t, {
          current: page,
          pageSize,
          total,
          sizeOptions: [5, 10, 20, 50],
        })}
      />
    </Card>
  );
};

export default SettingAuditCard;
