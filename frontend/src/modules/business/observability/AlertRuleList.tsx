import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Message, Popconfirm, Switch, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconRefresh } from '@arco-design/web-react/icon';
import { AppTable, PageContainer, PageEmpty, PageError, PageLoading } from '../../../components';
import { deleteAlertRule, getAlertRuleList, toggleAlertRule, type AlertRuleRow } from './api';
import '../../../assets/list-page.css';

export default function AlertRuleList() {
  const { t } = useTranslation();
  const [rows, setRows] = useState<AlertRuleRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getAlertRuleList({ page: 1, pageSize: 100 });
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

  const handleToggle = useCallback(async (row: AlertRuleRow, enabled: boolean) => {
    try {
      await toggleAlertRule(row.id, enabled);
      Message.success(t(enabled ? 'observability.alert_rule.enable_success' : 'observability.alert_rule.disable_success'));
      await loadData();
    } catch { Message.error(t('common.failed')); }
  }, [loadData, t]);

  const handleDelete = useCallback(async (row: AlertRuleRow) => {
    try {
      await deleteAlertRule(row.id);
      Message.success(t('common.deleteSuccess'));
      await loadData();
    } catch { Message.error(t('common.failed')); }
  }, [loadData, t]);

  const columns = useMemo<ColumnProps<AlertRuleRow>[]>(() => [
    { title: t('observability.alert_rule.name'), dataIndex: 'name', width: 220 },
    { title: t('observability.alert_rule.metric_source'), dataIndex: 'datasourceName', width: 180 },
    { title: t('observability.alert_rule.promql'), dataIndex: 'query', ellipsis: true },
    { title: t('observability.alert_rule.severity'), dataIndex: 'severity', width: 110, render: (value: string) => <Tag color={value === 'critical' ? 'red' : value === 'warning' ? 'orange' : 'blue'}>{value}</Tag> },
    { title: t('observability.alert_rule.status'), dataIndex: 'enabled', width: 100, render: (enabled: boolean, row) => <Switch checked={enabled} onChange={(checked) => void handleToggle(row, checked)} /> },
    { title: t('observability.alert_rule.fired_count', { defaultValue: 'Fired' }), dataIndex: 'firedCount', width: 90 },
    { title: t('common.action'), width: 100, render: (_value, row) => <Popconfirm title={t('observability.alert_rule.delete_confirm')} onOk={() => void handleDelete(row)}><Button type="text" size="small" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> },
  ], [handleDelete, handleToggle, t]);

  return <PageContainer title={t('operations.observability.alert_rule.menu')}>
    <Card className="page-panel system-list__table-card" extra={<Button icon={<IconRefresh />} onClick={() => void loadData()} disabled={loading}>{t('common.refresh')}</Button>}>
      {loading ? <PageLoading /> : null}
      {error && !loading ? <PageError onRetry={loadData} /> : null}
      {!loading && !error && rows.length === 0 ? <PageEmpty description={t('business.observability.alertRule.empty')} /> : null}
      {!loading && !error && rows.length > 0 ? <AppTable rowKey="id" columns={columns} data={rows} pagination={false} scroll={{ x: 'max-content' }} /> : null}
    </Card>
  </PageContainer>;
}
