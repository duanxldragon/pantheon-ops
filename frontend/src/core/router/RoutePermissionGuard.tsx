import React, { useEffect, useState } from 'react';
import { PageError, PageForbidden, RouteContentFallback } from '../../components';
import { ensureAuthUserInfo } from '../auth/bootstrap';
import { checkPermission } from '../permission/checkPermission';
import { useAuthStore } from '../../store/useAuthStore';

interface RoutePermissionGuardProps {
  permission?: string;
  children: React.ReactElement;
}

const RoutePermissionGuard: React.FC<RoutePermissionGuardProps> = ({ permission, children }) => {
  const { token, userInfo } = useAuthStore();
  // null = still loading, false = ready, true = profile fetch failed (retryable)
  const [profileLoadFailed, setProfileLoadFailed] = useState(false);
  const [retrySeq, setRetrySeq] = useState(0);

  useEffect(() => {
    if (!permission || !token || userInfo) {
      return;
    }
    let active = true;
    // Resolves null both when the profile request fails and when no session
    // exists; inside AuthGuard a session is guaranteed, so null means failure.
    void ensureAuthUserInfo().then((result) => {
      if (active && !result) {
        setProfileLoadFailed(true);
      }
    });
    return () => {
      active = false;
    };
  }, [permission, retrySeq, token, userInfo]);

  if (!permission) {
    return children;
  }

  if (!userInfo) {
    if (profileLoadFailed) {
      return (
        <PageError
          onRetry={() => {
            setProfileLoadFailed(false);
            setRetrySeq((seq) => seq + 1);
          }}
        />
      );
    }
    return <RouteContentFallback />;
  }

  return checkPermission(userInfo, permission) ? children : <PageForbidden />;
};

export default RoutePermissionGuard;
