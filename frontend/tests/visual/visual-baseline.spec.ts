import { expect, test, type Page } from '@playwright/test';
import type { MenuNode } from '../../src/modules/system/menu/api';
import { primeChineseLocale, signInAsAdmin } from '../smoke/helpers/auth';

function createVisualMenuNode(
  node: Pick<
    MenuNode,
    'id' | 'titleKey' | 'path' | 'type' | 'icon' | 'routeName' | 'module' | 'sort'
  > &
    Partial<MenuNode>,
): MenuNode {
  return {
    parentId: 0,
    component: '',
    pagePerm: '',
    perms: '',
    isVisible: 1,
    isCache: 0,
    isExternal: 0,
    activeMenu: '',
    ...node,
  };
}

const stableVisualMenuTree: MenuNode[] = [
  createVisualMenuNode({
    id: 6,
    titleKey: 'system.menu.dashboard',
    path: '/dashboard',
    component: 'dashboard',
    pagePerm: 'platform:dashboard:view',
    type: 'C',
    icon: 'dashboard',
    routeName: 'dashboard',
    module: 'platform',
    sort: 1,
  }),
  createVisualMenuNode({
    id: 1,
    titleKey: 'system.menu.access',
    path: '/system/access',
    type: 'M',
    icon: 'idcard',
    routeName: 'system-access',
    module: 'system.iam',
    sort: 20,
    children: [
      createVisualMenuNode({
        id: 7,
        parentId: 1,
        titleKey: 'system.menu.user',
        path: '/system/user',
        component: 'system/user/UserList',
        pagePerm: 'system:user:list',
        type: 'C',
        icon: 'user',
        routeName: 'system-user',
        module: 'system.iam',
        sort: 10,
      }),
      createVisualMenuNode({
        id: 8,
        parentId: 1,
        titleKey: 'system.menu.role',
        path: '/system/role',
        component: 'system/role/RoleList',
        pagePerm: 'system:role:list',
        type: 'C',
        icon: 'user-group',
        routeName: 'system-role',
        module: 'system.iam',
        sort: 20,
      }),
      createVisualMenuNode({
        id: 10,
        parentId: 1,
        titleKey: 'system.menu.permission',
        path: '/system/permission',
        component: 'system/permission/PermissionList',
        pagePerm: 'system:permission:list',
        type: 'C',
        icon: 'lock',
        routeName: 'system-permission',
        module: 'system.iam',
        sort: 30,
      }),
      createVisualMenuNode({
        id: 9,
        parentId: 1,
        titleKey: 'system.menu.menu',
        path: '/system/menu',
        component: 'system/menu/MenuList',
        pagePerm: 'system:menu:list',
        type: 'C',
        icon: 'menu',
        routeName: 'system-menu',
        module: 'system.iam',
        sort: 40,
      }),
    ],
  }),
  createVisualMenuNode({
    id: 2,
    titleKey: 'system.menu.org',
    path: '/system/org',
    type: 'M',
    icon: 'storage',
    routeName: 'system-org',
    module: 'system.org',
    sort: 30,
    children: [
      createVisualMenuNode({
        id: 35,
        parentId: 2,
        titleKey: 'system.menu.dept',
        path: '/system/dept',
        component: 'system/dept/DeptList',
        pagePerm: 'system:dept:list',
        type: 'C',
        icon: 'branch',
        routeName: 'system-dept',
        module: 'system.org',
        sort: 10,
      }),
      createVisualMenuNode({
        id: 43,
        parentId: 2,
        titleKey: 'system.menu.post',
        path: '/system/post',
        component: 'system/post/PostList',
        pagePerm: 'system:post:list',
        type: 'C',
        icon: 'tags',
        routeName: 'system-post',
        module: 'system.org',
        sort: 20,
      }),
    ],
  }),
  createVisualMenuNode({
    id: 5,
    titleKey: 'system.menu.security',
    path: '/system/security',
    type: 'M',
    icon: 'safe',
    routeName: 'system-security',
    module: 'system.auth',
    sort: 40,
    children: [
      createVisualMenuNode({
        id: 84,
        parentId: 5,
        titleKey: 'system.menu.loginLog',
        path: '/system/login-log',
        component: 'auth/LoginLogList',
        pagePerm: 'system:login-log:list',
        type: 'C',
        icon: 'clock',
        routeName: 'system-login-log',
        module: 'system.auth',
        sort: 10,
      }),
      createVisualMenuNode({
        id: 85,
        parentId: 5,
        titleKey: 'system.menu.session',
        path: '/system/session',
        component: 'auth/SessionList',
        pagePerm: 'system:session:list',
        type: 'C',
        icon: 'desktop',
        routeName: 'system-session',
        module: 'system.auth',
        sort: 20,
      }),
      createVisualMenuNode({
        id: 80,
        parentId: 5,
        titleKey: 'system.menu.operationLog',
        path: '/system/operation-log',
        component: 'system/audit/OperationLogList',
        pagePerm: 'system:operation-log:list',
        type: 'C',
        icon: 'file',
        routeName: 'system-operation-log',
        module: 'system.audit',
        sort: 30,
      }),
      createVisualMenuNode({
        id: 86,
        parentId: 5,
        titleKey: 'system.menu.securityEvent',
        path: '/system/security-event',
        component: 'auth/SecurityEventList',
        pagePerm: 'system:security-event:list',
        type: 'C',
        icon: 'safe',
        routeName: 'system-security-event',
        module: 'system.auth',
        sort: 40,
      }),
    ],
  }),
  createVisualMenuNode({
    id: 4,
    titleKey: 'system.menu.lowcode',
    path: '/system/lowcode',
    type: 'M',
    icon: 'code',
    routeName: 'system-lowcode',
    module: 'system.lowcode',
    sort: 45,
    children: [
      createVisualMenuNode({
        id: 64,
        parentId: 4,
        titleKey: 'system.menu.modules',
        path: '/system/modules',
        component: 'lowcode/dynamicmodule/ModuleManager',
        pagePerm: 'system:module:list',
        type: 'C',
        icon: 'apps',
        routeName: 'system-modules',
        module: 'system.lowcode',
        sort: 10,
      }),
      createVisualMenuNode({
        id: 70,
        parentId: 4,
        titleKey: 'system.menu.generator',
        path: '/system/generator',
        component: 'lowcode/generator/ModuleWizard',
        pagePerm: 'system:generator:use',
        type: 'C',
        icon: 'code',
        routeName: 'system-generator',
        module: 'system.lowcode',
        sort: 20,
      }),
    ],
  }),
  createVisualMenuNode({
    id: 3,
    titleKey: 'system.menu.config',
    path: '/system/config',
    type: 'M',
    icon: 'tool',
    routeName: 'system-config',
    module: 'system.config',
    sort: 50,
    children: [
      createVisualMenuNode({
        id: 51,
        parentId: 3,
        titleKey: 'system.menu.dict',
        path: '/system/dict',
        component: 'system/dict/DictPage',
        pagePerm: 'system:dict:list',
        type: 'C',
        icon: 'book',
        routeName: 'system-dict',
        module: 'system.config',
        sort: 10,
      }),
      createVisualMenuNode({
        id: 60,
        parentId: 3,
        titleKey: 'system.menu.setting',
        path: '/system/setting',
        component: 'system/setting/SettingOverviewPage',
        pagePerm: 'system:setting:list',
        type: 'C',
        icon: 'settings',
        routeName: 'system-setting',
        module: 'system.config',
        sort: 20,
      }),
      createVisualMenuNode({
        id: 73,
        parentId: 3,
        titleKey: 'system.menu.i18n',
        path: '/system/i18n',
        component: 'system/i18n/I18nList',
        pagePerm: 'system:i18n:list',
        type: 'C',
        icon: 'language',
        routeName: 'system-i18n',
        module: 'system.config',
        sort: 30,
      }),
    ],
  }),
];

async function installStablePlatformVisualRoutes(page: Page) {
  await page.route('**/api/v1/system/setting/public', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          settings: {
            'site.name': 'Pantheon Base',
            'org.enabled': 'true',
            'org.required_for_user': 'false',
            'i18n.default_language': 'zh-CN',
            'ui.default_theme': 'indigo',
            'ui.enable_tab_bar': 'true',
            'login.session_idle_minutes': '30',
          },
        },
      }),
    });
  });

  await page.route('**/api/v1/system/menu/tree**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: stableVisualMenuTree,
      }),
    });
  });

  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          id: 1,
          username: 'admin',
          nickname: 'Platform Admin',
          roles: ['admin'],
          perms: [
            'platform:dashboard:view',
            'system:user:list',
            'system:user:view',
            'system:user:create',
            'system:user:update',
            'system:user:reset',
            'system:user:export',
            'system:user:import',
            'system:user:batch-update',
            'system:user:batch-delete',
          ],
          preferences: { theme: 'indigo', layoutMode: 'vertical', densityMode: 'compact' },
        },
      }),
    });
  });

  await page.route('**/api/v1/dashboard/summary', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          totalUsers: 12,
          enabledUsers: 10,
          totalRoles: 4,
          totalDepts: 3,
          totalPosts: 5,
          totalDictTypes: 6,
          totalSettings: 8,
          totalI18nEntries: 20,
          activeModuleCount: 3,
          visibleMenuCount: 18,
          activeSessionCount: 4,
          loginSuccessCount: 18,
          loginFailureCount: 2,
          totalSecurityEventCount: 3,
          pendingSecurityEventCount: 1,
          todayOperationCount: 27,
          lastSuccessfulLoginAt: '2026-09-13T08:30:00Z',
          periodDays: 7,
          recentLogins: [
            {
              id: 101,
              username: 'admin',
              ipaddr: '10.10.0.10',
              browser: 'Chrome',
              os: 'Windows',
              status: 1,
              msg: 'auth.login.success',
              loginTime: '2026-09-13T08:30:00Z',
            },
            {
              id: 102,
              username: 'operator',
              ipaddr: '10.10.0.11',
              browser: 'Firefox',
              os: 'macOS',
              status: 1,
              msg: 'auth.login.success',
              loginTime: '2026-09-13T07:15:00Z',
            },
          ],
          orgGovernanceTaskCount: 2,
          orgGovernanceTasks: [
            {
              taskKey: 'dept-leaderless-1',
              domain: 'dept',
              issue: 'leaderless',
              action: 'assign-leader',
              scopeLabel: '部门',
              issueLabel: '缺负责人部门',
              actionLabel: '补负责人',
              resourceLabel: '平台运营部',
              relatedUserCount: 3,
              routePath: '/system/dept',
              routeStateDeptId: 11,
            },
            {
              taskKey: 'post-disabled-1',
              domain: 'post',
              issue: 'disabled',
              action: 'delete-or-keep-disabled',
              scopeLabel: '岗位',
              issueLabel: '已禁用岗位',
              actionLabel: '删除或保留停用岗位',
              resourceLabel: '值班工程师',
              relatedUserCount: 0,
              routePath: '/system/post',
              routeStateDeptId: 12,
            },
          ],
        },
      }),
    });
  });

  await page.route('**/api/v1/system/user/list**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 1,
              username: 'admin',
              nickname: 'Platform Admin',
              email: 'admin@example.test',
              phone: '13800000001',
              deptId: 11,
              deptName: '平台运营部',
              postId: 21,
              postName: '平台管理员',
              status: 1,
              createdAt: '2026-01-15T08:00:00Z',
              roleIds: [1],
              roleKeys: ['admin'],
              roleNames: ['admin'],
            },
            {
              id: 2,
              username: 'operator',
              nickname: '值班工程师',
              email: 'operator@example.test',
              phone: '13800000002',
              deptId: 12,
              deptName: '运维支持部',
              postId: 22,
              postName: '值班工程师',
              status: 1,
              createdAt: '2026-02-20T09:30:00Z',
              roleIds: [2],
              roleKeys: ['operator'],
              roleNames: ['operator'],
            },
          ],
          total: 2,
          page: 1,
          pageSize: 10,
        },
      }),
    });
  });

  await page.route('**/api/v1/system/role/list**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 1,
              roleName: 'admin',
              roleKey: 'admin',
              sort: 1,
              status: 1,
              createdAt: '2026-01-01T00:00:00Z',
              menuIds: [1, 6, 7],
              permissionKeys: ['platform:dashboard:view'],
              dataScope: 'all',
            },
            {
              id: 2,
              roleName: 'operator',
              roleKey: 'operator',
              sort: 2,
              status: 1,
              createdAt: '2026-01-02T00:00:00Z',
              menuIds: [6, 7],
              permissionKeys: ['platform:dashboard:view'],
              dataScope: 'dept',
            },
          ],
          total: 2,
          page: 1,
          pageSize: 9999,
        },
      }),
    });
  });

  await page.route('**/api/v1/system/dept/tree**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: [
          {
            id: 10,
            parentId: 0,
            ancestors: '0',
            isRoot: true,
            deptName: 'Pantheon',
            sort: 1,
            leaderUserId: 1,
            leader: 'admin',
            phone: '',
            email: '',
            status: 1,
            childDeptCount: 2,
            postCount: 2,
            isLeaderless: false,
            isNoPost: false,
            isEmpty: false,
            children: [
              {
                id: 11,
                parentId: 10,
                ancestors: '0,10',
                isRoot: false,
                deptName: '平台运营部',
                sort: 1,
                leaderUserId: 1,
                leader: 'admin',
                phone: '',
                email: '',
                status: 1,
                childDeptCount: 0,
                postCount: 1,
                isLeaderless: false,
                isNoPost: false,
                isEmpty: false,
                children: [],
              },
              {
                id: 12,
                parentId: 10,
                ancestors: '0,10',
                isRoot: false,
                deptName: '运维支持部',
                sort: 2,
                leaderUserId: 0,
                leader: '',
                phone: '',
                email: '',
                status: 1,
                childDeptCount: 0,
                postCount: 1,
                isLeaderless: true,
                isNoPost: false,
                isEmpty: false,
                children: [],
              },
            ],
          },
        ],
      }),
    });
  });

  await page.route('**/api/v1/system/post/list**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 21,
              deptId: 11,
              deptName: '平台运营部',
              postCode: 'platform-admin',
              postName: '平台管理员',
              sort: 1,
              status: 1,
              remark: '',
              assignedUserCount: 1,
              governanceTags: [],
              governanceTagLabels: [],
              governanceBlockedBy: [],
              governanceBlockedDesc: [],
              governanceActions: [],
              governanceActionLabel: [],
              createdAt: '2026-01-10T00:00:00Z',
            },
            {
              id: 22,
              deptId: 12,
              deptName: '运维支持部',
              postCode: 'on-call',
              postName: '值班工程师',
              sort: 2,
              status: 1,
              remark: '',
              assignedUserCount: 1,
              governanceTags: [],
              governanceTagLabels: [],
              governanceBlockedBy: [],
              governanceBlockedDesc: [],
              governanceActions: [],
              governanceActionLabel: [],
              createdAt: '2026-01-11T00:00:00Z',
            },
          ],
          total: 2,
          page: 1,
          pageSize: 9999,
        },
      }),
    });
  });

  await page.route('**/api/v1/system/refresh/state**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 200, data: { topics: {} } }),
    });
  });
}

async function waitForVisualStability(page: Page) {
  await page.waitForLoadState('domcontentloaded');
  await page.evaluate(async () => {
    await document.fonts.ready;
  });
}

async function primeColorMode(page: Page, mode: 'light' | 'dark') {
  await page.addInitScript((colorMode) => {
    globalThis.localStorage?.setItem('pantheon_color_mode', colorMode);
  }, mode);
}

test('login page visual baseline', async ({ page }) => {
  await primeColorMode(page, 'light');
  await primeChineseLocale(page);
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.auth-login-page')).toBeVisible();
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('login.png');
});

test('@mobile login page visual baseline', async ({ page }) => {
  await primeColorMode(page, 'light');
  await primeChineseLocale(page);
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.auth-login-page')).toBeVisible();
  await expect(page.locator('html')).toHaveAttribute('data-color-mode', 'light');
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('login-mobile.png');
});

test('@dark login page visual baseline', async ({ page }) => {
  await primeColorMode(page, 'dark');
  await primeChineseLocale(page);
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.auth-login-page')).toBeVisible();
  await expect(page.locator('html')).toHaveAttribute('data-color-mode', 'dark');
  await expect(page.locator('body')).toHaveAttribute('arco-theme', 'dark');
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('login-dark.png');
});

test('workspace dashboard visual baseline', async ({ page }) => {
  await installStablePlatformVisualRoutes(page);
  await signInAsAdmin(page);
  await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.dashboard-hero-card')).toBeVisible();
  await expect(page.locator('.dashboard-stat-card').first()).toBeVisible();
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('dashboard.png', {
    mask: [
      page.locator('.dashboard-stat-card__value'),
      page.locator('.dashboard-stat-card--users .arco-tag'),
      page.locator('.dashboard-focus-item__value'),
      page.locator('.dashboard-domain-card__summary'),
      page.locator('.dashboard-login-table .app-table tbody tr'),
    ],
  });
});

test('system user list visual baseline', async ({ page }) => {
  await installStablePlatformVisualRoutes(page);
  await signInAsAdmin(page);
  await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.system-list__table-card')).toBeVisible();
  await expect(page.locator('.app-table')).toBeVisible();
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('system-user-list.png', {
    mask: [
      page.locator('.governance-summary-bar__metric-value'),
      page.locator('.system-list__table-card .app-table tbody tr'),
    ],
  });
});
