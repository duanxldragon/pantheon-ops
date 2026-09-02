// PageEmpty - Shell visual contract compliance
import React from 'react';

export interface PageEmptyProps {
  description?: string;
}

export const PageEmpty: React.FC<PageEmptyProps> = ({ description = '暂无数据' }) => {
  return (
    <div className="page-empty">
      <div className="page-empty__inner">
        <p>{description}</p>
      </div>
    </div>
  );
};

export {};
