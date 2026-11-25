import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Alert,
  Button,
  Card,
  DatePicker,
  Divider,
  Form,
  Input,
  InputNumber,
  Message,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Tabs,
  Tag,
  Upload,
} from '@arco-design/web-react';
import { IconDelete, IconPlus, IconUpload } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppModal,
  AppTable,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
  SubmitBar,
  buildStandardPagination,
} from '../../../components';
import { usePermission } from '../../../hooks/usePermission';
import { isForbiddenRequestError, isRequestError } from '../../../api/request';
import { getBizScopeOptions, type BizScopeOptionItem } from '../bizscope/api';
import { getGroupList, type GroupRow } from '../cmdb/group/api';
import DeployTargetSelector from '../deploy/task/DeployTargetSelector';
import type { DeployTargetResolution } from '../deploy/api';
import PasswordInput from '../../../components/patterns/forms/PasswordInput';
import SecretTextArea from '../shared/SecretTextArea';
import { isTaskActive, TASK_STATUSES, taskStatusColor } from './taskState';
import { buildCreateTaskPayload, type CreateTaskFormValues } from './createTaskPayload';
import { FILE_CHOICE_PAGE_SIZE, useFileChoices } from './useFileChoices';
import { useFiledistListQuery } from './useFiledistListQuery';
import {
  createTask,
  deleteFile,
  deleteTask,
  getFileList,
  getTaskList,
  uploadFile,
  type CreateTaskPayload,
  type DistributionTaskRow,
  type FileRecordRow,
} from './api';
import '../../../assets/list-page.css';
import './filedist.css';

const formatSize = (size: number) => {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

function flattenGroups(groups: GroupRow[]): GroupRow[] {
  return groups.flatMap((group) => [group, ...flattenGroups(group.children || [])]);
}

export default function FiledistList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const location = useLocation();
  const navigate = useNavigate();
  const [createForm] = Form.useForm<CreateTaskFormValues>();

  const [filesPage, setFilesPage] = useState(1);
  const [filesPageSize, setFilesPageSize] = useState(20);

  const [tasksPage, setTasksPage] = useState(1);
  const [tasksPageSize, setTasksPageSize] = useState(20);

  const [keyword, setKeyword] = useState('');
  const [taskStatus, setTaskStatus] = useState('');

  const [uploadVisible, setUploadVisible] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const [createVisible, setCreateVisible] = useState(false);
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const {
    items: createFiles,
    total: fileChoicesTotal,
    page: fileChoicesPage,
    loading: fileChoicesLoading,
    error: fileChoicesError,
    keyword: fileChoicesKeyword,
    search: searchFileChoices,
    setPage: setFileChoicesPage,
    reload: reloadFileChoices,
  } = useFileChoices(createVisible);
  const [scopeOptions, setScopeOptions] = useState<BizScopeOptionItem[]>([]);
  const [groups, setGroups] = useState<Array<{ id: number; label: string }>>([]);
  const [selectedScopeId, setSelectedScopeId] = useState<number>();
  const [targetResolution, setTargetResolution] = useState<DeployTargetResolution | null>(null);
  const [scheduledAtRaw, setScheduledAtRaw] = useState<unknown>(undefined);
  const [authMode, setAuthMode] = useState<'password' | 'privateKey'>('password');
  const [createOptionsLoading, setCreateOptionsLoading] = useState(false);
  const [createOptionsError, setCreateOptionsError] = useState<unknown>(null);
  const createGeneration = useRef(0);
  const createOptionsGeneration = useRef(0);
  const createController = useRef<AbortController | null>(null);
  const createInFlight = useRef(false);
  useEffect(
    () => () => {
      createGeneration.current += 1;
      createOptionsGeneration.current += 1;
      createController.current?.abort();
    },
    [],
  );

  const lookupFiles = useCallback(
    (signal: AbortSignal) =>
      getFileList({ keyword, page: filesPage, pageSize: filesPageSize }, signal),
    [keyword, filesPage, filesPageSize],
  );
  const lookupTasks = useCallback(
    (signal: AbortSignal) =>
      getTaskList(
        { keyword, status: taskStatus, page: tasksPage, pageSize: tasksPageSize },
        signal,
      ),
    [keyword, taskStatus, tasksPage, tasksPageSize],
  );
  const {
    items: files,
    total: filesTotal,
    loading: filesLoading,
    error: filesError,
    refresh: loadFiles,
  } = useFiledistListQuery(lookupFiles);
  const {
    items: tasks,
    total: tasksTotal,
    loading: tasksLoading,
    error: tasksError,
    refresh: loadTasks,
    refreshSilently: refreshTasksSilently,
  } = useFiledistListQuery(lookupTasks);

  const hasActiveTasks = useMemo(() => tasks.some((task) => isTaskActive(task.status)), [tasks]);
  useEffect(() => {
    if (!hasActiveTasks) {
      return;
    }
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      await refreshTasksSilently();
      if (!stopped) timer = setTimeout(() => void poll(), 5000);
    };
    timer = setTimeout(() => void poll(), 5000);
    return () => {
      clearTimeout(timer);
      stopped = true;
    };
  }, [hasActiveTasks, refreshTasksSilently]);

  const handleKeywordChange = useCallback((value: string) => {
    setKeyword(value);
    setFilesPage(1);
    setTasksPage(1);
  }, []);

  const handleClearAll = useCallback(() => {
    setKeyword('');
    setTaskStatus('');
    setFilesPage(1);
    setTasksPage(1);
  }, []);

  const submitUpload = async () => {
    if (!selectedFile) return;
    setUploading(true);
    try {
      await uploadFile(selectedFile);
      Message.success(t('business.filedist.uploadSuccess'));
      setUploadVisible(false);
      setSelectedFile(null);
      await loadFiles();
    } catch {
      Message.error(t('business.filedist.uploadFailed'));
    } finally {
      setUploading(false);
    }
  };
  const handleDeleteFile = useCallback(
    async (row: FileRecordRow) => {
      try {
        await deleteFile(row.id);
        Message.success(t('common.deleteSuccess'));
        await loadFiles();
      } catch {
        Message.error(t('common.failed'));
      }
    },
    [loadFiles, t],
  );
  const handleDeleteTask = useCallback(
    async (row: DistributionTaskRow) => {
      try {
        await deleteTask(row.id);
        Message.success(t('common.deleteSuccess'));
        await loadTasks();
      } catch {
        Message.error(t('common.failed'));
      }
    },
    [loadTasks, t],
  );

  const openCreate = useCallback(async () => {
    createGeneration.current += 1;
    const activation = ++createOptionsGeneration.current;
    createController.current?.abort();
    const controller = new AbortController();
    createController.current = controller;
    createForm.resetFields();
    setTargetResolution(null);
    setSelectedScopeId(undefined);
    setCreateError(null);
    setScheduledAtRaw(undefined);
    setAuthMode('password');
    searchFileChoices('');
    setFileChoicesPage(1);
    setScopeOptions([]);
    setGroups([]);
    setCreateOptionsError(null);
    setCreateOptionsLoading(true);
    setCreateVisible(true);
    try {
      const [scopeResponse, groupResponse] = await Promise.all([
        getBizScopeOptions(controller.signal),
        getGroupList(),
      ]);
      if (controller.signal.aborted || activation !== createOptionsGeneration.current) return;
      setScopeOptions(scopeResponse || []);
      setGroups(
        flattenGroups(groupResponse || []).map((group) => ({ id: group.id, label: group.name })),
      );
    } catch (requestError) {
      if (!controller.signal.aborted && activation === createOptionsGeneration.current)
        setCreateOptionsError(requestError);
    } finally {
      if (!controller.signal.aborted && activation === createOptionsGeneration.current)
        setCreateOptionsLoading(false);
    }
  }, [createForm, searchFileChoices, setFileChoicesPage]);

  const closeCreate = useCallback(() => {
    if (createInFlight.current) return;
    createGeneration.current += 1;
    createOptionsGeneration.current += 1;
    createController.current?.abort();
    setCreateVisible(false);
    setCreateSubmitting(false);
    setCreateError(null);
    setTargetResolution(null);
    setSelectedScopeId(undefined);
    setScheduledAtRaw(undefined);
    createForm.resetFields();
  }, [createForm]);

  const submitCreate = async () => {
    if (createInFlight.current) return;
    const generation = createGeneration.current;
    let values: CreateTaskFormValues;
    try {
      values = await createForm.validate();
    } catch {
      return;
    }
    // Form validation is asynchronous; a second click may have validated too.
    if (createInFlight.current || generation !== createGeneration.current) return;
    if (
      !targetResolution ||
      targetResolution.selection.businessScopeId !== values.businessScopeId ||
      targetResolution.hosts.length === 0 ||
      targetResolution.unavailableCount > 0 ||
      targetResolution.ineligibleCount > 0
    ) {
      Message.warning(t('business.filedist.create.selectionRequired'));
      return;
    }
    let payload: CreateTaskPayload;
    try {
      payload = buildCreateTaskPayload(values, targetResolution, authMode, scheduledAtRaw);
    } catch (validationError) {
      setCreateError(
        t(
          validationError instanceof Error
            ? validationError.message
            : 'business.filedist.createFailed',
          { defaultValue: t('business.filedist.createFailed') },
        ),
      );
      return;
    }
    setCreateSubmitting(true);
    createInFlight.current = true;
    setCreateError(null);
    try {
      const created = await createTask(payload);
      if (generation !== createGeneration.current) return;
      Message.success(t('business.filedist.createSuccess'));
      createInFlight.current = false;
      closeCreate();
      navigate(`/business/filedist/tasks/${created.id}`);
    } catch (createException) {
      if (generation !== createGeneration.current) return;
      let message: string;
      if (isForbiddenRequestError(createException)) {
        message = t('business.filedist.createForbidden');
      } else if (
        isRequestError(createException) &&
        createException.messageKey &&
        createException.messageKey !== 'create_failed'
      ) {
        message = t(createException.messageKey, {
          defaultValue: t('business.filedist.createFailed'),
        });
      } else {
        message = t('business.filedist.createFailed');
      }
      setCreateError(message);
    } finally {
      createInFlight.current = false;
      setCreateSubmitting(false);
    }
  };

  const createSubmitDisabled = Boolean(
    createOptionsLoading ||
    createOptionsError ||
    fileChoicesLoading ||
    fileChoicesError ||
    !createFiles.length ||
    !targetResolution ||
    !selectedScopeId ||
    targetResolution.selection.businessScopeId !== selectedScopeId ||
    targetResolution.hosts.length === 0 ||
    targetResolution.unavailableCount > 0 ||
    targetResolution.ineligibleCount > 0,
  );

  const fileColumns = useMemo<ColumnProps<FileRecordRow>[]>(
    () => [
      {
        title: t('business.filedist.fileName'),
        dataIndex: 'name',
        width: 320,
        fixed: 'left',
        render: (_: unknown, row) => (
          <Space>
            <span>{row.fileName || row.name || row.displayName || '-'}</span>
            {hasPerm('business:filedist:delete') ? (
              <Popconfirm
                title={t('business.filedist.deleteConfirm')}
                onOk={() => void handleDeleteFile(row)}
              >
                <Button type="text" status="danger" size="small" icon={<IconDelete />}>
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('business.filedist.size'),
        dataIndex: 'size',
        width: 120,
        render: (_: unknown, row) => formatSize(row.fileSize || row.size || 0),
      },
      {
        title: t('business.filedist.storageType'),
        dataIndex: 'storageType',
        width: 130,
        render: (_: unknown, row) =>
          t(`business.filedist.storage.${row.storageType || 'local'}`, {
            defaultValue: t('business.filedist.status.unknown'),
          }),
      },
      {
        title: t('business.filedist.status'),
        dataIndex: 'status',
        width: 110,
        render: (_: unknown, row) => (
          <Tag color="green">
            {t(`business.filedist.status.${row.status || 'active'}`, {
              defaultValue: t('business.filedist.status.unknown'),
            })}
          </Tag>
        ),
      },
      {
        title: t('business.filedist.createdAt'),
        dataIndex: 'createdAt',
        width: 190,
        render: (_: unknown, row) => row.createdAt || row.uploadedAt || '-',
      },
    ],
    [handleDeleteFile, hasPerm, t],
  );
  const taskColumns = useMemo<ColumnProps<DistributionTaskRow>[]>(
    () => [
      { title: t('business.filedist.taskName'), dataIndex: 'name', width: 220 },
      {
        title: t('business.filedist.fileName'),
        dataIndex: 'fileName',
        width: 180,
        render: (_: unknown, row) => row.fileName || '-',
      },
      {
        title: t('business.filedist.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={taskStatusColor(row.status)}>
            {t(`business.filedist.status.${row.status}`, {
              defaultValue: t('business.filedist.status.unknown'),
            })}
          </Tag>
        ),
      },
      {
        title: t('business.filedist.progress'),
        dataIndex: 'progress',
        width: 100,
        render: (_: unknown, row) => `${row.progress ?? 0}%`,
      },
      { title: t('business.filedist.totalHosts'), dataIndex: 'totalTargets', width: 110 },
      {
        title: t('common.action'),
        width: 150,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space>
            <Button
              type="text"
              size="small"
              onClick={() => navigate(`/business/filedist/tasks/${row.id}`)}
            >
              {t('common.detail')}
            </Button>
            {hasPerm('business:filedist:delete') && !isTaskActive(row.status) ? (
              <Popconfirm title={t('common.deleteConfirm')} onOk={() => void handleDeleteTask(row)}>
                <Button type="text" status="danger" size="small" icon={<IconDelete />}>
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [handleDeleteTask, hasPerm, navigate, t],
  );

  const activeTab = location.pathname.includes('/tasks') ? 'tasks' : 'files';
  const canCreate = hasPerm('business:filedist:create');

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('operations.filedist.menu')}
          title={t('operations.filedist.menu')}
          description={t('business.filedist.taskOverview')}
          metrics={[
            {
              key: 'files',
              label: t('business.filedist.filesTab', { count: filesTotal }),
              value: filesTotal,
            },
            {
              key: 'tasks',
              label: t('business.filedist.tasksTab', { count: tasksTotal }),
              value: tasksTotal,
            },
          ]}
        />
        <Card className="page-panel system-list__table-card">
          <SearchToolbar
            keyword={keyword}
            keywordPlaceholder={t('common.keyword')}
            onKeywordChange={handleKeywordChange}
            inlineFilters={
              activeTab === 'tasks' ? (
                <Select
                  value={taskStatus || undefined}
                  onChange={(value) => {
                    setTaskStatus(value || '');
                    setTasksPage(1);
                  }}
                  placeholder={t('business.filedist.status')}
                  allowClear
                >
                  {TASK_STATUSES.map((status) => (
                    <Select.Option key={status} value={status}>
                      {t(`business.filedist.status.${status}`)}
                    </Select.Option>
                  ))}
                </Select>
              ) : undefined
            }
            hasActiveFilters={Boolean(keyword || taskStatus)}
            onClearAll={handleClearAll}
          />
          <Tabs
            activeTab={activeTab}
            onChange={(key) =>
              navigate(key === 'tasks' ? '/business/filedist/tasks' : '/business/filedist/files')
            }
          >
            <Tabs.TabPane
              key="files"
              title={t('business.filedist.filesTab', { count: filesTotal })}
            >
              <Space direction="vertical" style={{ width: '100%' }}>
                {canCreate ? (
                  <Button
                    type="primary"
                    icon={<IconUpload />}
                    onClick={() => setUploadVisible(true)}
                  >
                    {t('business.filedist.upload')}
                  </Button>
                ) : null}
                {filesLoading && files.length === 0 ? <PageLoading /> : null}
                {filesError && !filesLoading && files.length === 0 ? (
                  <PageError
                    description={
                      isForbiddenRequestError(filesError)
                        ? t('business.filedist.createForbidden')
                        : undefined
                    }
                    onRetry={() => void loadFiles()}
                  />
                ) : null}
                {filesError && files.length > 0 ? (
                  <Alert
                    type="warning"
                    content={
                      isForbiddenRequestError(filesError)
                        ? t('business.filedist.createForbidden')
                        : t('business.filedist.refreshFailed')
                    }
                  />
                ) : null}
                {!filesLoading && !filesError && files.length === 0 ? (
                  <PageEmpty description={t('business.filedist.noFiles')} />
                ) : null}
                {files.length > 0 ? (
                  <AppTable
                    rowKey="id"
                    data={files}
                    columns={fileColumns}
                    loading={filesLoading}
                    scroll={{ x: 'max-content' }}
                    pagination={buildStandardPagination(t, {
                      current: filesPage,
                      pageSize: filesPageSize,
                      total: filesTotal,
                      onChange: (nextPage, nextPageSize) => {
                        setFilesPage(nextPage || 1);
                        if (nextPageSize && nextPageSize !== filesPageSize) {
                          setFilesPageSize(nextPageSize);
                          setFilesPage(1);
                        }
                      },
                      pageSizeChangeResetCurrent: true,
                    })}
                  />
                ) : null}
              </Space>
            </Tabs.TabPane>
            <Tabs.TabPane
              key="tasks"
              title={t('business.filedist.tasksTab', { count: tasksTotal })}
            >
              <Space direction="vertical" style={{ width: '100%' }}>
                {canCreate ? (
                  <Button type="primary" icon={<IconPlus />} onClick={() => void openCreate()}>
                    {t('business.filedist.createTask')}
                  </Button>
                ) : null}
                {tasksLoading && tasks.length === 0 ? <PageLoading /> : null}
                {tasksError && !tasksLoading && tasks.length === 0 ? (
                  <PageError
                    description={
                      isForbiddenRequestError(tasksError)
                        ? t('business.filedist.createForbidden')
                        : undefined
                    }
                    onRetry={() => void loadTasks()}
                  />
                ) : null}
                {tasksError && tasks.length > 0 ? (
                  <Alert
                    type="warning"
                    content={
                      isForbiddenRequestError(tasksError)
                        ? t('business.filedist.createForbidden')
                        : t('business.filedist.refreshFailed')
                    }
                  />
                ) : null}
                {!tasksLoading && !tasksError && tasks.length === 0 ? (
                  <PageEmpty description={t('business.filedist.noTasks')} />
                ) : null}
                {tasks.length > 0 ? (
                  <AppTable
                    rowKey="id"
                    data={tasks}
                    columns={taskColumns}
                    loading={tasksLoading}
                    scroll={{ x: 'max-content' }}
                    pagination={buildStandardPagination(t, {
                      current: tasksPage,
                      pageSize: tasksPageSize,
                      total: tasksTotal,
                      onChange: (nextPage, nextPageSize) => {
                        setTasksPage(nextPage || 1);
                        if (nextPageSize && nextPageSize !== tasksPageSize) {
                          setTasksPageSize(nextPageSize);
                          setTasksPage(1);
                        }
                      },
                      pageSizeChangeResetCurrent: true,
                    })}
                  />
                ) : null}
              </Space>
            </Tabs.TabPane>
          </Tabs>
        </Card>
      </Space>
      <Modal
        title={t('business.filedist.upload')}
        visible={uploadVisible}
        onCancel={() => {
          setUploadVisible(false);
          setSelectedFile(null);
        }}
        onOk={() => void submitUpload()}
        confirmLoading={uploading}
      >
        <Upload
          showUploadList={false}
          autoUpload={false}
          onChange={(fileList) => setSelectedFile(fileList[0]?.originFile || null)}
        >
          <Button icon={<IconUpload />}>{t('business.filedist.selectFile')}</Button>
        </Upload>
      </Modal>
      <AppModal
        size="xl"
        title={t('business.filedist.createTask')}
        visible={createVisible}
        footer={null}
        onCancel={closeCreate}
      >
        {createOptionsLoading ? <PageLoading /> : null}
        {createOptionsError ? (
          <PageError
            description={
              isForbiddenRequestError(createOptionsError)
                ? t('business.filedist.createForbidden')
                : t('business.filedist.create.optionsFailed')
            }
            onRetry={() => void openCreate()}
          />
        ) : null}
        <div inert={createSubmitting}>
          <Form form={createForm} layout="vertical" disabled={createSubmitting}>
            <Form.Item
              field="name"
              label={t('business.filedist.taskName')}
              rules={[{ required: true }]}
            >
              <Input maxLength={128} />
            </Form.Item>
            <Form.Item
              field="fileId"
              label={t('business.filedist.create.selectFile')}
              rules={[{ required: true }]}
            >
              <Select
                showSearch
                filterOption={false}
                loading={fileChoicesLoading}
                placeholder={t('business.filedist.create.selectFilePlaceholder')}
                onInputValueChange={(value, reason) => {
                  if (reason !== 'manual') return;
                  createGeneration.current += 1;
                  createForm.setFieldsValue({ fileId: undefined });
                  searchFileChoices(value);
                }}
              >
                {createFiles.map((file) => (
                  <Select.Option key={file.id} value={file.id}>
                    {file.fileName || file.name} · {formatSize(file.fileSize || file.size || 0)}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
            {fileChoicesTotal > 0 ? (
              <Pagination
                simple
                current={fileChoicesPage}
                total={fileChoicesTotal}
                pageSize={FILE_CHOICE_PAGE_SIZE}
                onChange={(page) => {
                  createGeneration.current += 1;
                  createForm.setFieldsValue({ fileId: undefined });
                  setFileChoicesPage(page);
                }}
              />
            ) : null}
            {fileChoicesError ? (
              <PageError
                description={
                  isForbiddenRequestError(fileChoicesError)
                    ? t('business.filedist.createForbidden')
                    : t('business.filedist.create.optionsFailed')
                }
                onRetry={reloadFileChoices}
              />
            ) : null}
            {!fileChoicesLoading &&
            !fileChoicesError &&
            !fileChoicesKeyword &&
            fileChoicesPage === 1 &&
            createFiles.length === 0 ? (
              <Alert type="warning" content={t('business.filedist.create.noFileFirst')} />
            ) : null}
            <Form.Item
              field="businessScopeId"
              label={t('business.deploy.task.businessScope')}
              rules={[{ required: true }]}
            >
              <Select
                placeholder={t('business.deploy.task.businessScopePlaceholder')}
                onChange={(value) => {
                  createGeneration.current += 1;
                  setSelectedScopeId(value);
                  setTargetResolution(null);
                }}
              >
                {scopeOptions.map((item) => (
                  <Select.Option key={item.id} value={item.id}>
                    {item.name}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
            {selectedScopeId ? (
              <Form.Item label={t('business.filedist.create.targets')} required>
                <DeployTargetSelector
                  key={selectedScopeId}
                  scopeId={selectedScopeId}
                  groups={groups}
                  onChange={(resolution) => {
                    createGeneration.current += 1;
                    setTargetResolution(resolution);
                  }}
                />
              </Form.Item>
            ) : null}
            {targetResolution && targetResolution.hosts.length > 0 ? (
              <>
                <Divider orientation="left">{t('business.filedist.create.targetConfig')}</Divider>
                <div className="filedist-target-config">
                  <Form.Item
                    field="username"
                    label={t('business.filedist.create.sshUsername')}
                    rules={[{ required: true }]}
                  >
                    <Input
                      autoComplete="off"
                      placeholder={t('business.filedist.create.usernamePlaceholder')}
                    />
                  </Form.Item>
                  <Form.Item
                    field="targetPath"
                    label={t('business.filedist.targetPath')}
                    rules={[{ required: true }]}
                  >
                    <Input placeholder={t('business.filedist.create.pathPlaceholder')} />
                  </Form.Item>
                  <Form.Item label={t('business.filedist.create.authMode')}>
                    <Select
                      value={authMode}
                      onChange={(mode: 'password' | 'privateKey') => {
                        createForm.setFieldsValue({ password: undefined, privateKey: undefined });
                        setAuthMode(mode);
                      }}
                    >
                      <Select.Option value="password">
                        {t('business.filedist.create.password')}
                      </Select.Option>
                      <Select.Option value="privateKey">
                        {t('business.filedist.create.privateKey')}
                      </Select.Option>
                    </Select>
                  </Form.Item>
                  {authMode === 'password' ? (
                    <Form.Item
                      field="password"
                      label={t('business.filedist.create.password')}
                      rules={[{ required: true }]}
                    >
                      <PasswordInput autoComplete="new-password" />
                    </Form.Item>
                  ) : (
                    <Form.Item
                      field="privateKey"
                      label={t('business.filedist.create.privateKey')}
                      rules={[{ required: true }]}
                    >
                      <SecretTextArea autoComplete="off" autoSize={{ minRows: 3, maxRows: 6 }} />
                    </Form.Item>
                  )}
                  <Form.Item
                    field="maxRetries"
                    label={t('business.filedist.create.maxRetries')}
                    initialValue={3}
                  >
                    <InputNumber min={0} max={20} style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item field="scheduledAt" label={t('business.filedist.create.scheduledAt')}>
                    <DatePicker
                      style={{ width: '100%' }}
                      showTime
                      format="YYYY-MM-DD HH:mm:ss"
                      onChange={setScheduledAtRaw}
                    />
                  </Form.Item>
                  {targetResolution.hosts.map((host) => (
                    <Form.Item
                      className="filedist-target-config__fingerprint"
                      key={host.id}
                      field={`hostFingerprints.${host.id}`}
                      label={t('business.filedist.create.hostFingerprintFor', {
                        host: `${host.hostname} · ${host.ip}`,
                      })}
                      rules={[
                        { required: true },
                        {
                          match: /^SHA256:[A-Za-z0-9+/]{43}$/,
                          message: t('business.filedist.create.invalidFingerprint'),
                        },
                      ]}
                    >
                      <Input
                        autoComplete="off"
                        placeholder={t('business.filedist.create.fingerprintPlaceholder')}
                      />
                    </Form.Item>
                  ))}
                </div>
                <Alert type="warning" content={t('business.filedist.create.endpointHint')} />
              </>
            ) : null}
            {createError ? (
              <Alert type="error" role="alert" content={createError} style={{ marginBottom: 16 }} />
            ) : null}
            <SubmitBar
              loading={createSubmitting}
              onCancel={closeCreate}
              onSubmit={() => void submitCreate()}
              submitDisabled={createSubmitDisabled}
              submitText={t('business.filedist.createTask')}
            />
          </Form>
        </div>
      </AppModal>
    </PageContainer>
  );
}
