import { apiRequest } from '../../../api/request';

// --- Cluster ---

export interface K8sClusterRow {
  id: number;
  code: string;
  name: string;
  environment: string;
  businessScopeId: number;
  businessScopeName: string;
  deptId: number;
  apiServer: string;
  version: string;
  status: string;
  totalNodes: number;
  readyNodes: number;
  totalPods: number;
  runningPods: number;
  cpuCapacity: number;
  cpuAllocatable: number;
  memoryCapacity: number;
  memoryAllocatable: number;
  lastSyncedAt: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface K8sClusterListQuery {
  keyword?: string;
  environment?: string;
  status?: string;
  businessScopeId?: number;
  page?: number;
  pageSize?: number;
}

export interface K8sClusterListResp {
  items: K8sClusterRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface K8sClusterPayload {
  code: string;
  name: string;
  environment: string;
  businessScopeId?: number;
  kubeconfig: string;
  remark?: string;
}

export interface K8sNodeRow {
  name: string;
  status: string;
  internalIp: string;
  os: string;
  kubeletVersion: string;
  cpuCapacity: string;
  memoryCapacity: string;
  podCapacity: number;
  allocatablePods: number;
}

export interface K8sNodeListResp {
  items: K8sNodeRow[];
  total: number;
}

export function getK8sClusterList(params?: K8sClusterListQuery) {
  return apiRequest<K8sClusterListResp>({
    url: '/business/k8s/clusters',
    method: 'get',
    params,
  });
}

export function getK8sClusterDetail(id: number) {
  return apiRequest<K8sClusterRow>({
    url: `/business/k8s/clusters/${id}`,
    method: 'get',
  });
}

export function createK8sCluster(data: K8sClusterPayload) {
  return apiRequest<K8sClusterRow>({
    url: '/business/k8s/clusters',
    method: 'post',
    data,
  });
}

export function updateK8sCluster(id: number, data: Partial<K8sClusterPayload>) {
  return apiRequest<K8sClusterRow>({
    url: `/business/k8s/clusters/${id}`,
    method: 'put',
    data,
  });
}

export function deleteK8sCluster(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/k8s/clusters/${id}`,
    method: 'delete',
  });
}

export function syncK8sCluster(id: number) {
  return apiRequest<K8sClusterRow>({
    url: `/business/k8s/clusters/${id}/sync`,
    method: 'post',
  });
}

export function getK8sClusterNodes(id: number) {
  return apiRequest<K8sNodeListResp>({
    url: `/business/k8s/clusters/${id}/nodes`,
    method: 'get',
  });
}

// --- Namespace ---

export interface K8sNamespaceRow {
  name: string;
  status: string;
  labels: Record<string, string>;
  creationTimestamp: string;
}

export interface K8sNamespaceListResp {
  items: K8sNamespaceRow[];
  total: number;
}

export function getK8sNamespaces(clusterId: number) {
  return apiRequest<K8sNamespaceListResp>({
    url: `/business/k8s/clusters/${clusterId}/namespaces`,
    method: 'get',
  });
}

export function createK8sNamespace(clusterId: number, data: { name: string; labels?: Record<string, string> }) {
  return apiRequest<K8sNamespaceRow>({
    url: `/business/k8s/clusters/${clusterId}/namespaces`,
    method: 'post',
    data,
  });
}

export function deleteK8sNamespace(clusterId: number, name: string) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/k8s/clusters/${clusterId}/namespaces/${name}`,
    method: 'delete',
  });
}

// --- Workload ---

export interface K8sWorkloadRow {
  kind: string;
  name: string;
  namespace: string;
  replicas: number;
  readyReplicas: number;
  images: string[];
  status: string;
  age: string;
}

export interface K8sWorkloadListResp {
  items: K8sWorkloadRow[];
  total: number;
}

export interface K8sWorkloadListQuery {
  clusterId: number;
  namespace?: string;
  kind?: string;
}

export interface K8sPodRow {
  name: string;
  status: string;
  nodeName: string;
  restarts: number;
  createdAt: string;
}

export interface K8sPodListResp {
  items: K8sPodRow[];
  total: number;
}

export function getK8sWorkloadList(params: K8sWorkloadListQuery) {
  return apiRequest<K8sWorkloadListResp>({
    url: '/business/k8s/workloads',
    method: 'get',
    params,
  });
}

export function getK8sWorkloadPods(clusterId: number, namespace: string, kind: string, name: string) {
  return apiRequest<K8sPodListResp>({
    url: `/business/k8s/clusters/${clusterId}/workloads/${namespace}/${kind}/${name}/pods`,
    method: 'get',
  });
}

export function scaleK8sWorkload(clusterId: number, namespace: string, kind: string, name: string, replicas: number) {
  return apiRequest<{ scaled: boolean }>({
    url: `/business/k8s/clusters/${clusterId}/workloads/${namespace}/${kind}/${name}/scale`,
    method: 'post',
    data: { replicas },
  });
}

export function restartK8sWorkload(clusterId: number, namespace: string, kind: string, name: string) {
  return apiRequest<{ restarted: boolean }>({
    url: `/business/k8s/clusters/${clusterId}/workloads/${namespace}/${kind}/${name}/restart`,
    method: 'post',
  });
}

// --- Release ---

export interface K8sReleaseRow {
  id: number;
  name: string;
  clusterId: number;
  namespace: string;
  workloadType: string;
  workloadName: string;
  containerName: string;
  imageBefore: string;
  imageAfter: string;
  strategy: string;
  status: string;
  idempotencyKey: string;
  previousReleaseId: number;
  attempt: number;
  targetGeneration: number;
  targetRevision: string;
  observedGeneration: number;
  observedRevision: string;
  desiredReplicas: number;
  updatedReplicas: number;
  availableReplicas: number;
  readyReplicas: number;
  conditionSummary: string;
  errorMessage: string;
  startedAt: string;
  finishedAt: string;
  lastReconciledAt: string;
  createdBy: string;
  createdAt: string;
}

export interface K8sReleaseListQuery {
  clusterId?: number;
  namespace?: string;
  workloadType?: string;
  workloadName?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

export interface K8sReleaseListResp {
  items: K8sReleaseRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface K8sReleasePayload {
  name: string;
  clusterId: number;
  namespace: string;
  workloadType: string;
  workloadName: string;
  containerName?: string;
  image: string;
  strategy?: string;
  idempotencyKey?: string;
}

export function getK8sReleaseList(params?: K8sReleaseListQuery) {
  return apiRequest<K8sReleaseListResp>({
    url: '/business/k8s/releases',
    method: 'get',
    params,
  });
}

export function createK8sRelease(data: K8sReleasePayload) {
  return apiRequest<K8sReleaseRow>({
    url: '/business/k8s/releases',
    method: 'post',
    data,
  });
}

export function rollbackK8sRelease(id: number, idempotencyKey?: string) {
  return apiRequest<K8sReleaseRow>({
    url: `/business/k8s/releases/${id}/rollback`,
    method: 'post',
    data: { idempotencyKey },
  });
}

export function reconcileK8sRelease(id: number) {
  return apiRequest<K8sReleaseRow>({
    url: `/business/k8s/releases/${id}/reconcile`,
    method: 'post',
  });
}

// --- ConfigMap ---

export interface K8sConfigMapRow {
  name: string;
  namespace: string;
  keyCount: number;
}

export interface K8sConfigMapListResp {
  items: K8sConfigMapRow[];
  total: number;
}

export function getK8sConfigMaps(clusterId: number, namespace?: string) {
  return apiRequest<K8sConfigMapListResp>({
    url: `/business/k8s/clusters/${clusterId}/configmaps`,
    method: 'get',
    params: namespace ? { namespace } : undefined,
  });
}

export function createK8sConfigMap(clusterId: number, namespace: string, data: { name: string; data: Record<string, string> }) {
  return apiRequest<K8sConfigMapRow>({
    url: `/business/k8s/clusters/${clusterId}/configmaps`,
    method: 'post',
    params: { namespace },
    data,
  });
}

export function deleteK8sConfigMap(clusterId: number, namespace: string, name: string) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/k8s/clusters/${clusterId}/configmaps/${name}`,
    method: 'delete',
    params: { namespace },
  });
}

// --- Secret ---

export interface K8sSecretRow {
  name: string;
  namespace: string;
  type: string;
  keyCount: number;
}

export interface K8sSecretListResp {
  items: K8sSecretRow[];
  total: number;
}

export function getK8sSecrets(clusterId: number, namespace?: string) {
  return apiRequest<K8sSecretListResp>({
    url: `/business/k8s/clusters/${clusterId}/secrets`,
    method: 'get',
    params: namespace ? { namespace } : undefined,
  });
}

export function createK8sSecret(clusterId: number, namespace: string, data: { name: string; type: string; data: Record<string, string> }) {
  return apiRequest<K8sSecretRow>({
    url: `/business/k8s/clusters/${clusterId}/secrets`,
    method: 'post',
    params: { namespace },
    data,
  });
}

export function deleteK8sSecret(clusterId: number, namespace: string, name: string) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/k8s/clusters/${clusterId}/secrets/${name}`,
    method: 'delete',
    params: { namespace },
  });
}
