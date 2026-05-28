import { apiRequest } from '../../../../api/request';

export type LabelValueMode = 'free' | 'enum' | 'dict';
export type LabelSchemaStatus = 'enabled' | 'disabled';

export interface LabelSchemaRow {
  id: number;
  key: string;
  name: string;
  category: string;
  valueMode: LabelValueMode;
  dictCode: string;
  options: string[];
  required: boolean;
  status: LabelSchemaStatus;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface LabelSchemaQuery {
  keyword?: string;
  status?: string;
  category?: string;
  page?: number;
  pageSize?: number;
}

export interface LabelSchemaListResp {
  items: LabelSchemaRow[];
  total: number;
  page: number;
  pageSize: number;
}

type LabelSchemaListApiResp = LabelSchemaListResp | LabelSchemaRow[];

export interface LabelSchemaPayload {
  key?: string;
  name?: string;
  category?: string;
  valueMode?: LabelValueMode;
  dictCode?: string;
  options?: string[];
  required?: boolean;
  status?: LabelSchemaStatus;
  description?: string;
}

function normalizeLabelSchemaRow(row: Partial<LabelSchemaRow>): LabelSchemaRow {
  return {
    id: Number(row.id || 0),
    key: String(row.key || ''),
    name: String(row.name || ''),
    category: String(row.category || 'base'),
    valueMode: (row.valueMode || 'free') as LabelValueMode,
    dictCode: String(row.dictCode || ''),
    options: Array.isArray(row.options) ? row.options.map((item) => String(item)) : [],
    required: Boolean(row.required),
    status: (row.status || 'enabled') as LabelSchemaStatus,
    description: String(row.description || ''),
    createdAt: String(row.createdAt || ''),
    updatedAt: String(row.updatedAt || ''),
  };
}

export async function getLabelSchemaList(params?: LabelSchemaQuery): Promise<LabelSchemaListResp> {
  const result = await apiRequest<LabelSchemaListApiResp>({
    url: '/business/cmdb/labels',
    method: 'get',
    params,
  });
  if (Array.isArray(result)) {
    const current = params?.page && params.page > 0 ? params.page : 1;
    const currentPageSize =
      params?.pageSize && params.pageSize > 0 ? params.pageSize : Math.max(result.length, 10);
    return {
      items: result.map((item) => normalizeLabelSchemaRow(item)),
      total: result.length,
      page: current,
      pageSize: currentPageSize,
    };
  }
  return {
    items: Array.isArray(result.items) ? result.items.map((item) => normalizeLabelSchemaRow(item)) : [],
    total: typeof result.total === 'number' ? result.total : Array.isArray(result.items) ? result.items.length : 0,
    page: typeof result.page === 'number' ? result.page : params?.page || 1,
    pageSize:
      typeof result.pageSize === 'number'
        ? result.pageSize
        : params?.pageSize || (Array.isArray(result.items) ? result.items.length || 10 : 10),
  };
}

export function getLabelSchemaOptions(params?: Pick<LabelSchemaQuery, 'status' | 'category'>) {
  return apiRequest<Partial<LabelSchemaRow>[]>({
    url: '/business/cmdb/labels/options',
    method: 'get',
    params,
  }).then((result) => result.map((item) => normalizeLabelSchemaRow(item)));
}

export function createLabelSchema(data: LabelSchemaPayload) {
  return apiRequest<LabelSchemaRow>({
    url: '/business/cmdb/labels',
    method: 'post',
    data,
  });
}

export function updateLabelSchema(id: number, data: LabelSchemaPayload) {
  return apiRequest<LabelSchemaRow>({
    url: `/business/cmdb/labels/${id}`,
    method: 'put',
    data,
  });
}

export function deleteLabelSchema(id: number) {
  return apiRequest<void>({
    url: `/business/cmdb/labels/${id}`,
    method: 'delete',
  });
}
