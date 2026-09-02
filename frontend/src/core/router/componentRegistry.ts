// 从 base 包重新导出组件注册表
// 契约检查脚本通过正则表达式解析，需要显式列出所有 key

import { lazy } from 'react';

export const componentRegistry = {
  'dashboard': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'auth/SecurityCenter': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'auth/LoginLogList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'auth/SecurityEventList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'auth/SessionList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/profile/ProfileCenter': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/dict/DictPage': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/i18n/I18nList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/dept/DeptList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/menu/MenuList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/permission/PermissionList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/post/PostList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/role/RoleList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/setting/SettingOverviewPage': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/setting/SettingGroupPage': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/user/UserList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/user/UserDetail': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'system/audit/OperationLogList': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'lowcode/dynamicmodule/ModuleManager': lazy(() => import('@duanxldragon/pantheon-base-ui')),
  'lowcode/generator/ModuleWizard': lazy(() => import('@duanxldragon/pantheon-base-ui')),
};

