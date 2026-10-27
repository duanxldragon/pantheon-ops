import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconRefresh } from '@arco-design/web-react/icon';
import { AppTable, PageContainer, PageEmpty, PageError, PageLoading } from '../../../components';
import { getAlertEventList, type AlertEventRow } from './api';
import '../../../assets/list-page.css';

export default function AlertRecordList() {
  const { t } = useTranslation();
  const [rows, setRows] = useState<AlertEventRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getAlertEventList({ page: 1, pageSize: 100 });
      setRows(result.items);
    } catch (requestError) {
      setRows([]);
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => { void loadData(); }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [loadData]);

  const columns = useMemo<ColumnProps<AlertEventRow>[]>(() => [
    { title: t('observability.alert_record.rule_name'), dataIndex: 'ruleName', width: 220 },
    { title: t('observability.alert_record.severity'), dataIndex: 'severity', width: 110, render: (value: string) => <Tag color={value === 'critical' ? 'red' : value === 'warning' ? 'orange' : 'blue'}>{value}</Tag> },
    { title: t('observability.alert_record.status'), dataIndex: 'status', width: 110, render: (value: string) => <Tag color={value === 'firing' ? 'red' : 'green'}>{t(`observability.alert_record.${value}`, { defaultValue: value })}</Tag> },
    { title: t('observability.alert_record.fired_at'), dataIndex: 'firedAt', width: 190, render: (value: string) => new Date(value).toLocaleString() },
    { title: t('observability.alert_record.resolved_at'), dataIndex: 'resolvedAt', width: 190, render: (value?: string) => value ? new Date(value).toLocaleString() : '-' },
    { title: t('observability.alert_record.value', { defaultValue: 'Value' }), dataIndex: 'value', width: 100 },
  ], [t]);

  return <PageContainer title={t('operations.observability.alert_record.menu')}>
    <Card className="page-panel system-list__table-card" extra={<Button icon={<IconRefresh />} onClick={() => void loadData()} disabled={loading}>{t('common.refresh')}</Button>}>
      {loading ? <PageLoading /> : null}
      {error && !loading ? <PageError onRetry={loadData} /> : null}
      {!loading && !error && rows.length === 0 ? <PageEmpty description={t('business.observability.alertRecord.empty')} /> : null}
      {!loading && !error && rows.length > 0 ? <AppTable rowKey="id" columns={columns} data={rows} pagination={false} scroll={{ x: 'max-content' }} /> : null}
    </Card>
  </PageContainer>;
}
