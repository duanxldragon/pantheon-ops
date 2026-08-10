import React from 'react';
import { Space } from '@arco-design/web-react';

interface PageActionsProps {
  children: React.ReactNode;
  className?: string;
}

const PageActions: React.FC<PageActionsProps> = ({ children, className }) => (
  <Space size={8} className={className ? `page-actions ${className}` : 'page-actions'}>
    {children}
  </Space>
);

export default PageActions;
