import React, { useEffect } from 'react';
import { PageForbidden, RouteContentFallback } from '../../components';
import { ensureAuthUserInfo } from '../auth/bootstrap';
import { useAuthStore } from '../../store/useAuthStore';

interface RoutePermissionGuardProps {
  permission?: string;
  children: React.ReactElement;
}

const RoutePermissionGuard: React.FC<RoutePermissionGuardProps> = ({ permission, children }) => {
  const { token, userInfo } = useAuthStore();

  useEffect(() => {
    if (!permission || !token || userInfo) {
      return;
    }
    void ensureAuthUserInfo().catch(() => undefined);
  }, [permission, token, userInfo]);

  if (!permission) {
    return children;
  }

  if (!userInfo) {
    return <RouteContentFallback />;
  }

  const isAdmin = userInfo.roles?.includes('admin') || false;
  const allowed = isAdmin || userInfo.perms?.includes(permission);

  return allowed ? children : <PageForbidden />;
};

export default RoutePermissionGuard;
