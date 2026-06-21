import React from 'react';
import { isNetworkRequestError, isServerRequestError, isTimeoutRequestError } from '../../api/request';
import PageError from './PageError';
import PageNetworkError from './PageNetworkError';
import PageServerError from './PageServerError';

interface PageRequestErrorProps {
  error: unknown;
  onRetry?: () => void;
  description?: React.ReactNode;
}

const PageRequestError: React.FC<PageRequestErrorProps> = ({ error, onRetry, description }) => {
  if (isNetworkRequestError(error)) {
    return <PageNetworkError timeout={isTimeoutRequestError(error)} onRetry={onRetry} />;
  }

  if (isServerRequestError(error)) {
    return <PageServerError onRetry={onRetry} />;
  }

  return <PageError description={description} onRetry={onRetry} />;
};

export default PageRequestError;
