import { useAuthStore } from '../store/useAuthStore';

export const usePermission = () => {
  const { userInfo } = useAuthStore();
  const isAdmin = userInfo?.roles?.includes('admin') || false;

  return {
    isAdmin,
    hasPerm: (perm: string) => isAdmin || userInfo?.perms?.includes(perm) || false,
    hasAnyPerm: (perms: string[]) =>
      isAdmin || perms.some((perm) => userInfo?.perms?.includes(perm)),
  };
};
