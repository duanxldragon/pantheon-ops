// AppDrawer - Shell visual contract compliance
import { Drawer, type DrawerProps } from '@arco-design/web-react';
import React from 'react';

export interface AppDrawerProps extends DrawerProps {
  children?: React.ReactNode;
}

export const AppDrawer: React.FC<AppDrawerProps> = ({ wrapClassName, maskClosable = false, ...props }) => {
  return (
    <Drawer
      wrapClassName={`app-drawer ${wrapClassName || ''}`}
      maskClosable={maskClosable}
      {...props}
    />
  );
};

export {};
