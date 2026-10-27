import { create } from 'zustand';
import { getMenuTree, type MenuNode } from '../modules/system/iam/menu/api';

let menuFetchSeq = 0;
let menuFetchPromise: Promise<MenuNode[]> | null = null;

interface MenuState {
  menuTree: MenuNode[];
  loading: boolean;
  /** True when the last fetch attempt failed and menuTree is still empty. */
  loadError: boolean;
  fetchMenuTree: (options?: { force?: boolean }) => Promise<MenuNode[]>;
  resetMenuTree: () => void;
}

export const useMenuStore = create<MenuState>((set, get) => ({
  menuTree: [],
  loading: false,
  loadError: false,
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
    set({ loading: true, loadError: false });
    menuFetchPromise = getMenuTree({ scope: 'nav' })
      .then((data) => {
        if (currentSeq === menuFetchSeq) {
          set({ menuTree: data, loading: false, loadError: false });
        }
        return data;
      })
      .catch(() => {
        if (currentSeq === menuFetchSeq) {
          set({ loading: false, loadError: true });
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
        set({ loading: false, loadError: true });
      }
      return [];
    }
  },
  resetMenuTree: () => {
    menuFetchSeq += 1;
    menuFetchPromise = null;
    set({ menuTree: [], loading: false, loadError: false });
  },
}));
