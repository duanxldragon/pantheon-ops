import React, { useEffect } from 'react';
import { PageForbidden, RouteContentFallback } from '../../components';
import { ensureAuthUserInfo } from '../auth/bootstrap';
import { checkPermission } from '../permissions/checkPermission';
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

  return checkPermission(userInfo, permission) ? children : <PageForbidden />;
};

export default RoutePermissionGuard;
