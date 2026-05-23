/**
 * 动态模块管理 - API接口
 */

import { apiRequest } from '../../../api/request';
import type { ModuleSchema } from '../generator/schema';

export interface ModuleRegistration {
  id: number;
  name: string;
  displayName: string;
  scope: string;
  source: string;
  owner?: string;
  boundedContext?: string;
  summary?: string;
  sourceTable?: string;
  autoRecycle?: boolean;
  tableName: string;
  status: number; // 1:已接入, 2:已卸载, 3:待激活, 4:失败
  installedAt: string;
  uninstalledAt?: string;
  lastVerifiedAt?: string;
  lastError?: string;
  lastVerificationResult?: string;
  builtIn?: boolean;
}

export interface RegistryRepairSummary {
  checkedModules: number;
  generatedRegistryRefs: number;
  markedUninstalledModules: number;
  artifactReadyModules: number;
  preservedUninstalledCount: number;
}

export interface ActivationAuditSummary {
  checkedModules: number;
  activatedModules: number;
  pendingModules: number;
  runtimeReadyModules: number;
  frontendReadyModules: number;
}

export interface RegisterModulePayload {
  name: string;
}

/**
 * 获取已注册模块列表
 */
export function getRegisteredModules() {
  return apiRequest<ModuleRegistration[]>({
    url: '/system/dynamic-modules',
    method: 'get',
  });
}

/**
 * 注册新模块
 */
export function registerModule(data: RegisterModulePayload) {
  return apiRequest<{ registered: boolean; message: string; module: ModuleRegistration }>({
    url: '/system/dynamic-modules',
    method: 'post',
    data,
  });
}

/**
 * 卸载模块
 */
export function unregisterModule(name: string, dropTable = false) {
  return apiRequest<{ unregistered: boolean; message: string }>({
    url: `/system/dynamic-modules/${name}?dropTable=${dropTable}`,
    method: 'delete',
  });
}

export function deleteModuleRecord(name: string) {
  return apiRequest<{ deleted: boolean; message: string }>({
    url: `/system/dynamic-modules/${name}/record`,
    method: 'delete',
  });
}

export function purgeModule(
  name: string,
  options?: { dropTable?: boolean; purgeSource?: boolean },
) {
  const dropTable = options?.dropTable ? 'true' : 'false';
  const purgeSource = options?.purgeSource === false ? 'false' : 'true';
  return apiRequest<{ deleted: boolean; message: string }>({
    url: `/system/dynamic-modules/${name}/purge?dropTable=${dropTable}&purgeSource=${purgeSource}`,
    method: 'delete',
  });
}

export function repairRegistries() {
  return apiRequest<{ repaired: boolean; message: string; summary: RegistryRepairSummary }>({
    url: '/system/dynamic-modules/repair',
    method: 'post',
  });
}

export function auditPendingActivations() {
  return apiRequest<{ audited: boolean; message: string; summary: ActivationAuditSummary }>({
    url: '/system/dynamic-modules/activation-audit',
    method: 'post',
  });
}

/**
 * 获取模块状态
 */
export function getModuleStatus(name: string) {
  return apiRequest<ModuleRegistration>({
    url: `/system/dynamic-modules/${name}`,
    method: 'get',
  });
}

export function getGeneratedModuleSchema(module: string) {
  return apiRequest<ModuleSchema>({
    url: '/system/dynamic-modules/schema',
    method: 'get',
    params: { module },
    skipErrorMessage: true,
  });
}
