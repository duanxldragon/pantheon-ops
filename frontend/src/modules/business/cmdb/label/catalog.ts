import type { LabelSchemaPayload } from './api';

export const labelCategoryOptions = [
  { value: 'base', i18nKey: 'business.cmdb.label.category.base' },
  { value: 'environment', i18nKey: 'business.cmdb.label.category.environment' },
  { value: 'business', i18nKey: 'business.cmdb.label.category.business' },
  { value: 'topology', i18nKey: 'business.cmdb.label.category.topology' },
  { value: 'middleware', i18nKey: 'business.cmdb.label.category.middleware' },
  { value: 'database', i18nKey: 'business.cmdb.label.category.database' },
] as const;

export const labelPresetOptions: Array<{
  value: string;
  i18nKey: string;
  payload: LabelSchemaPayload;
}> = [
  {
    value: 'env',
    i18nKey: 'business.cmdb.label.preset.env',
    payload: {
      key: 'env',
      name: '环境',
      category: 'environment',
      valueMode: 'dict',
      dictCode: 'cmdb_env',
      options: ['dev', 'test', 'prod'],
      description: '部署环境，如 dev/test/prod',
      status: 'enabled',
    },
  },
  {
    value: 'region',
    i18nKey: 'business.cmdb.label.preset.region',
    payload: {
      key: 'region',
      name: '区域',
      category: 'topology',
      valueMode: 'enum',
      options: ['cn-east-1', 'cn-north-1', 'edge-idc-a'],
      description: '区域、机房或可用区',
      status: 'enabled',
    },
  },
  {
    value: 'biz',
    i18nKey: 'business.cmdb.label.preset.biz',
    payload: {
      key: 'biz',
      name: '业务系统',
      category: 'business',
      valueMode: 'enum',
      options: ['order-center', 'user-center', 'ops-platform'],
      description: '业务系统或应用归属',
      status: 'enabled',
    },
  },
  {
    value: 'cluster',
    i18nKey: 'business.cmdb.label.preset.cluster',
    payload: {
      key: 'cluster',
      name: '集群',
      category: 'topology',
      valueMode: 'free',
      options: [],
      description: '集群或资源池归属',
      status: 'enabled',
    },
  },
  {
    value: 'db_type',
    i18nKey: 'business.cmdb.label.preset.dbType',
    payload: {
      key: 'db_type',
      name: '数据库类型',
      category: 'database',
      valueMode: 'enum',
      options: ['mysql', 'postgresql'],
      description: '主机承载的数据库类型',
      status: 'enabled',
    },
  },
  {
    value: 'middleware',
    i18nKey: 'business.cmdb.label.preset.middleware',
    payload: {
      key: 'middleware',
      name: '中间件',
      category: 'middleware',
      valueMode: 'enum',
      options: ['nginx', 'redis', 'minio', 'harbor'],
      description: '主机承载的中间件类型',
      status: 'enabled',
    },
  },
];
