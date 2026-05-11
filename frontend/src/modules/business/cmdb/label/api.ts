import { apiRequest } from '../../../../api/request';

export type LabelValueMode = 'free' | 'enum' | 'dict';
export type LabelSchemaStatus = 'enabled' | 'disabled';

export interface LabelSchemaRow {
  id: number;
  key: string;
  name: string;
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
}

export interface LabelSchemaPayload {
  key?: string;
  name?: string;
  valueMode?: LabelValueMode;
  dictCode?: string;
  options?: string[];
  required?: boolean;
  status?: LabelSchemaStatus;
  description?: string;
}

export function getLabelSchemaList(params?: LabelSchemaQuery) {
  return apiRequest<LabelSchemaRow[]>({
    url: '/business/cmdb/labels',
    method: 'get',
    params,
  });
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
