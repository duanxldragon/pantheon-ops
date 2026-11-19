import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Message,
  Popconfirm,
  Progress,
  Space,
  Tag,
  Tabs,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { AppTable, PageContainer, PageEmpty, PageError, PageLoading } from '../../../components';
import { usePermission } from '../../../hooks/usePermission';
import { isForbiddenRequestError, isRequestError } from '../../../api/request';
import {
  cancelTask,
  retryTask,
  startTask,
  type DistributionEventRow,
  type DistributionTargetRow,
} from './api';
import { canRetryTask, canStartTask, isTaskActive, taskStatusColor } from './taskState';
import { useTaskDetail } from './useTaskDetail';
import './filedist.css';
import '../../../assets/list-page.css';

export default function FiledistDetail() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const taskId = Number(id);
  const { hasPerm } = usePermission();
  const { task, targets, events, loading, error, refreshError, refresh } = useTaskDetail(taskId);
  const [submitting, setSubmitting] = useState<{ taskId: number; action: string } | null>(null);
  const actionGeneration = useRef(0);
  const actionInFlight = useRef(false);
  useEffect(
    () => () => {
      actionGeneration.current += 1;
      actionInFlight.current = false;
    },
    [taskId],
  );

  const action = async (
    name: string,
    runner: () => Promise<unknown>,
    successKey: string,
    failureKey: string,
  ) => {
    if (actionInFlight.current) return;
    actionInFlight.current = true;
    const generation = actionGeneration.current;
    setSubmitting({ taskId, action: name });
    try {
      await runner();
      if (generation !== actionGeneration.current) return;
      Message.success(t(successKey));
      refresh();
    } catch (requestError) {
      if (generation !== actionGeneration.current) return;
      const key = isForbiddenRequestError(requestError)
        ? 'business.filedist.createForbidden'
        : isRequestError(requestError) && requestError.messageKey?.startsWith('business.')
          ? requestError.messageKey
          : failureKey;
      Message.error(t(key, { defaultValue: t(failureKey) }));
      refresh();
    } finally {
      if (generation === actionGeneration.current) {
        actionInFlight.current = false;
        setSubmitting(null);
      }
    }
  };
  const busy = submitting?.taskId === taskId;
  const status = (value: string) => (
    <Tag color={taskStatusColor(value)}>
      {t(`business.filedist.status.${value}`, {
        defaultValue: t('business.filedist.status.unknown'),
      })}
    </Tag>
  );
  const errorMessage = (value?: string) =>
    value ? t(value, { defaultValue: t('business.filedist.task.transferFailed') }) : '-';
  const targetColumns: ColumnProps<DistributionTargetRow>[] = [
    {
      title: t('business.filedist.hostname'),
      dataIndex: 'host',
      width: 180,
      render: (_, row) => `${row.host}:${row.port}`,
    },
    { title: t('business.filedist.targetPath'), dataIndex: 'targetPath', width: 220 },
    {
      title: t('business.filedist.status'),
      dataIndex: 'status',
      width: 120,
      render: (_, row) => status(row.status),
    },
    {
      title: t('business.filedist.progress'),
      dataIndex: 'progress',
      width: 140,
      render: (_, row) => <Progress percent={row.progress ?? 0} size="small" />,
    },
    {
      title: t('business.filedist.create.maxRetries'),
      dataIndex: 'retryCount',
      width: 100,
      render: (_, row) => `${row.retryCount}/${row.maxRetries}`,
    },
    {
      title: t('business.filedist.message'),
      dataIndex: 'error',
      width: 220,
      render: (_, row) => errorMessage(row.error),
    },
  ];
  const eventColumns: ColumnProps<DistributionEventRow>[] = [
    { title: t('business.filedist.time'), dataIndex: 'createdAt', width: 190 },
    { title: t('business.filedist.type'), dataIndex: 'type', width: 130 },
    { title: t('business.filedist.message'), dataIndex: 'message', width: 260 },
  ];
  if (loading)
    return (
      <PageContainer>
        <PageLoading />
      </PageContainer>
    );
  if (error || !task)
    return (
      <PageContainer>
        <PageError
          description={
            isForbiddenRequestError(error)
              ? t('business.filedist.createForbidden')
              : t(
                  !Number.isSafeInteger(taskId) ||
                    taskId <= 0 ||
                    (isRequestError(error) && error.status === 404)
                    ? 'business.filedist.taskMissing'
                    : 'common.loadFailedDesc',
                )
          }
          onRetry={refresh}
        />
      </PageContainer>
    );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel">
          <div className="filedist-detail__toolbar">
            <Button type="text" onClick={() => navigate('/business/filedist/tasks')}>
              {t('business.filedist.backToTasks')}
            </Button>
            <Space wrap>
              <Button disabled={busy} onClick={refresh}>
                {t('common.refresh')}
              </Button>
              {hasPerm('business:filedist:update') && task.status === 'pending' ? (
                <Button
                  type="primary"
                  disabled={busy || !canStartTask(task)}
                  loading={busy && submitting?.action === 'start'}
                  onClick={() =>
                    void action(
                      'start',
                      () => startTask(task.id),
                      'business.filedist.taskStarted',
                      'business.filedist.taskStartFailed',
                    )
                  }
                >
                  {t('business.filedist.start')}
                </Button>
              ) : null}
              {hasPerm('business:filedist:update') && isTaskActive(task.status) ? (
                <Popconfirm
                  title={t('business.filedist.cancelConfirm')}
                  onOk={() =>
                    action(
                      'cancel',
                      () => cancelTask(task.id),
                      'business.filedist.taskCanceled',
                      'business.filedist.taskCancelFailed',
                    )
                  }
                >
                  <Button
                    status="danger"
                    disabled={busy}
                    loading={busy && submitting?.action === 'cancel'}
                  >
                    {t('common.cancel')}
                  </Button>
                </Popconfirm>
              ) : null}
              {hasPerm('business:filedist:update') && canRetryTask(task) ? (
                <Button
                  disabled={busy}
                  loading={busy && submitting?.action === 'retry'}
                  onClick={() =>
                    void action(
                      'retry',
                      () => retryTask(task.id),
                      'business.filedist.taskRetried',
                      'business.filedist.taskRetryFailed',
                    )
                  }
                >
                  {t('common.retry')}
                </Button>
              ) : null}
            </Space>
          </div>
          {refreshError ? (
            <Alert
              type="warning"
              content={
                isForbiddenRequestError(refreshError)
                  ? t('business.filedist.createForbidden')
                  : t('business.filedist.refreshFailed')
              }
            />
          ) : null}
          <Descriptions
            className="filedist-detail__summary"
            column={1}
            title={t('business.filedist.taskOverview')}
            data={[
              { label: t('business.filedist.taskName'), value: task.name || '-' },
              { label: t('business.filedist.fileName'), value: task.fileName || '-' },
              { label: t('business.filedist.status'), value: status(task.status) },
              {
                label: t('business.filedist.overallProgress'),
                value: <Progress percent={task.progress ?? 0} />,
              },
              { label: t('business.filedist.totalHosts'), value: task.totalTargets ?? 0 },
              { label: t('business.filedist.success'), value: task.successCount ?? 0 },
              { label: t('business.filedist.failed'), value: task.failedCount ?? 0 },
              { label: t('business.filedist.create.scheduledAt'), value: task.scheduledAt || '-' },
              { label: t('business.filedist.startedAt'), value: task.startedAt || '-' },
              { label: t('business.filedist.finishedAt'), value: task.finishedAt || '-' },
              { label: t('business.filedist.message'), value: errorMessage(task.error) },
            ]}
          />
        </Card>
        <Card className="page-panel system-list__table-card">
          <Tabs defaultActiveTab="hosts">
            <Tabs.TabPane
              key="hosts"
              title={t('business.filedist.hostsTab', { count: targets.length })}
            >
              {targets.length ? (
                <AppTable
                  rowKey="id"
                  columns={targetColumns}
                  data={targets}
                  pagination={false}
                  scroll={{ x: 980 }}
                />
              ) : (
                <PageEmpty description={t('business.filedist.noTargets')} />
              )}
            </Tabs.TabPane>
            <Tabs.TabPane
              key="events"
              title={t('business.filedist.eventsTab', { count: events.length })}
            >
              {events.length ? (
                <AppTable
                  rowKey="id"
                  columns={eventColumns}
                  data={events}
                  pagination={false}
                  scroll={{ x: 600 }}
                />
              ) : (
                <PageEmpty description={t('business.filedist.noEvents')} />
              )}
            </Tabs.TabPane>
          </Tabs>
        </Card>
      </Space>
    </PageContainer>
  );
}
