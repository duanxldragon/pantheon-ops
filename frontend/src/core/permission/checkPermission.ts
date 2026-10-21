import type { UserInfo } from '../../store/authTypes';

export function checkPermission(
  userInfo: UserInfo | null | undefined,
  permission: string,
): boolean {
  if (!permission) {
    return true;
  }

  if (userInfo?.roles?.includes('admin')) {
    return true;
  }

  return userInfo?.perms?.includes(permission) ?? false;
}
