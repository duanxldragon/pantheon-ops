import { create } from 'zustand';
import type { UserPlatformPreferences } from '../modules/auth/api';

export interface UserInfo {
  id: number;
  username: string;
  nickname: string;
  avatar?: string;
  email?: string;
  phone?: string;
  roles?: string[];
  perms?: string[];
  preferences?: UserPlatformPreferences;
}

export function hasAuthCookie(): boolean {
  return document.cookie.indexOf('pantheon_csrf_token=') >= 0;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  userInfo: UserInfo | null;
  setTokens: (token: string, refreshToken: string) => void;
  setUserInfo: (userInfo: UserInfo) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()((set) => ({
  token: null,
  refreshToken: null,
  userInfo: null,
  setTokens: (token, refreshToken) => set({ token, refreshToken }),
  setUserInfo: (userInfo) => set({ userInfo }),
  clearAuth: () => set({ token: null, refreshToken: null, userInfo: null }),
}));
