import { apiRequest } from '../../../api/request';
import { downloadCsvFile, downloadFile } from '../../../api/file';

export interface OperationLogQuery {
  title?: string;
  operName?: string;
  status?: number;
  businessType?: number;
  sourceDomain?: string;
  sourcePage?: string;
  failureCategory?: string;
  page: number;
  pageSize: number;
}

export interface OperationLogRow {
  id: number;
  title: string;
  businessType: number;
  method: string;
  operName: string;
  operUrl: string;
  operIp: string;
  sourceDomain: string;
  sourcePage: string;
  operParam: string;
  jsonResult: string;
  status: number;
  failureCategory: string;
  errorMsg: string;
  operTime: string;
  costTime: number;
}

export interface OperationLogPageResp {
  items: OperationLogRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface OperationLogCleanupPayload {
  retentionDays?: number;
  startedAt?: string;
  endedAt?: string;
}

export interface OperationLogBatchDeletePayload {
  ids: number[];
}

export function getOperationLogList(params: OperationLogQuery) {
  return apiRequest<OperationLogPageResp>({
    url: '/system/operation-log/list',
    method: 'get',
    params,
  });
}

export function getOperationLog(id: number) {
  return apiRequest<OperationLogRow>({
    url: `/system/operation-log/${id}`,
    method: 'get',
  });
}

export function deleteOperationLog(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/operation-log/${id}`,
    method: 'delete',
  });
}

export function cleanupOperationLogs(data: OperationLogCleanupPayload) {
  return apiRequest<{ clearedCount: number }>({
    url: '/system/operation-log/cleanup',
    method: 'post',
    data,
  });
}

export function batchDeleteOperationLogs(data: OperationLogBatchDeletePayload) {
  return apiRequest<{ deletedCount: number }>({
    url: '/system/operation-log/batch-delete',
    method: 'post',
    data,
  });
}

export function exportOperationLogs(data: Partial<OperationLogQuery>) {
  return downloadFile({
    url: '/system/operation-log/export',
    method: 'post',
    data,
    filename: 'system-operation-log-export.csv',
  });
}

export function exportSelectedOperationLogs(rows: OperationLogRow[]) {
  downloadCsvFile(
    'system-operation-log-export.csv',
    [
      'requestId',
      'title',
      'businessType',
      'sourceDomain',
      'sourcePage',
      'method',
      'operName',
      'operUrl',
      'operIp',
      'status',
      'failureCategory',
      'errorMsg',
      'operTime',
      'costTime',
    ],
    rows.map((row) => [
      '',
      row.title || '',
      String(row.businessType ?? ''),
      row.sourceDomain || '',
      row.sourcePage || '',
      row.method || '',
      row.operName || '',
      row.operUrl || '',
      row.operIp || '',
      String(row.status ?? ''),
      row.failureCategory || '',
      row.errorMsg || '',
      row.operTime || '',
      String(row.costTime ?? ''),
    ]),
  );
}
