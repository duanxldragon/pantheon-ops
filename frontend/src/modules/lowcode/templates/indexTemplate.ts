/**
 * 前端模块注册模板 (index.ts)
 *
 * 基于 ${toSrcRoot}/core/router/types -> defineModule
 * 参考: system/user/index.ts
 *
 * 模板变量占位符:
 * - ${toSrcRoot}       - 相对路径到 src 根目录
 * - ${name}            - 模块名 (如 cmdb/host)
 * - ${scope}           - 作用域 (business/system)
 * - ${modelName}       - PascalCase 模型名 (如 CmdbHost)
 * - ${routePath}       - 路由路径 (如 cmdb/host)
 * - ${titleKey}        - 标题 i18n key
 * - ${permissionPrefix} - 权限前缀 (如 cmdb:host)
 * - ${moduleNamespace} - i18n 命名空间 (如 cmdb.host)
 * - ${routeName}       - 路由名称 (如 cmdb-host)
 * - ${listComponentKey} - 列表组件 key
 * - ${detailComponentKey} - 详情组件 key
 * - ${generateNavigation} - 是否生成导航 (true/false)
 * - ${generateDetailRoute} - 是否生成详情路由 (true/false)
 * - ${dashboardWidgets} - 仪表板挂件配置字符串 (含缩进和尾部逗号，或为空)
 * - ${permissionItems} - 权限项数组字符串 (含缩进和换行)
 */

export const INDEX_TEMPLATE = `import { defineModule } from '${toSrcRoot}/core/router/types';

export const \${modelName}Module = defineModule({
  name: '\${name}',
  scope: '\${scope}',
  routes: \${generateNavigation}
    ? [
  {
    path: '\${routePath}',
    routeName: '\${routeName}',
    titleKey: '\${titleKey}',
    icon: 'apps',
    pagePermission: '\${permissionPrefix}:list',
    componentKey: '\${listComponentKey}',
  },
  \${generateDetailRoute}
    ? {
  path: '\${routePath}/:id',
  routeName: '\${routeName}-detail',
  titleKey: '\${titleKey}',
  pagePermission: '\${permissionPrefix}:view',
  activeMenu: '\${buildPageRoutePath}',
  componentKey: '\${detailComponentKey}',
},
    : ''
}
  ]
    : '[]'
},
  menus: \${generateNavigation}
    ? [
  { path: '\${buildPageRoutePath}', titleKey: '\${titleKey}', icon: 'apps', routeName: '\${routeName}', module: '\${moduleNamespace}' },
]
    : '[]'
},\${dashboardWidgets}
  permissions: \${permissionItems}
    ? [
  \${permissionItems}
]
    : '[]'
},
  i18nNamespaces: ['\${moduleNamespace}'],
});
`;
