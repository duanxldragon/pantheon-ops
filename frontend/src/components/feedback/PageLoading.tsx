// PageLoading - Shell visual contract compliance
import { Spin } from '@arco-design/web-react';
import React from 'react';

export const PageLoading: React.FC = () => {
  return (
    <div className="page-loading">
      <Spin />
    </div>
  );
};

export {};
