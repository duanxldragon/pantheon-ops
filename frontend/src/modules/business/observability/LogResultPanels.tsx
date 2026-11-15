import { useMemo } from 'react';
import { Tabs, Tag } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import type { ReactNode } from 'react';
import { AppTable, buildStandardPagination } from '../../../components';
import { boundedRawLogRows, summarizeLogRows, type LogRow } from './log-results';

export default function LogResultPanels({ rows, logs }: Readonly<{ rows: LogRow[]; logs: ReactNode }>) {
  const { t } = useTranslation();
  const summary = useMemo(() => summarizeLogRows(rows), [rows]);
  const raw = useMemo(() => boundedRawLogRows(rows), [rows]);
  return <Tabs defaultActiveTab="logs" className="observability-log-results">
    <Tabs.TabPane key="logs" title={t('business.observability.logs.tabs.logs')}>{logs}</Tabs.TabPane>
    <Tabs.TabPane key="trend" title={t('business.observability.logs.tabs.trend')}>
      <p className="observability-log-results__note">{t('business.observability.logs.returnedRowsOnly')}</p>
      {summary.invalidTimes ? <Tag>{t('business.observability.logs.invalidTimes', { count: summary.invalidTimes })}</Tag> : null}
      <AppTable rowKey="time" data={summary.trend} columns={[
        { title: t('business.observability.logs.startUTC'), dataIndex: 'time' },
        { title: t('business.observability.logs.returnedCount'), dataIndex: 'count' },
      ]} pagination={buildStandardPagination(t, { pageSize: 20, total: summary.trend.length })} />
    </Tabs.TabPane>
    <Tabs.TabPane key="fields" title={t('business.observability.logs.tabs.fields')}>
      <p className="observability-log-results__note">{t('business.observability.logs.returnedRowsOnly')}</p>
      {summary.fieldsLimited ? <Tag>{t('business.observability.logs.fieldsLimited')}</Tag> : null}
      <AppTable rowKey="name" data={summary.fields} columns={[
        { title: t('business.observability.logs.fieldName'), dataIndex: 'name', ellipsis: true },
        { title: t('business.observability.logs.returnedCount'), dataIndex: 'rows' },
        { title: t('business.observability.logs.distinctValues'), dataIndex: 'distinct' },
      ]} pagination={buildStandardPagination(t, { pageSize: 20, total: summary.fields.length })} />
    </Tabs.TabPane>
    <Tabs.TabPane key="raw" title={t('business.observability.logs.tabs.raw')}>
      <p className="observability-log-results__note">{t('business.observability.logs.rawProjection')}</p>
      {raw.omittedRows ? <Tag>{t('business.observability.logs.rawLimited', { count: raw.omittedRows })}</Tag> : null}
      <pre className="observability-log-results__raw" role="region" tabIndex={0} aria-label={t('business.observability.logs.tabs.raw')}>{raw.text}</pre>
    </Tabs.TabPane>
  </Tabs>;
}
