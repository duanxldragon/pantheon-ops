// PageError - Shell visual contract compliance
import { Result } from '@arco-design/web-react';
import React from 'react';

export interface PageErrorProps {
  message?: string;
}

export const PageError: React.FC<PageErrorProps> = ({ message = '加载失败' }) => {
  return (
    <div className="page-result">
      <Result status="error" title={message} />
    </div>
  );
};

export {};
