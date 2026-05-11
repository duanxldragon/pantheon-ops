import { getMe, type UserInfo } from '../../modules/auth/api';
import { hasAuthCookie, useAuthStore } from '../../store/useAuthStore';

const COOKIE_TOKEN_PLACEHOLDER = '_cookie';

let pendingProfilePromise: Promise<UserInfo | null> | null = null;

export function ensureAuthUserInfo() {
  const { userInfo, setTokens, setUserInfo } = useAuthStore.getState();

  if (!hasAuthCookie()) {
    return Promise.resolve(null);
  }

  if (userInfo) {
    return Promise.resolve(userInfo);
  }

  if (pendingProfilePromise) {
    return pendingProfilePromise;
  }

  pendingProfilePromise = getMe()
    .then((profile) => {
      setTokens(COOKIE_TOKEN_PLACEHOLDER, COOKIE_TOKEN_PLACEHOLDER);
      setUserInfo(profile);
      return profile;
    })
    .catch(() => {
      return null;
    })
    .finally(() => {
      pendingProfilePromise = null;
    });

  return pendingProfilePromise;
}
