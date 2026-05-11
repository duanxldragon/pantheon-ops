import React from 'react';
import { Spin } from '@arco-design/web-react';

const PageLoading: React.FC = () => (
  <div className="page-loading">
    <Spin loading />
  </div>
);

export default PageLoading;
