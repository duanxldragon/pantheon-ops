import { useCallback, useEffect, useRef, useState } from 'react';

interface ListPage<T> {
  items: T[];
  total: number;
}

// Both list pagination and mutation callbacks use the current committed query.
// An activation owns its controller; refresh and polling share its pending GET.
export function useFiledistListQuery<T>(lookup: (signal: AbortSignal) => Promise<ListPage<T>>) {
  const [state, setState] = useState<{
    lookup: typeof lookup;
    items: T[];
    total: number;
    loading: boolean;
    error: unknown;
  }>();
  const currentRefresh = useRef<((silent: boolean) => Promise<void>) | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    let pending: Promise<void> | null = null;
    let queuedRefresh: Promise<void> | null = null;
    const refresh = (silent: boolean): Promise<void> => {
      if (controller.signal.aborted) return Promise.resolve();
      if (!silent) {
        setState((previous) => ({
          lookup,
          items: previous?.lookup === lookup ? previous.items : [],
          total: previous?.lookup === lookup ? previous.total : 0,
          loading: true,
          error: null,
        }));
      }
      if (pending) {
        // A mutation may finish after this GET began. Coalesce foreground
        // invalidations into one fresh read after it settles; polls only join.
        if (!silent) {
          queuedRefresh ??= pending.then(() => {
            queuedRefresh = null;
            return refresh(false);
          });
          return queuedRefresh;
        }
        return pending;
      }
      pending = Promise.resolve()
        .then(() => lookup(controller.signal))
        .then((response) => {
          if (!controller.signal.aborted) {
            setState({
              lookup,
              items: response.items || [],
              total: response.total ?? 0,
              loading: false,
              error: null,
            });
          }
        })
        .catch((error: unknown) => {
          if (!controller.signal.aborted) {
            setState((previous) => ({
              lookup,
              items: previous?.lookup === lookup ? previous.items : [],
              total: previous?.lookup === lookup ? previous.total : 0,
              loading: false,
              error,
            }));
          }
        })
        .finally(() => {
          pending = null;
        });
      return pending;
    };
    currentRefresh.current = refresh;
    const timer = setTimeout(() => void refresh(false), 0);
    return () => {
      clearTimeout(timer);
      controller.abort();
      if (currentRefresh.current === refresh) currentRefresh.current = null;
    };
  }, [lookup]);

  const refresh = useCallback(() => currentRefresh.current?.(false) ?? Promise.resolve(), []);
  const refreshSilently = useCallback(
    () => currentRefresh.current?.(true) ?? Promise.resolve(),
    [],
  );
  const visible =
    state?.lookup === lookup ? state : { items: [] as T[], total: 0, loading: true, error: null };
  return { ...visible, refresh, refreshSilently };
}
