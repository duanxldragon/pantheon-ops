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

// --- Service ---

export interface K8sServicePort {
  name?: string;
  port: number;
  targetPort: number;
  nodePort?: number;
  protocol: string;
}

export interface K8sServiceRow {
  name: string;
  namespace: string;
  type: string;
  clusterIP: string;
  ports: K8sServicePort[];
  selector: Record<string, string>;
  loadBalancerIP?: string;
}

export function getK8sServiceList(clusterId: number, params?: { namespace?: string; name?: string }) {
  return apiRequest<{ items: K8sServiceRow[]; total: number }>({
    url: `/business/k8s/clusters/${clusterId}/services`,
    method: 'get',
    params,
  });
}

export function createK8sService(clusterId: number, namespace: string, data: { name: string; type: string; ports: K8sServicePort[]; selector: Record<string, string> }) {
  return apiRequest<K8sServiceRow>({
    url: `/business/k8s/clusters/${clusterId}/services`,
    method: 'post',
    params: { namespace },
    data,
  });
}

export function deleteK8sService(clusterId: number, namespace: string, name: string) {
  return apiRequest<void>({
    url: `/business/k8s/clusters/${clusterId}/services/${encodeURIComponent(name)}`,
    method: 'delete',
    params: { namespace },
  });
}

// --- Ingress ---

export interface K8sIngressRulePath {
  path: string;
  pathType: string;
  serviceName: string;
  servicePort: number;
}

export interface K8sIngressRule {
  host: string;
  paths: K8sIngressRulePath[];
}

export interface K8sIngressTLS {
  hosts: string[];
  secretName: string;
}

export interface K8sIngressRow {
  id: number;
  name: string;
  namespace: string;
  clusterId: number;
  clusterName?: string;
  ingressClass?: string;
  rules: K8sIngressRule[];
  tls?: K8sIngressTLS[];
  annotations?: Record<string, string>;
  status: string;
  loadBalancerIp?: string;
  hosts: string[];
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

export interface K8sIngressListQuery {
  clusterId: number;
  namespace?: string;
  name?: string;
  host?: string;
}

export interface K8sIngressListResp {
  items: K8sIngressRow[];
  total: number;
}

export function getK8sIngressList(params: K8sIngressListQuery) {
  return apiRequest<K8sIngressListResp>({
    url: '/business/k8s/ingresses',
    method: 'get',
    params,
  });
}

export function syncK8sIngress(id: number) {
  return apiRequest<{ synced: boolean }>({
    url: `/business/k8s/ingresses/${id}/sync`,
    method: 'post',
  });
}

export function deleteK8sIngress(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/k8s/ingresses/${id}`,
    method: 'delete',
  });
}

// --- Storage (PV/PVC/StorageClass) ---

export interface K8sPVRow {
  id: number;
  name: string;
  clusterId: number;
  clusterName?: string;
  capacity: string;
  storageClass?: string;
  accessModes: string[];
  reclaimPolicy: string;
  status: string;
  claimRef?: string;
  volumeMode?: string;
  createdAt: string;
  updatedAt: string;
}

export interface K8sPVListQuery {
  clusterId: number;
  storageClass?: string;
  status?: string;
  name?: string;
}

export interface K8sPVCRow {
  id: number;
  name: string;
  namespace: string;
  clusterId: number;
  clusterName?: string;
  pvName?: string;
  storageClass?: string;
  requestStorage: string;
  accessModes: string[];
  volumeMode?: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface K8sPVCListQuery {
  clusterId: number;
  namespace?: string;
  storageClass?: string;
  status?: string;
  name?: string;
}

export interface K8sStorageClassRow {
  id: number;
  name: string;
  clusterId: number;
  clusterName?: string;
  provisioner: string;
  reclaimPolicy: string;
  volumeBindingMode: string;
  allowVolumeExpansion: boolean;
  isDefault: boolean;
}

export interface K8sStorageClassListQuery {
  clusterId: number;
  name?: string;
}

export interface K8sStorageListResp<T> {
  items: T[];
  total: number;
}

export function getK8sPVList(params: K8sPVListQuery) {
  return apiRequest<K8sStorageListResp<K8sPVRow>>({
    url: '/business/k8s/storage/pvs',
    method: 'get',
    params,
  });
}

export function syncK8sPVs(clusterId: number) {
  return apiRequest<{ synced: number }>({
    url: '/business/k8s/storage/pvs/sync',
    method: 'post',
    data: { clusterId },
  });
}

export function deleteK8sPV(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/k8s/storage/pvs/${id}`,
    method: 'delete',
  });
}

export function getK8sPVCList(params: K8sPVCListQuery) {
  return apiRequest<K8sStorageListResp<K8sPVCRow>>({
    url: '/business/k8s/storage/pvcs',
    method: 'get',
    params,
  });
}

export function syncK8sPVCs(clusterId: number) {
  return apiRequest<{ synced: number }>({
    url: '/business/k8s/storage/pvcs/sync',
    method: 'post',
    data: { clusterId },
  });
}

export function deleteK8sPVC(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/k8s/storage/pvcs/${id}`,
    method: 'delete',
  });
}

export function getK8sStorageClassList(params: K8sStorageClassListQuery) {
  return apiRequest<K8sStorageListResp<K8sStorageClassRow>>({
    url: '/business/k8s/storage/storage-classes',
    method: 'get',
    params,
  });
}

export function syncK8sStorageClasses(clusterId: number) {
  return apiRequest<{ synced: number }>({
    url: '/business/k8s/storage/storage-classes/sync',
    method: 'post',
    data: { clusterId },
  });
}

// --- Velero Backup ---

export interface K8sBackupRow {
  id: number;
  name: string;
  clusterId: number;
  backupType: string;
  includedNamespaces?: string[];
  excludedNamespaces?: string[];
  labelSelector?: string;
  includePvs: boolean;
  ttl: number;
  status: string;
  phase?: string;
  startTime?: string;
  completionTime?: string;
  expiration?: string;
  errors: number;
  warnings: number;
  backupSize?: number;
  storageLocation?: string;
  volumeSnapshotsCount: number;
  createdAt: string;
  createdBy: string;
}

export interface K8sBackupListResp {
  items: K8sBackupRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface K8sScheduleRow {
  id: number;
  name: string;
  clusterId: number;
  cronExpression: string;
  enabled: boolean;
  backupTemplate?: Record<string, unknown>;
  lastBackupTime?: string;
  nextBackupTime?: string;
  lastBackupStatus?: string;
  phase?: string;
  createdAt: string;
  createdBy: string;
}

export interface K8sScheduleListResp {
  items: K8sScheduleRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface K8sRestoreRow {
  id: number;
  name: string;
  clusterId: number;
  backupName: string;
  restorePvs: boolean;
  status: string;
  phase?: string;
  startTime?: string;
  completionTime?: string;
  errors: number;
  warnings: number;
  createdAt: string;
  createdBy: string;
}

export interface K8sRestoreListResp {
  items: K8sRestoreRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface K8sStorageLocationRow {
  id: number;
  name: string;
  provider: string;
  bucket: string;
  region?: string;
  endpoint?: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface K8sVeleroStatus {
  installed: boolean;
  version?: string;
  namespace?: string;
  runningPods: number;
  storageLocations: number;
}

export function getK8sBackupList(params: { clusterId: number; status?: string; backupType?: string; page?: number; pageSize?: number }) {
  return apiRequest<K8sBackupListResp>({ url: '/business/k8s/backup/backups', method: 'get', params });
}

export function createK8sBackup(data: { name: string; clusterId: number; backupType: string; includedNamespaces?: string[]; labelSelector?: string; includePvs: boolean; ttl: number; storageLocation?: string }) {
  return apiRequest<K8sBackupRow>({ url: '/business/k8s/backup/backups', method: 'post', data });
}

export function deleteK8sBackup(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/k8s/backup/backups/${id}`, method: 'delete' });
}

export function getK8sScheduleList(params: { clusterId: number; page?: number; pageSize?: number }) {
  return apiRequest<K8sScheduleListResp>({ url: '/business/k8s/backup/schedules', method: 'get', params });
}

export function createK8sSchedule(data: { name: string; clusterId: number; cronExpression: string; enabled: boolean; backupTemplate: Record<string, unknown> }) {
  return apiRequest<K8sScheduleRow>({ url: '/business/k8s/backup/schedules', method: 'post', data });
}

export function deleteK8sSchedule(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/k8s/backup/schedules/${id}`, method: 'delete' });
}

export function pauseK8sSchedule(id: number) {
  return apiRequest<{ paused: boolean }>({ url: `/business/k8s/backup/schedules/${id}/pause`, method: 'post' });
}

export function resumeK8sSchedule(id: number) {
  return apiRequest<{ resumed: boolean }>({ url: `/business/k8s/backup/schedules/${id}/resume`, method: 'post' });
}

export function getK8sRestoreList(params: { clusterId: number; status?: string; page?: number; pageSize?: number }) {
  return apiRequest<K8sRestoreListResp>({ url: '/business/k8s/backup/restores', method: 'get', params });
}

export function createK8sRestore(data: { name: string; clusterId: number; backupName: string; restorePvs: boolean }) {
  return apiRequest<K8sRestoreRow>({ url: '/business/k8s/backup/restores', method: 'post', data });
}

export function deleteK8sRestore(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/k8s/backup/restores/${id}`, method: 'delete' });
}

export function getK8sVeleroStatus(clusterId: number) {
  return apiRequest<K8sVeleroStatus>({ url: '/business/k8s/backup/velero/status', method: 'get', params: { clusterId } });
}

export function getK8sStorageLocationList() {
  return apiRequest<K8sStorageLocationRow[]>({ url: '/business/k8s/backup/locations', method: 'get' });
}

export function createK8sStorageLocation(data: { name: string; provider: string; bucket: string; region?: string; endpoint?: string; accessKeyId?: string; secretAccessKey?: string; isDefault: boolean }) {
  return apiRequest<K8sStorageLocationRow>({ url: '/business/k8s/backup/locations', method: 'post', data });
}

export function deleteK8sStorageLocation(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/k8s/backup/locations/${id}`, method: 'delete' });
}
