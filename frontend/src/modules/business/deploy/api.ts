import { apiRequest, downloadFile, uploadImportFile } from '@duanxldragon/pantheon-base-ui';

export type DeployAction = 'install' | 'uninstall' | 'upgrade' | 'reinstall';

export type DeployFailurePolicy = 'fail_fast' | 'continue_hosts' | 'continue_steps' | (string & {});

export type DeployStepType =
  | 'package'
  | 'script'
  | 'nginx'
  | 'redis'
  | 'jdk'
  | 'mysql'
  | 'minio'
  | 'harbor'
  | 'prometheus';

export interface DeployPackageRow {
  id: number;
  name: string;
  version: string;
  description: string;
  installCommand: string;
  uninstallCommand: string;
  executionMode: 'fixed' | 'orchestrated';
  templateCode?: string;
  templateConfig?: Record<string, unknown>;
  sourceObjectKey?: string;
  sourceFileName?: string;
  sourceUrl?: string;
  status: string;
  latestDeployedAt?: string;
  latestTaskId?: number;
  latestTaskName?: string;
  latestTaskStatus?: string;
  latestHostCount?: number;
  latestSuccessCount?: number;
  createdAt: string;
  updatedAt: string;
}

export interface DeployPackageListResp {
  items: DeployPackageRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeployPackagePayload {
  name: string;
  version: string;
  description?: string;
  installCommand?: string;
  uninstallCommand?: string;
  executionMode?: 'fixed' | 'orchestrated';
  templateCode?: string;
  templateConfig?: Record<string, unknown>;
  sourceObjectKey?: string;
  sourceFileName?: string;
  sourceUrl?: string;
  status?: string;
}

export interface DeployTemplateStepRow {
  id: number;
  templateId: number;
  stepCode: string;
  stepName: string;
  stepType: DeployStepType;
  action: DeployAction;
  packageId: number;
  packageName: string;
  packageVersion: string;
  templateCode?: string;
  templateParams?: Record<string, unknown>;
  stepConfig?: Record<string, unknown>;
  dependencies?: string[];
  condition?: Record<string, unknown>;
  failurePolicy?: DeployFailurePolicy;
  continueOnHostFailure?: boolean;
  sort: number;
  createdAt: string;
  updatedAt: string;
}

export interface DeployTemplateRow {
  id: number;
  name: string;
  version: string;
  description: string;
  category: string;
  executionMode: 'fixed' | 'orchestrated';
  defaultAction: DeployAction;
  packageId: number;
  packageName: string;
  packageVersion: string;
  templateCode?: string;
  templateConfig?: Record<string, unknown>;
  parameterSchema?: Record<string, unknown>;
  status: string;
  stepCount: number;
  steps: DeployTemplateStepRow[];
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface DeployTemplatePayload {
  name: string;
  version: string;
  description?: string;
  category?: string;
  executionMode?: 'fixed' | 'orchestrated';
  defaultAction?: DeployAction;
  packageId?: number;
  templateCode?: string;
  templateConfig?: Record<string, unknown>;
  parameterSchema?: Record<string, unknown>;
  status?: string;
  steps?: Array<{
    stepCode: string;
    stepName: string;
    stepType: DeployStepType;
    action: DeployAction;
    packageId?: number;
    packageName?: string;
    packageVersion?: string;
    templateCode?: string;
    templateParams?: Record<string, unknown>;
    stepConfig?: Record<string, unknown>;
    dependencies?: string[];
    condition?: Record<string, unknown>;
    failurePolicy?: DeployFailurePolicy;
    continueOnHostFailure?: boolean;
    sort: number;
  }>;
}

export interface DeployTaskHostRow {
  id: number;
  taskId: number;
  hostId: number;
  hostname: string;
  hostIp: string;
  os: string;
  status: string;
  stdout: string;
  stderr: string;
  errorMessage: string;
  executorId: string;
  traceSteps?: Array<{
    at?: string;
    phase?: string;
    message?: string;
    stepCode?: string;
    stepName?: string;
    stepType?: DeployStepType;
    action?: string;
    packageName?: string;
  }>;
  startedAt?: string;
  finishedAt?: string;
  reportedAt?: string;
  durationSeconds?: number;
  updatedAt: string;
}

export interface DeployTaskRow {
  id: number;
  name: string;
  templateId: number;
  templateName: string;
  templateVersion: string;
  packageId: number;
  packageName: string;
  packageVersion: string;
  businessScopeId: number;
  businessScopeName: string;
  action: DeployAction;
  targetType: 'host' | 'group';
  targetIds: number[];
  executorType: 'manual' | 'simulated' | 'agent' | 'ssh';
  executionMode: 'fixed' | 'orchestrated';
  templateParams?: Record<string, unknown>;
  status: string;
  remark: string;
  externalTaskId: string;
  startedAt?: string;
  finishedAt?: string;
  hostCount: number;
  successCount: number;
  failedCount: number;
  runningCount: number;
  skippedCount: number;
  durationSeconds: number;
  createdAt: string;
  updatedAt: string;
  hosts: DeployTaskHostRow[];
}

export interface DeployTaskListResp {
  items: DeployTaskRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeployTaskPayload {
  name: string;
  templateId?: number;
  packageId?: number;
  businessScopeId?: number;
  action?: DeployAction;
  targetType: 'host' | 'group';
  targetIds: number[];
  executorType: 'manual' | 'simulated' | 'agent' | 'ssh';
  templateParams?: Record<string, unknown>;
  remark?: string;
}

export interface DeployCredentialRow {
  id: number;
  name: string;
  username: string;
  authMode: 'password' | 'private_key';
  version: number;
  status: string;
  createdAt: string;
  updatedAt: string;
}

// StartDeployTaskPayload is the CredentialRef-only Start contract. The SSH
// material is resolved server-side from the credential reference and must never
// be sent from the client. The idempotency key is generated per user submission
// and transported via the X-Idempotency-Key header.
export interface StartDeployTaskPayload {
  credentialRefId?: number;
  timeoutSeconds?: number;
  hostFingerprint?: string;
}

export interface MarkHostResultPayload {
  status: 'success' | 'failed' | 'skipped';
  stdout?: string;
  stderr?: string;
  errorMessage?: string;
  executorId?: string;
}

type ListQuery = Record<string, string | number | undefined>;

export function getDeployPackageList(params?: ListQuery) {
  return apiRequest<DeployPackageListResp>({
    url: '/business/deploy/packages',
    method: 'get',
    params,
  });
}

export function createDeployPackage(data: DeployPackagePayload) {
  return apiRequest<DeployPackageRow>({
    url: '/business/deploy/packages',
    method: 'post',
    data,
  });
}

export function getDeployPackageDetail(id: number) {
  return apiRequest<DeployPackageRow>({
    url: `/business/deploy/packages/${id}`,
    method: 'get',
  });
}

export function updateDeployPackage(id: number, data: Partial<DeployPackagePayload>) {
  return apiRequest<DeployPackageRow>({
    url: `/business/deploy/packages/${id}`,
    method: 'put',
    data,
  });
}

export function deleteDeployPackage(id: number) {
  return apiRequest<void>({
    url: `/business/deploy/packages/${id}`,
    method: 'delete',
  });
}

export function exportDeployPackages(params?: ListQuery) {
  return downloadFile({
    url: '/business/deploy/packages/export',
    method: 'get',
    params,
    filename: 'deploy-packages.csv',
  });
}

export function importDeployPackages(file: File) {
  return uploadImportFile('/business/deploy/packages/import', file);
}

export function getDeployTemplateList(params?: ListQuery) {
  return apiRequest<{ items: DeployTemplateRow[]; total: number; page: number; pageSize: number }>({
    url: '/business/deploy/templates',
    method: 'get',
    params,
  });
}

export function createDeployTemplate(data: DeployTemplatePayload) {
  return apiRequest<DeployTemplateRow>({
    url: '/business/deploy/templates',
    method: 'post',
    data,
  });
}

export function updateDeployTemplate(id: number, data: Partial<DeployTemplatePayload>) {
  return apiRequest<DeployTemplateRow>({
    url: `/business/deploy/templates/${id}`,
    method: 'put',
    data,
  });
}

export function deleteDeployTemplate(id: number) {
  return apiRequest<void>({
    url: `/business/deploy/templates/${id}`,
    method: 'delete',
  });
}

export function exportDeployTemplates(params?: ListQuery) {
  return downloadFile({
    url: '/business/deploy/templates/export',
    method: 'get',
    params,
    filename: 'deploy-templates.csv',
  });
}

export function importDeployTemplates(file: File) {
  return uploadImportFile('/business/deploy/templates/import', file);
}

export function getDeployTaskList(params?: ListQuery) {
  return apiRequest<DeployTaskListResp>({
    url: '/business/deploy/tasks',
    method: 'get',
    params,
  });
}

export function exportDeployTasks(params?: ListQuery) {
  return downloadFile({
    url: '/business/deploy/tasks/export',
    method: 'get',
    params,
    filename: 'deploy-tasks.csv',
  });
}

export function getDeployTaskDetail(id: number) {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}`,
    method: 'get',
    skipErrorMessage: true,
  });
}

export function getDeployTaskEventStreamUrl(id: number) {
  return `/api/v1/business/deploy/tasks/${id}/events`;
}

export function createDeployTask(data: DeployTaskPayload) {
  return apiRequest<DeployTaskRow>({
    url: '/business/deploy/tasks',
    method: 'post',
    data,
  });
}

export function updateDeployTask(
  id: number,
  data: Partial<DeployTaskPayload> & { templateId?: number; packageId?: number },
) {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}`,
    method: 'put',
    data,
  });
}

export function deleteDeployTask(id: number) {
  return apiRequest<void>({
    url: `/business/deploy/tasks/${id}`,
    method: 'delete',
  });
}

export function startDeployTask(id: number, data?: StartDeployTaskPayload) {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}/start`,
    method: 'post',
    data,
    headers: {
      'X-Idempotency-Key': crypto.randomUUID(),
    },
  });
}

export function getDeployCredentialList() {
  return apiRequest<DeployCredentialRow[]>({
    url: '/business/deploy/credentials',
    method: 'get',
  });
}

export function cancelDeployTask(id: number) {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}/cancel`,
    method: 'post',
    headers: { 'X-Idempotency-Key': crypto.randomUUID() },
  });
}

function controlDeployTask(id: number, action: 'pause' | 'resume' | 'retry') {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}/${action}`,
    method: 'post',
    headers: { 'X-Idempotency-Key': crypto.randomUUID() },
  });
}

export const pauseDeployTask = (id: number) => controlDeployTask(id, 'pause');
export const resumeDeployTask = (id: number) => controlDeployTask(id, 'resume');
export const retryDeployTask = (id: number) => controlDeployTask(id, 'retry');

export function markDeployHostResult(id: number, data: MarkHostResultPayload) {
  return apiRequest<DeployTaskHostRow>({
    url: `/business/deploy/task-hosts/${id}/result`,
    method: 'post',
    data,
  });
}
