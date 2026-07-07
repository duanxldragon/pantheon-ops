import React from 'react';
import { Drawer, type DrawerProps } from '@arco-design/web-react';

type AppDrawerSize = 'sm' | 'md' | 'lg' | 'xl' | 'detail';

interface AppDrawerProps extends DrawerProps {
  size?: AppDrawerSize;
}

const sizeWidthMap: Record<AppDrawerSize, number> = {
  sm: 560,
  md: 720,
  lg: 860,
  xl: 1000,
  detail: 960,
};

const AppDrawer: React.FC<AppDrawerProps> = ({
  className,
  size = 'xl',
  width,
  maskClosable = false,
  unmountOnExit = true,
  ...rest
}) => (
  <Drawer
    className={className ? `app-drawer ${className}` : 'app-drawer'}
    width={width ?? sizeWidthMap[size]}
    maskClosable={maskClosable}
    unmountOnExit={unmountOnExit}
    {...rest}
  />
);

export default AppDrawer;
