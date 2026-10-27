import { describe, expect, it } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useLayoutMenu } from '../../../../src/core/layout/useLayoutMenu';
import type { MenuNode } from '../../../../src/modules/system/iam/menu/api';

function menuNode(overrides: Partial<MenuNode> & Pick<MenuNode, 'id' | 'path'>): MenuNode {
  return {
    parentId: 0,
    titleKey: `menu.${overrides.id}`,
    component: '',
    pagePerm: '',
    perms: '',
    type: 'C',
    icon: '',
    routeName: '',
    module: 'system.lowcode',
    sort: 0,
    isVisible: 1,
    isCache: 0,
    isExternal: 0,
    activeMenu: '',
    ...overrides,
  };
}

const lowcodeGroup = menuNode({
  id: 100,
  path: '/system/lowcode',
  titleKey: 'system.menu.lowcode',
  type: 'M',
  children: [
    menuNode({ id: 101, path: '/system/modules', titleKey: 'system.menu.modules' }),
    menuNode({ id: 102, path: '/system/generator', titleKey: 'system.menu.generator' }),
  ],
});

function visibleTree(isPathEntitled: (path: string) => boolean) {
  const { result } = renderHook(() =>
    useLayoutMenu({
      menuTree: [lowcodeGroup],
      orgEnabled: true,
      tenantEnabled: true,
      tenantScoped: true,
      currentPath: '/dashboard',
      activeMenuPath: '/dashboard',
      isPathEntitled,
      t: (key) => key,
      handleMenuNavigation: () => undefined,
    }),
  );
  return result.current.visibleMenuTree;
}

describe('useLayoutMenu plan filtering', () => {
  it('drops a group when every child is plan-gated and the group has no route', () => {
    expect(visibleTree(() => false)).toEqual([]);
  });

  it('keeps the group but only the entitled children', () => {
    const [group] = visibleTree((path) => path === '/system/modules');

    expect(group.path).toBe('/system/lowcode');
    expect(group.children?.map((child) => child.path)).toEqual(['/system/modules']);
  });
});
