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
  nameI18nKey: string;
  descriptionI18nKey: string;
  payload: Omit<LabelSchemaPayload, 'name' | 'description'>;
}> = [
  {
    value: 'env',
    i18nKey: 'business.cmdb.label.preset.env',
    nameI18nKey: 'business.cmdb.label.presetValue.env.name',
    descriptionI18nKey: 'business.cmdb.label.presetValue.env.description',
    payload: {
      key: 'env',
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
    nameI18nKey: 'business.cmdb.label.presetValue.region.name',
    descriptionI18nKey: 'business.cmdb.label.presetValue.region.description',
    payload: {
      key: 'region',
      category: 'topology',
      valueMode: 'enum',
      options: ['cn-east-1', 'cn-north-1', 'edge-idc-a'],
      status: 'enabled',
    },
  },
  {
    value: 'biz',
    i18nKey: 'business.cmdb.label.preset.biz',
    nameI18nKey: 'business.cmdb.label.presetValue.biz.name',
    descriptionI18nKey: 'business.cmdb.label.presetValue.biz.description',
    payload: {
      key: 'biz',
      category: 'business',
      valueMode: 'enum',
      options: ['order-center', 'user-center', 'ops-platform'],
      status: 'enabled',
    },
  },
  {
    value: 'cluster',
    i18nKey: 'business.cmdb.label.preset.cluster',
    nameI18nKey: 'business.cmdb.label.presetValue.cluster.name',
    descriptionI18nKey: 'business.cmdb.label.presetValue.cluster.description',
    payload: {
      key: 'cluster',
      category: 'topology',
      valueMode: 'free',
      options: [],
      status: 'enabled',
    },
  },
  {
    value: 'db_type',
    i18nKey: 'business.cmdb.label.preset.dbType',
    nameI18nKey: 'business.cmdb.label.presetValue.dbType.name',
    descriptionI18nKey: 'business.cmdb.label.presetValue.dbType.description',
    payload: {
      key: 'db_type',
      category: 'database',
      valueMode: 'enum',
      options: ['mysql', 'postgresql'],
      status: 'enabled',
    },
  },
  {
    value: 'middleware',
    i18nKey: 'business.cmdb.label.preset.middleware',
    nameI18nKey: 'business.cmdb.label.presetValue.middleware.name',
    descriptionI18nKey: 'business.cmdb.label.presetValue.middleware.description',
    payload: {
      key: 'middleware',
      category: 'middleware',
      valueMode: 'enum',
      options: ['nginx', 'redis', 'minio', 'harbor'],
      status: 'enabled',
    },
  },
];
