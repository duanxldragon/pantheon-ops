import { apiRequest } from '../../../api/request';
import { downloadFile } from '../../../api/file';
import { uploadImportFile } from '../../../api/importExport';

export interface SystemI18n {
  id: string;
  module: string;
  group: string;
  key: string;
  locale: string;
  value: string;
  remark: string;
  createdAt: string;
  updatedAt: string;
}

export interface I18nQuery {
  module?: string;
  group?: string;
  locale?: string;
  key?: string;
  sortBy?: string;
  sortOrder?: string;
  page?: number;
  pageSize?: number;
}

export interface I18nPageResp {
  items: SystemI18n[];
  total: number;
  page: number;
  pageSize: number;
}

export interface I18nUpdateReq {
  value: string;
  remark?: string;
}

export interface I18nCreateReq {
  module: string;
  group?: string;
  key: string;
  locale: string;
  value: string;
  remark?: string;
}

export interface I18nSyncResp {
  count: number;
  keys: string[];
}

export interface I18nLocaleCoverage {
  locale: string;
  entryCount: number;
  missingCount: number;
}

export interface I18nOverviewResp {
  locales: string[];
  moduleCount: number;
  groupCount: number;
  totalEntries: number;
  uniqueKeyCount: number;
  missingValueCount: number;
  missingLocaleCount: number;
  coverage: I18nLocaleCoverage[];
}

export interface I18nMissingLocaleItem {
  module: string;
  group: string;
  key: string;
  missingLocales: string[];
}

export interface I18nMissingLocaleResp {
  items: I18nMissingLocaleItem[];
  total: number;
}

export interface I18nFillMissingLocaleResp {
  created: number;
  locales: string[];
  keys: string[];
}

export interface I18nHydrateBuiltinResp {
  created: number;
  updated: number;
  locales: string[];
  keys: string[];
}

export interface I18nDuplicateKeyConflict {
  key: string;
  modules: string[];
  groups: string[];
  locales: string[];
  values: string[];
  rowCount: number;
  suggestions: I18nRenameSuggestion[];
}

export interface I18nUnusedKeyItem {
  key: string;
  module: string;
  modules: string[];
  groups: string[];
  locales: string[];
  placeholder: boolean;
  lifecycleStatus: string;
  lifecycleMarkedAt?: string;
  observingDays: number;
  eligibleForArchive: boolean;
  eligibleForDelete: boolean;
}

export interface I18nModuleAuditItem {
  module: string;
  entryCount: number;
  keyCount: number;
  unusedKeyCount: number;
  duplicateKeyCount: number;
  missingLocaleCount: number;
  placeholderCount: number;
  stalePlaceholderCount: number;
  observingKeyCount: number;
  archivedKeyCount: number;
}

export interface I18nRenameSuggestion {
  module: string;
  suggestedKey: string;
}

export interface I18nStalePlaceholderItem {
  id: string | number;
  module: string;
  group: string;
  key: string;
  locale: string;
  value: string;
  updatedAt: string;
  staleDays: number;
}

export interface I18nAuditResp {
  duplicateKeys: I18nDuplicateKeyConflict[];
  unusedKeys: I18nUnusedKeyItem[];
  stalePlaceholders: I18nStalePlaceholderItem[];
  modules: I18nModuleAuditItem[];
  stalePlaceholderThresholdDays: number;
  unusedObservationThresholdDays: number;
}

export interface I18nCleanupUnusedResp {
  deleted: number;
  keys: string[];
  module: string;
}

export interface I18nUnusedLifecycleResp {
  module: string;
  affectedKeys: string[];
  affectedRows: number;
}

export interface I18nKeyReferenceFile {
  path: string;
  matchCount: number;
  suggestedReplacement: string;
  matches: I18nKeyReferenceMatch[];
}

export interface I18nKeyReferenceMatch {
  line: number;
  column: number;
  snippet: string;
  replacementHint: string;
}

export interface I18nRenamePreviewReq {
  module: string;
  oldKey: string;
  newKey: string;
}

export interface I18nRenamePreviewResp {
  module: string;
  oldKey: string;
  newKey: string;
  affectedRows: number;
  affectedLocales: string[];
  existingTargetRows: number;
  existingTargetLocales: string[];
  referenceFiles: I18nKeyReferenceFile[];
  requiresCodeMigration: boolean;
  canExecute: boolean;
}

export interface I18nRenameExecuteReq extends I18nRenamePreviewReq {
  confirmSourceUpdated: boolean;
}

export interface I18nRenameExecuteResp {
  module: string;
  oldKey: string;
  newKey: string;
  renamedRows: number;
  renamedLocales: string[];
}

export function getLangPack(locale: string) {
  return apiRequest<Record<string, string>>({
    url: '/system/i18n/pack',
    method: 'get',
    params: { locale },
  });
}

export function getI18nList(params: I18nQuery) {
  return apiRequest<I18nPageResp>({
    url: '/system/i18n/list',
    method: 'get',
    params,
  });
}

export function getI18nOverview() {
  return apiRequest<I18nOverviewResp>({
    url: '/system/i18n/overview',
    method: 'get',
  });
}

export function getI18nAudit() {
  return apiRequest<I18nAuditResp>({
    url: '/system/i18n/audit',
    method: 'get',
  });
}

export function getI18nMissingLocales(module?: string) {
  return apiRequest<I18nMissingLocaleResp>({
    url: '/system/i18n/missing-locales',
    method: 'get',
    params: module ? { module } : undefined,
  });
}

export function fillI18nMissingLocales(module?: string) {
  return apiRequest<I18nFillMissingLocaleResp>({
    url: '/system/i18n/fill-missing-locales',
    method: 'post',
    params: module ? { module } : undefined,
  });
}

export function hydrateBuiltinI18nLocales(module?: string) {
  return apiRequest<I18nHydrateBuiltinResp>({
    url: '/system/i18n/hydrate-builtin-locales',
    method: 'post',
    params: module ? { module } : undefined,
  });
}

export function cleanupUnusedI18nKeys(module?: string) {
  return apiRequest<I18nCleanupUnusedResp>({
    url: '/system/i18n/cleanup-unused',
    method: 'post',
    params: module ? { module } : undefined,
  });
}

export function startUnusedObservation(module?: string) {
  return apiRequest<I18nUnusedLifecycleResp>({
    url: '/system/i18n/lifecycle/observe',
    method: 'post',
    params: module ? { module } : undefined,
  });
}

export function archiveObservedUnusedKeys(module?: string) {
  return apiRequest<I18nUnusedLifecycleResp>({
    url: '/system/i18n/lifecycle/archive',
    method: 'post',
    params: module ? { module } : undefined,
  });
}

export function deleteArchivedUnusedKeys(module?: string, confirmArchived = false) {
  return apiRequest<I18nUnusedLifecycleResp>({
    url: '/system/i18n/lifecycle/delete',
    method: 'post',
    data: { module, confirmArchived },
  });
}

export function previewI18nRename(data: I18nRenamePreviewReq) {
  return apiRequest<I18nRenamePreviewResp>({
    url: '/system/i18n/rename/preview',
    method: 'post',
    data,
  });
}

export function renameI18nKey(data: I18nRenameExecuteReq) {
  return apiRequest<I18nRenameExecuteResp>({
    url: '/system/i18n/rename',
    method: 'post',
    data,
  });
}

export function getI18nDetail(id: string) {
  return apiRequest<SystemI18n>({
    url: `/system/i18n/${id}`,
    method: 'get',
  });
}

export function createI18n(data: I18nCreateReq) {
  return apiRequest<SystemI18n>({
    url: '/system/i18n',
    method: 'post',
    data,
  });
}

export function updateI18n(id: string, data: I18nUpdateReq) {
  return apiRequest({
    url: `/system/i18n/${id}`,
    method: 'put',
    data,
  });
}

export function deleteI18n(id: string) {
  return apiRequest({
    url: `/system/i18n/${id}`,
    method: 'delete',
  });
}

export function batchDeleteI18n(ids: string[]) {
  return apiRequest<{ deleted: boolean; count: number }>({
    url: '/system/i18n/batch-delete',
    method: 'post',
    data: { ids: ids.map((id) => Number(id)) },
  });
}

export function syncI18nKeys() {
  return apiRequest<I18nSyncResp>({
    url: '/system/i18n/sync-keys',
    method: 'post',
  });
}

export function refreshI18nCache() {
  return apiRequest({
    url: '/system/i18n/cache/refresh',
    method: 'post',
  });
}

export function refreshI18nLocales(locales: string[]) {
  return apiRequest({
    url: '/system/i18n/cache/refresh',
    method: 'post',
    data: { locales },
  });
}

export function exportI18n(data?: I18nQuery) {
  return downloadFile({
    url: '/system/i18n/export',
    method: 'post',
    data,
    filename: 'system-i18n-export.csv',
  });
}

export function downloadI18nImportTemplate() {
  return downloadFile({
    url: '/system/i18n/import-template',
    method: 'get',
    filename: 'system-i18n-import-template.csv',
  });
}

export function importI18n(file: File) {
  return uploadImportFile('/system/i18n/import', file);
}
