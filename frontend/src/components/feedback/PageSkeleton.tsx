import React from 'react';
import { Skeleton } from '@arco-design/web-react';

interface PageSkeletonProps {
  /** Number of skeleton rows for the list/table variant. */
  rows?: number;
  /** `table` mimics a list page; `form` mimics a detail/edit form. */
  variant?: 'table' | 'form';
}

/**
 * Page-level skeleton placeholder. A structured alternative to a bare `Spin`
 * for first-paint of tables and forms, reducing perceived latency without
 * faking content. Pair with the page's loading flag.
 */
const PageSkeleton: React.FC<PageSkeletonProps> = ({ rows = 6, variant = 'table' }) => (
  <div className="page-skeleton" aria-busy="true" aria-live="polite">
    <Skeleton
      animation
      text={{ rows: variant === 'form' ? Math.max(4, rows) : rows, width: '100%' }}
      image={variant === 'form'}
    />
  </div>
);

export default PageSkeleton;
