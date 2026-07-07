import React from 'react';

interface PageContainerProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  className?: string;
}

const PageContainer: React.FC<PageContainerProps> = ({ children, className, ...rest }) => (
  <div className={className ? `page-container ${className}` : 'page-container'} {...rest}>
    {children}
  </div>
);

export default PageContainer;
