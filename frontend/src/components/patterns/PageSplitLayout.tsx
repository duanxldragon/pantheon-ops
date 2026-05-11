import React from 'react';

interface PageSplitLayoutProps {
  children: React.ReactNode;
  rail?: React.ReactNode;
  className?: string;
  mainClassName?: string;
  railClassName?: string;
}

const PageSplitLayout: React.FC<PageSplitLayoutProps> = ({
  children,
  rail,
  className,
  mainClassName,
  railClassName,
}) => (
  <div
    className={[
      'page-split-layout',
      rail ? 'page-split-layout--with-rail' : 'page-split-layout--single',
      className,
    ]
      .filter(Boolean)
      .join(' ')}
  >
    <div className={mainClassName ? `page-main-column ${mainClassName}` : 'page-main-column'}>
      {children}
    </div>
    {rail ? (
      <aside className={railClassName ? `page-side-column ${railClassName}` : 'page-side-column'}>
        {rail}
      </aside>
    ) : null}
  </div>
);

export default PageSplitLayout;
