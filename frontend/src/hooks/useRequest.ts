import { useCallback, useEffect, useRef, useState } from 'react';
import type { RequestConfig } from '../api/request';

export interface UseRequestOptions<T> {
  manual?: boolean;
  defaultParams?: unknown[];
  onSuccess?: (data: T, params: unknown[]) => void;
  onError?: (error: unknown, params: unknown[]) => void;
  onFinally?: (params: unknown[]) => void;
}

export interface UseRequestResult<T> {
  data: T | undefined;
  loading: boolean;
  error: unknown;
  run: (config?: RequestConfig) => Promise<T | undefined>;
  runAsync: (config?: RequestConfig) => Promise<T>;
  refresh: () => Promise<T | undefined>;
  reset: () => void;
  mutate: (data: T | ((prev: T | undefined) => T)) => void;
}

export function useRequest<T>(
  service: (config?: RequestConfig) => Promise<T>,
  options: UseRequestOptions<T> = {},
): UseRequestResult<T> {
  const { manual = false, defaultParams = [], onSuccess, onError, onFinally } = options;

  const [data, setData] = useState<T | undefined>(undefined);
  const [loading, setLoading] = useState(!manual);
  const [error, setError] = useState<unknown>(null);

  const serviceRef = useRef(service);

  useEffect(() => {
    serviceRef.current = service;
  }, [service]);

  const run = useCallback(
    async (config?: RequestConfig): Promise<T | undefined> => {
      setLoading(true);
      setError(null);
      try {
        const result = await serviceRef.current(config);
        setData(result);
        onSuccess?.(result, defaultParams);
        return result;
      } catch (err) {
        setError(err);
        onError?.(err, defaultParams);
        return undefined;
      } finally {
        setLoading(false);
        onFinally?.(defaultParams);
      }
    },
    [defaultParams, onSuccess, onError, onFinally],
  );

  const runAsync = useCallback(
    async (config?: RequestConfig): Promise<T> => {
      setLoading(true);
      setError(null);
      try {
        const result = await serviceRef.current(config);
        setData(result);
        onSuccess?.(result, defaultParams);
        return result;
      } catch (err) {
        setError(err);
        onError?.(err, defaultParams);
        throw err;
      } finally {
        setLoading(false);
        onFinally?.(defaultParams);
      }
    },
    [defaultParams, onSuccess, onError, onFinally],
  );

  const refresh = useCallback(async (): Promise<T | undefined> => {
    return run();
  }, [run]);

  const reset = useCallback(() => {
    setData(undefined);
    setLoading(!manual);
    setError(null);
  }, [manual]);

  const mutate = useCallback((newData: T | ((prev: T | undefined) => T)) => {
    setData((prev) =>
      typeof newData === 'function' ? (newData as (prev: T | undefined) => T)(prev) : newData,
    );
  }, []);

  return {
    data,
    loading,
    error,
    run,
    runAsync,
    refresh,
    reset,
    mutate,
  };
}
