// Shared menu contract type: importing it from modules/system/* would be a cross-layer
// reach, banned by docs/designs/REPOSITORY_LAYOUT.md §8.2 — this file is the sanctioned home.
export interface MenuNode {
  id: number;
  parentId: number;
  titleKey: string;
  path: string;
  component: string;
  pagePerm: string;
  perms: string;
  type: string;
  icon: string;
  routeName: string;
  module: string;
  sort: number;
  isVisible: number;
  isCache: number;
  isExternal: number;
  activeMenu: string;
  children?: MenuNode[];
}
