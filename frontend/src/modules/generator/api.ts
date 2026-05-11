import { apiRequest } from '../../api/request';
import type { GeneratedFile } from './exporter';
import type { ModuleField, ModuleSchema } from './schema';

export interface GenerateAndRegisterResp {
  module: {
    id: number;
    name: string;
    displayName: string;
    scope: string;
    tableName: string;
    status: number;
    installedAt: string;
    builtIn?: boolean;
  };
  summary: {
    moduleKey: string;
    routePath: string;
    routeName: string;
    componentKey: string;
    permissionPrefix: string;
    contract: {
      templateVersion: string;
      dataScopeEnabled: boolean;
      dataScopeMode: string;
      dependencyCount: number;
      relationCount: number;
      dependencies?: ModuleSchema['dependencies'];
      relations?: ModuleSchema['relations'];
    };
    parentMenuPath: string;
    parentMenuSource: 'explicit' | 'inferred' | 'top_level';
    parentMenuExists: boolean;
    backendModulePath: string;
    frontendModulePath: string;
    schemaPath: string;
    requiresRestart: boolean;
    requiresFrontendBuild: boolean;
    verifications: Array<{
      code: string;
      status: 'pass' | 'warn' | 'info';
      messageKey: string;
      detail: string;
    }>;
  };
  writtenFiles: string[];
  requiresRestart: boolean;
  requiresFrontendBuild: boolean;
  message: string;
}

export function generateAndRegisterModule(data: {
  schema: ModuleSchema;
  files: GeneratedFile[];
  overwrite?: boolean;
}) {
  return apiRequest<GenerateAndRegisterResp>({
    url: '/system/dynamic-modules/generate',
    method: 'post',
    data,
    skipErrorMessage: true,
  });
}

export interface GeneratorTableOption {
  tableName: string;
  comment: string;
  engine: string;
  rows: number;
}

export interface GeneratorTablePreview {
  tableName: string;
  tableComment: string;
  suggestedName: string;
  suggestedScope: 'system' | 'business';
  suggestedTitle: string;
  fields: ModuleField[];
}

export interface GeneratorDatasource {
  id: string;
  name: string;
  driver: string;
  host?: string;
  port?: number;
  databaseName: string;
  username?: string;
  status: number;
  remark?: string;
  readonlyScope?: string;
  lastCheckedAt?: string;
  lastCheckStatus?: string;
  lastCheckError?: string;
  isCurrent: boolean;
}

export interface UpsertGeneratorDatasourcePayload {
  name: string;
  driver: string;
  host: string;
  port: number;
  databaseName: string;
  username: string;
  password?: string;
  status: number;
  remark?: string;
}

export function listGeneratorDatasources() {
  return apiRequest<GeneratorDatasource[]>({
    url: '/system/generator/datasources',
    method: 'get',
  });
}

export function createGeneratorDatasource(data: UpsertGeneratorDatasourcePayload) {
  return apiRequest<GeneratorDatasource>({
    url: '/system/generator/datasources',
    method: 'post',
    data,
  });
}

export function updateGeneratorDatasource(id: string, data: UpsertGeneratorDatasourcePayload) {
  return apiRequest<GeneratorDatasource>({
    url: `/system/generator/datasources/${id}`,
    method: 'put',
    data,
  });
}

export function deleteGeneratorDatasource(id: string) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/generator/datasources/${id}`,
    method: 'delete',
  });
}

export function testGeneratorDatasource(id: string) {
  return apiRequest<GeneratorDatasource>({
    url: `/system/generator/datasources/${id}/test`,
    method: 'post',
  });
}

export function listGeneratorTables(datasourceId?: string, keyword?: string) {
  return apiRequest<GeneratorTableOption[]>({
    url: '/system/generator/tables',
    method: 'get',
    params: { datasourceId, keyword },
  });
}

export function previewGeneratorTable(tableName: string, datasourceId?: string) {
  return apiRequest<GeneratorTablePreview>({
    url: '/system/generator/table-schema',
    method: 'get',
    params: { tableName, datasourceId },
  });
}
