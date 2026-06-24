import { apiRequest } from '../../../../api/request';
import { downloadFile } from '../../../../api/file';
import { uploadImportFile } from '../../../../api/importExport';

export interface DictTypeRow {
  id: number;
  dictCode: string;
  dictName: string;
  module: string;
  status: number;
  remark: string;
  itemCount: number;
  activeItemCount: number;
  disabledItemCount: number;
  lastItemUpdatedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface DictTypeQuery {
  dictCode?: string;
  dictName?: string;
  status?: number;
}

export interface DictTypePayload {
  dictCode: string;
  dictName: string;
  module: string;
  status: number;
  remark: string;
}

export interface DictItemRow {
  id: number;
  dictCode: string;
  itemLabelKey: string;
  itemValue: string;
  itemColor: string;
  sort: number;
  status: number;
  remark: string;
  createdAt: string;
  updatedAt: string;
}

export interface DictItemPageResp {
  items: DictItemRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DictItemQuery {
  dictCode: string;
  keyword?: string;
  status?: number;
  page?: number;
  pageSize?: number;
}

export interface DictItemPayload {
  dictCode: string;
  itemLabelKey: string;
  itemValue: string;
  itemColor: string;
  sort: number;
  status: number;
  remark: string;
}

export interface DictCacheRefreshPayload {
  codes?: string[];
}

export interface DictCacheRefreshResp {
  refreshedCodes: string[];
  clearedAll: number;
}

export interface DictTypeBatchStatusPayload {
  typeIds: number[];
  status: number;
}

export interface DictItemBatchStatusPayload {
  itemIds: number[];
  status: number;
}

export interface DictBatchStatusResp {
  updatedCount: number;
}

export interface BatchDeletePayload {
  ids: number[];
}

export interface BatchDeleteResp {
  deletedCount: number;
  failedCount: number;
  failures: Array<{ id: number; reason: string }>;
}

export interface DictUsageReference {
  filePath: string;
  line: number;
  column: number;
  snippet: string;
  domain: string;
  moduleHint: string;
}

export interface DictUsageAnalysisResp {
  dictCode: string;
  referenceCount: number;
  scannedProjectRoot: string;
  references: DictUsageReference[];
}

function normalizeDictTypeRow(row: Partial<DictTypeRow>): DictTypeRow {
  return {
    id: Number(row.id || 0),
    dictCode: row.dictCode || '',
    dictName: row.dictName || '',
    module: row.module || 'system',
    status: Number(row.status || 1),
    remark: row.remark || '',
    itemCount: Number(row.itemCount || 0),
    activeItemCount: Number(row.activeItemCount || 0),
    disabledItemCount: Number(row.disabledItemCount || 0),
    lastItemUpdatedAt: row.lastItemUpdatedAt || '',
    createdAt: row.createdAt || '',
    updatedAt: row.updatedAt || '',
  };
}

function normalizeDictItemRow(row: Partial<DictItemRow>): DictItemRow {
  return {
    id: Number(row.id || 0),
    dictCode: row.dictCode || '',
    itemLabelKey: row.itemLabelKey || '',
    itemValue: row.itemValue || '',
    itemColor: row.itemColor || '',
    sort: Number(row.sort || 0),
    status: Number(row.status || 1),
    remark: row.remark || '',
    createdAt: row.createdAt || '',
    updatedAt: row.updatedAt || '',
  };
}

export async function getDictTypeList(params?: DictTypeQuery) {
  const resp = await apiRequest<DictTypeRow[] | Partial<DictTypeRow>[]>({
    url: '/system/dict/type/list',
    method: 'get',
    params,
  });
  return Array.isArray(resp) ? resp.map(normalizeDictTypeRow) : [];
}

export function createDictType(data: DictTypePayload) {
  return apiRequest<DictTypeRow>({
    url: '/system/dict/type',
    method: 'post',
    data,
  });
}

export function updateDictType(id: number, data: DictTypePayload) {
  return apiRequest<DictTypeRow>({
    url: `/system/dict/type/${id}`,
    method: 'put',
    data,
  });
}

export function batchUpdateDictTypeStatus(data: DictTypeBatchStatusPayload) {
  return apiRequest<DictBatchStatusResp>({
    url: '/system/dict/type/batch-status',
    method: 'post',
    data,
  });
}

export function deleteDictType(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/dict/type/${id}`,
    method: 'delete',
  });
}

export function batchDeleteDictTypes(data: BatchDeletePayload) {
  return apiRequest<BatchDeleteResp>({
    url: '/system/dict/type/batch-delete',
    method: 'post',
    data,
  });
}

export async function getDictItemList(params: DictItemQuery) {
  const resp = await apiRequest<DictItemPageResp | Partial<DictItemRow>[]>({
    url: '/system/dict/item/list',
    method: 'get',
    params,
  });
  if (Array.isArray(resp)) {
    const items = resp.map(normalizeDictItemRow);
    return {
      items,
      total: items.length,
      page: params.page || 1,
      pageSize: params.pageSize || items.length || 10,
    };
  }
  return {
    items: Array.isArray(resp.items) ? resp.items.map(normalizeDictItemRow) : [],
    total: Number(resp.total || 0),
    page: Number(resp.page || params.page || 1),
    pageSize: Number(resp.pageSize || params.pageSize || 10),
  };
}

export function createDictItem(data: DictItemPayload) {
  return apiRequest<DictItemRow>({
    url: '/system/dict/item',
    method: 'post',
    data,
  });
}

export function updateDictItem(id: number, data: DictItemPayload) {
  return apiRequest<DictItemRow>({
    url: `/system/dict/item/${id}`,
    method: 'put',
    data,
  });
}

export function batchUpdateDictItemStatus(data: DictItemBatchStatusPayload) {
  return apiRequest<DictBatchStatusResp>({
    url: '/system/dict/item/batch-status',
    method: 'post',
    data,
  });
}

export function reorderDictItem(id: number, direction: 'up' | 'down') {
  return apiRequest<DictItemRow>({
    url: `/system/dict/item/${id}/reorder`,
    method: 'put',
    data: { direction },
  });
}

export function analyzeDictUsage(dictCode: string) {
  return apiRequest<DictUsageAnalysisResp>({
    url: '/system/dict/usage',
    method: 'get',
    params: { dictCode },
  });
}

export function deleteDictItem(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/dict/item/${id}`,
    method: 'delete',
  });
}

export function batchDeleteDictItems(data: BatchDeletePayload) {
  return apiRequest<BatchDeleteResp>({
    url: '/system/dict/item/batch-delete',
    method: 'post',
    data,
  });
}

export function refreshDictCache(data: DictCacheRefreshPayload) {
  return apiRequest<DictCacheRefreshResp>({
    url: '/system/dict/cache/refresh',
    method: 'post',
    data,
  });
}

export function exportDictTypes(data?: DictTypeQuery) {
  return downloadFile({
    url: '/system/dict/type/export',
    method: 'post',
    data,
    filename: 'system-dict-type-export.csv',
  });
}

export function downloadDictTypeImportTemplate() {
  return downloadFile({
    url: '/system/dict/type/import-template',
    method: 'get',
    filename: 'system-dict-type-import-template.csv',
  });
}

export function importDictTypes(file: File) {
  return uploadImportFile('/system/dict/type/import', file);
}

export function exportDictItems(data: DictItemQuery) {
  return downloadFile({
    url: '/system/dict/item/export',
    method: 'post',
    data,
    filename: 'system-dict-item-export.csv',
  });
}

export function downloadDictItemImportTemplate() {
  return downloadFile({
    url: '/system/dict/item/import-template',
    method: 'get',
    filename: 'system-dict-item-import-template.csv',
  });
}

export function importDictItems(file: File) {
  return uploadImportFile('/system/dict/item/import', file);
}
