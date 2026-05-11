import { Suspense, lazy, type ReactElement, useEffect, useRef, useState } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Spin } from '@arco-design/web-react';
import { hasAuthCookie, useAuthStore } from './store/useAuthStore';
import { registeredModules } from './core/router/modules';
import { getRegisteredComponent } from './core/router/componentRegistry';
import RoutePermissionGuard from './core/router/RoutePermissionGuard';
import { ensureAuthUserInfo } from './core/auth/bootstrap';
import { scheduleHighFrequencyRouteWarmup } from './core/router/prefetch';
import { PageNotFound, RouteContentFallback } from './components';
import {
  handleVerifySuccess,
  handleVerifyCancel,
} from './components/feedback/secondaryVerifyController';
import { findFirstNavigableMenuPath } from './modules/system/menu/api';
import { useMenuStore } from './store/useMenuStore';

const BaseLayout = lazy(() => import('./core/layout'));
const LoginPage = lazy(() => import('./modules/auth/Login'));
const SecondaryVerifyModal = lazy(() =>
  import('./components/feedback/SecondaryVerifyModal').then((module) => ({
    default: module.SecondaryVerifyModal,
  })),
);

const AuthGuard = ({ children }: { children: ReactElement }) => {
  const { token } = useAuthStore();
  if (!token && !hasAuthCookie()) {
    return <Navigate to="/login" replace />;
  }
  return children;
};

function resolveDefaultAuthedPath(
  hasDashboardPermission: boolean,
  fallbackMenuPath: string | null,
) {
  if (hasDashboardPermission) {
    return '/dashboard';
  }
  if (fallbackMenuPath) {
    return fallbackMenuPath;
  }
  return '/dashboard';
}

const DefaultHomeRedirect = () => {
  const { token, userInfo } = useAuthStore();
  const { menuTree, loading, fetchMenuTree } = useMenuStore();

  useEffect(() => {
    if (!token || menuTree.length > 0 || loading) {
      return;
    }
    void fetchMenuTree();
  }, [fetchMenuTree, loading, menuTree.length, token]);

  if (!userInfo) {
    return <RouteContentFallback />;
  }

  const targetPath = resolveDefaultAuthedPath(
    Boolean(
      userInfo.roles?.includes('admin') || userInfo.perms?.includes('platform:dashboard:view'),
    ),
    findFirstNavigableMenuPath(menuTree),
  );

  if (
    loading &&
    menuTree.length === 0 &&
    targetPath === '/dashboard' &&
    !userInfo.perms?.includes('platform:dashboard:view') &&
    !userInfo.roles?.includes('admin')
  ) {
    return <RouteContentFallback />;
  }

  return <Navigate to={targetPath} replace />;
};

function App() {
  const [verifyVisible, setVerifyVisible] = useState(false);
  const warmupTokenRef = useRef<string | null>(null);
  const { token } = useAuthStore();
  const { menuTree, fetchMenuTree } = useMenuStore();

  useEffect(() => {
    const showHandler = () => setVerifyVisible(true);
    window.addEventListener('pantheon:show-verify-modal', showHandler);
    return () => window.removeEventListener('pantheon:show-verify-modal', showHandler);
  }, []);

  useEffect(() => {
    if (!token) {
      warmupTokenRef.current = null;
      return;
    }
    if (warmupTokenRef.current === token) {
      return;
    }
    warmupTokenRef.current = token;

    let active = true;
    void Promise.all([
      ensureAuthUserInfo().catch(() => null),
      menuTree.length > 0 ? Promise.resolve(menuTree) : fetchMenuTree(),
    ]).then(([userInfo, resolvedMenuTree]) => {
      if (!active) {
        return;
      }
      scheduleHighFrequencyRouteWarmup(
        {
          userInfo,
          menuTree: resolvedMenuTree,
        },
        {
          excludePaths: [window.location.pathname],
        },
      );
    });
    return () => {
      active = false;
    };
  }, [fetchMenuTree, menuTree, menuTree.length, token]);

  const handleSuccess = (token: string) => {
    setVerifyVisible(false);
    handleVerifySuccess(token);
  };

  const handleCancel = () => {
    setVerifyVisible(false);
    handleVerifyCancel();
  };

  const allRoutes = registeredModules.flatMap((m) => m.routes);

  return (
    <Suspense fallback={<Spin loading />}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <AuthGuard>
              <BaseLayout />
            </AuthGuard>
          }
        >
          <Route index element={<DefaultHomeRedirect />} />
          {allRoutes.map((route) => {
            const Component = route.component || getRegisteredComponent(route.componentKey);
            if (!Component) return null;
            return (
              <Route
                key={route.path}
                path={route.path}
                element={
                  <Suspense fallback={<RouteContentFallback />}>
                    <RoutePermissionGuard permission={route.pagePermission}>
                      <Component />
                    </RoutePermissionGuard>
                  </Suspense>
                }
              />
            );
          })}
        </Route>
        <Route path="*" element={<PageNotFound />} />
      </Routes>
      {verifyVisible ? (
        <SecondaryVerifyModal
          visible={verifyVisible}
          onSuccess={handleSuccess}
          onCancel={handleCancel}
        />
      ) : null}
    </Suspense>
  );
}

export default App;
