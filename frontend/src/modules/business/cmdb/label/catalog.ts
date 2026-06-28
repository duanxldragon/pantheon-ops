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
  descriptionI18nKey: string;
  payload: LabelSchemaPayload;
}> = [
  {
    value: 'env',
    i18nKey: 'business.cmdb.label.preset.env',
    descriptionI18nKey: 'business.cmdb.label.preset.env.description',
    payload: {
      key: 'env',
      name: '环境',
      category: 'environment',
      valueMode: 'dict',
      dictCode: 'cmdb_env',
      options: ['dev', 'test', 'prod'],
      status: 'enabled',
    },
  },
  {
    value: 'region',
    i18nKey: 'business.cmdb.label.preset.region',
    descriptionI18nKey: 'business.cmdb.label.preset.region.description',
    payload: {
      key: 'region',
      name: '区域',
      category: 'topology',
      valueMode: 'enum',
      options: ['cn-east-1', 'cn-north-1', 'edge-idc-a'],
      status: 'enabled',
    },
  },
  {
    value: 'biz',
    i18nKey: 'business.cmdb.label.preset.biz',
    descriptionI18nKey: 'business.cmdb.label.preset.biz.description',
    payload: {
      key: 'biz',
      name: '业务系统',
      category: 'business',
      valueMode: 'enum',
      options: ['order-center', 'user-center', 'ops-platform'],
      status: 'enabled',
    },
  },
  {
    value: 'cluster',
    i18nKey: 'business.cmdb.label.preset.cluster',
    descriptionI18nKey: 'business.cmdb.label.preset.cluster.description',
    payload: {
      key: 'cluster',
      name: '集群',
      category: 'topology',
      valueMode: 'free',
      options: [],
      status: 'enabled',
    },
  },
  {
    value: 'db_type',
    i18nKey: 'business.cmdb.label.preset.dbType',
    descriptionI18nKey: 'business.cmdb.label.preset.dbType.description',
    payload: {
      key: 'db_type',
      name: '数据库类型',
      category: 'database',
      valueMode: 'enum',
      options: ['mysql', 'postgresql'],
      status: 'enabled',
    },
  },
  {
    value: 'middleware',
    i18nKey: 'business.cmdb.label.preset.middleware',
    descriptionI18nKey: 'business.cmdb.label.preset.middleware.description',
    payload: {
      key: 'middleware',
      name: '中间件',
      category: 'middleware',
      valueMode: 'enum',
      options: ['nginx', 'redis', 'minio', 'harbor'],
      status: 'enabled',
    },
  },
];
