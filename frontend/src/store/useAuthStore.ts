import { create } from 'zustand';
import { hasAuthSessionHint } from '../core/auth/clientSession';
import {
  clearStoredAuthUser,
  getBootstrappedAuthSession,
  persistAuthUser,
} from '../core/auth/sessionSnapshot';
import type { UserInfo } from './authTypes';

export function hasAuthSession(): boolean {
  return hasAuthSessionHint();
}

interface AuthState {
  /**
   * In Cookie auth mode, this is only a boolean flag for "is logged in" state
   * (non-empty = logged in), not the actual token. The actual access/refresh tokens
   * are stored in HttpOnly cookies managed by the backend and are never accessible
   * to JavaScript.
   */
  token: string | null;
  /**
   * In Cookie auth mode, this field is unused — the actual refresh token is held
   * in an HttpOnly cookie. Kept for type compatibility only.
   */
  refreshToken: string | null;
  userInfo: UserInfo | null;
  /** Whether the user is currently authenticated (token is non-empty). */
  isAuthenticated: boolean;
  setTokens: (token: string, refreshToken: string) => void;
  setUserInfo: (userInfo: UserInfo) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  ...getBootstrappedAuthSession(),
  get isAuthenticated() {
    return !!get().token;
  },
  setTokens: (token, refreshToken) => set({ token, refreshToken }),
  setUserInfo: (userInfo) => {
    persistAuthUser(userInfo);
    set({ userInfo });
  },
  clearAuth: () => {
    clearStoredAuthUser();
    set({ token: null, refreshToken: null, userInfo: null });
  },
}));
