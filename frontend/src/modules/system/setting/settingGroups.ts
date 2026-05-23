export const settingGroupOrder = [
  'basic',
  'platform',
  'security',
  'login',
  'audit',
  'upload',
  'i18n',
  'ui',
] as const;

export type SettingGroupKey = (typeof settingGroupOrder)[number];

export interface SettingGroupMeta {
  key: SettingGroupKey;
  titleKey: string;
  descriptionKey: string;
  path: string;
  tone: 'neutral' | 'warning' | 'danger';
}

export const settingGroups: SettingGroupMeta[] = settingGroupOrder.map((key) => ({
  key,
  titleKey: `system.setting.group.${key}`,
  descriptionKey: `system.setting.groupHint.${key}`,
  path: `/system/setting/${key}`,
  tone: key === 'audit' ? 'danger' : key === 'security' || key === 'login' ? 'warning' : 'neutral',
}));

export function isSettingGroupKey(value?: string): value is SettingGroupKey {
  return settingGroupOrder.includes(value as SettingGroupKey);
}

export function resolveSettingGroupMeta(value?: string): SettingGroupMeta {
  if (isSettingGroupKey(value)) {
    return settingGroups.find((item) => item.key === value) || settingGroups[0];
  }
  return settingGroups[0];
}

export function getSettingGroupPath(groupKey: SettingGroupKey) {
  return `/system/setting/${groupKey}`;
}
