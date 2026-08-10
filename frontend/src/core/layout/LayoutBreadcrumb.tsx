import React from 'react';
import { Breadcrumb } from '@arco-design/web-react';
import type { LayoutBreadcrumbItem } from './useLayoutMenu';

interface LayoutBreadcrumbProps {
  items: LayoutBreadcrumbItem[];
}

const LayoutBreadcrumb: React.FC<LayoutBreadcrumbProps> = ({ items }) => {
  return (
    <Breadcrumb className="app-shell__header-breadcrumb">
      {items.map((item) => (
        <Breadcrumb.Item key={`${item.path}-${item.label}`}>{item.label}</Breadcrumb.Item>
      ))}
    </Breadcrumb>
  );
};

export default LayoutBreadcrumb;
