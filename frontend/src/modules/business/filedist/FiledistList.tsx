import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Card, Modal, Popconfirm, Space, Tabs, Tag, Upload, Message } from '@arco-design/web-react';
import { IconDelete, IconUpload } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { AppTable, GovernanceSummaryBar, PageContainer, PageEmpty, PageError, PageLoading } from '../../../components';
import { deleteFile, deleteTask, getFileList, getTaskList, uploadFile, type DistributionTaskRow, type FileRecordRow } from './api';
import '../../../assets/list-page.css';

const formatSize = (size: number) => {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

const taskStatusColor = (status: string) => {
  if (status === 'completed') {
    return 'green';
  }
  if (status === 'failed') {
    return 'red';
  }
  return 'arcoblue';
};

export default function FiledistList() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [files, setFiles] = useState<FileRecordRow[]>([]);
  const [tasks, setTasks] = useState<DistributionTaskRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [uploadVisible, setUploadVisible] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const [fileResponse, taskResponse] = await Promise.all([getFileList({ page: 1, pageSize: 100 }), getTaskList({ page: 1, pageSize: 100 })]);
      setFiles(fileResponse.items || []); setTasks(taskResponse.items || []);
    } catch (requestError) { setError(requestError); } finally { setLoading(false); }
  }, []);
  useEffect(() => {
    const timeoutId = window.setTimeout(() => { void load(); }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [load]);

  const submitUpload = async () => {
    if (!selectedFile) return;
    setUploading(true);
    try { await uploadFile(selectedFile); Message.success(t('business.filedist.uploadSuccess')); setUploadVisible(false); setSelectedFile(null); await load(); }
    catch { Message.error(t('business.filedist.uploadFailed')); }
    finally { setUploading(false); }
  };
  const handleDeleteFile = useCallback(async (row: FileRecordRow) => { try { await deleteFile(row.id); await load(); } catch { Message.error(t('common.failed')); } }, [load, t]);
  const handleDeleteTask = useCallback(async (row: DistributionTaskRow) => { try { await deleteTask(row.id); await load(); } catch { Message.error(t('common.failed')); } }, [load, t]);

  const fileColumns = useMemo<ColumnProps<FileRecordRow>[]>(() => [
    {
      title: t('business.filedist.fileName'),
      dataIndex: 'name',
      width: 320,
      fixed: 'left',
      render: (_: unknown, row) => (
        <Space>
          <span>{row.fileName || row.name || row.displayName || '-'}</span>
          <Popconfirm title={t('business.filedist.deleteConfirm')} onOk={() => void handleDeleteFile(row)}>
            <Button type="text" status="danger" size="small" icon={<IconDelete />}>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
    { title: t('business.filedist.size'), dataIndex: 'size', width: 120, render: (_: unknown, row) => formatSize(row.fileSize || row.size || 0) },
    { title: t('business.filedist.storageType'), dataIndex: 'storageType', width: 130, render: (_: unknown, row) => row.storageType || 'local' },
    { title: t('business.filedist.status'), dataIndex: 'status', width: 110, render: (_: unknown, row) => <Tag color="green">{row.status || 'active'}</Tag> },
    { title: t('business.filedist.createdAt'), dataIndex: 'createdAt', width: 190, render: (_: unknown, row) => row.createdAt || row.uploadedAt || '-' },
  ], [handleDeleteFile, t]);
  const taskColumns = useMemo<ColumnProps<DistributionTaskRow>[]>(() => [
    { title: t('business.filedist.taskName'), dataIndex: 'name', width: 260 },
    { title: t('business.filedist.fileName'), dataIndex: 'fileName', width: 220, render: (_: unknown, row) => row.fileName || '-' },
    { title: t('business.filedist.status'), dataIndex: 'status', width: 120, render: (_: unknown, row) => <Tag color={taskStatusColor(row.status)}>{row.status}</Tag> },
    { title: t('business.filedist.progress'), dataIndex: 'progress', width: 110, render: (_: unknown, row) => `${row.progress ?? 0}%` },
    { title: t('business.filedist.totalHosts'), dataIndex: 'totalTargets', width: 120 },
    { title: t('common.action'), width: 150, fixed: 'right', render: (_: unknown, row) => <Space><Button type="text" size="small" onClick={() => navigate(`/business/filedist/tasks/${row.id}`)}>{t('common.view')}</Button><Popconfirm title={t('common.deleteConfirm')} onOk={() => void handleDeleteTask(row)}><Button type="text" status="danger" size="small" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm></Space> },
  ], [handleDeleteTask, navigate, t]);

  const activeTab = location.pathname.includes('/tasks') ? 'tasks' : 'files';
  return <PageContainer><Space direction="vertical" size={16} className="system-page-template">
    <GovernanceSummaryBar eyebrow={t('operations.filedist.menu')} title={t('operations.filedist.menu')} description={t('business.filedist.taskOverview')} metrics={[{ key: 'files', label: t('business.filedist.filesTab', { count: files.length }), value: files.length }, { key: 'tasks', label: t('business.filedist.tasksTab', { count: tasks.length }), value: tasks.length }]} />
    <Card className="page-panel system-list__table-card"><Tabs activeTab={activeTab} onChange={(key) => navigate(key === 'tasks' ? '/business/filedist/tasks' : '/business/filedist/files')}>
      <Tabs.TabPane key="files" title={t('business.filedist.filesTab', { count: files.length })}><Space direction="vertical" style={{ width: '100%' }}><Button type="primary" icon={<IconUpload />} onClick={() => setUploadVisible(true)}>{t('business.filedist.upload')}</Button>{loading ? <PageLoading /> : null}{error && !loading ? <PageError onRetry={() => void load()} /> : null}{!loading && !error && files.length === 0 ? <PageEmpty description={t('business.filedist.noFiles')} /> : null}{!loading && !error && files.length > 0 ? <AppTable rowKey="id" data={files} columns={fileColumns} pagination={false} scroll={{ x: 'max-content' }} /> : null}</Space></Tabs.TabPane>
      <Tabs.TabPane key="tasks" title={t('business.filedist.tasksTab', { count: tasks.length })}>{loading ? <PageLoading /> : null}{error && !loading ? <PageError onRetry={() => void load()} /> : null}{!loading && !error && tasks.length === 0 ? <PageEmpty description={t('business.filedist.noTasks')} /> : null}{!loading && !error && tasks.length > 0 ? <AppTable rowKey="id" data={tasks} columns={taskColumns} pagination={false} scroll={{ x: 'max-content' }} /> : null}</Tabs.TabPane>
    </Tabs></Card>
  </Space><Modal title={t('business.filedist.upload')} visible={uploadVisible} onCancel={() => { setUploadVisible(false); setSelectedFile(null); }} onOk={() => void submitUpload()} confirmLoading={uploading}><Upload showUploadList={false} autoUpload={false} onChange={(fileList) => setSelectedFile(fileList[0]?.originFile || null)}><Button icon={<IconUpload />}>{t('business.filedist.selectFile')}</Button></Upload></Modal></PageContainer>;
}
