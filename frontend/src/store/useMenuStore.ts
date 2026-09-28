import { create } from 'zustand';
import { getMenuTree, type MenuNode } from '../modules/system/menu/api';

let menuFetchSeq = 0;
let menuFetchPromise: Promise<MenuNode[]> | null = null;

interface MenuState {
  menuTree: MenuNode[];
  loading: boolean;
  fetchMenuTree: (options?: { force?: boolean }) => Promise<MenuNode[]>;
  resetMenuTree: () => void;
}

export const useMenuStore = create<MenuState>((set, get) => ({
  menuTree: [],
  loading: false,
  fetchMenuTree: async (options) => {
    const currentState = get();
    const force = Boolean(options?.force);
    if (!force && currentState.menuTree.length > 0) {
      return currentState.menuTree;
    }
    if (!force && menuFetchPromise) {
      return menuFetchPromise;
    }
    const currentSeq = ++menuFetchSeq;
    set({ loading: true });
    menuFetchPromise = getMenuTree({ scope: 'nav' })
      .then((data) => {
        if (currentSeq === menuFetchSeq) {
          set({ menuTree: data, loading: false });
        }
        return data;
      })
      .catch(() => {
        if (currentSeq === menuFetchSeq) {
          set({ loading: false });
        }
        return [];
      })
      .finally(() => {
        if (menuFetchPromise) {
          menuFetchPromise = null;
        }
      });
    try {
      return await menuFetchPromise;
    } catch {
      if (currentSeq === menuFetchSeq) {
        set({ loading: false });
      }
      return [];
    }
  },
  resetMenuTree: () => {
    menuFetchSeq += 1;
    menuFetchPromise = null;
    set({ menuTree: [], loading: false });
  },
}));
