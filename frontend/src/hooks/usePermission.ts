import { useAuthStore } from '../store/useAuthStore';
import { checkPermission } from '../core/permissions/checkPermission';

export const usePermission = () => {
  const { userInfo } = useAuthStore();
  const isAdmin = userInfo?.roles?.includes('admin') || false;

  return {
    isAdmin,
    hasPerm: (perm: string) => checkPermission(userInfo, perm),
    hasAnyPerm: (perms: string[]) => perms.some((perm) => checkPermission(userInfo, perm)),
  };
};
