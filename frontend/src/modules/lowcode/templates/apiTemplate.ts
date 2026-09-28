/**
 * 前端 API 层模板 (api.ts)
 *
 * 基于真实 system/user/api.ts 结构
 *
 * 模板变量占位符:
 * - ${toSrcRoot}         - 相对路径到 src 根目录
 * - ${routePath}         - API 路由路径
 * - ${modelName}         - PascalCase 模型名
 * - ${extraImports}      - 额外的 import 语句字符串
 * - ${relationLookupApi}  - 关系查找 API 字符串
 * - ${relationBindingApi} - 多对多关系绑定 API 字符串
 * - ${mutatingApi}       - 增删改 API 字符串
 * - ${extraApi}          - 导入导出 API 字符串
 *
 * 方法调用占位符 (需由生成器方法替换):
 * - {{generateTSFields:list}}
 * - {{generateTSFields:detail}}
 * - {{generateTSFields:create}}
 * - {{generateTSFields:update}}
 * - {{generateTSQueryFields}}
 */

export const API_TEMPLATE = `import { apiRequest } from '${toSrcRoot}/api/request';
{{extraImports}}

export interface {{modelName}}ListRow {
  id: number;
{{generateTSFields:list}}
  createdAt: string;
}

export interface {{modelName}}Detail {
  id: number;
{{generateTSFields:detail}}
  createdAt: string;
  updatedAt: string;
}

export interface {{modelName}}ListQuery {
{{generateTSQueryFields}}
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface {{modelName}}ListPageResp {
  items: {{modelName}}ListRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface {{modelName}}CreatePayload {
{{generateTSFields:create}}
}

export interface {{modelName}}UpdatePayload {
{{generateTSFields:update}}
}

{{relationLookupApi}}
{{relationBindingApi}}

export function get{{modelName}}List(params?: {{modelName}}ListQuery) {
  return apiRequest<{{modelName}}ListPageResp>({
    url: '{{routePath}}/list',
    method: 'get',
    params,
  });
}

export function get{{modelName}}Detail(id: number) {
  return apiRequest<{{modelName}}Detail>({
    url: \`{{routePath}}/\${id}\`,
    method: 'get',
  });
}

{{mutatingApi}}
{{extraApi}}
`;
