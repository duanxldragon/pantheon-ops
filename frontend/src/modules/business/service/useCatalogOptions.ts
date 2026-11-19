import { useEffect, useMemo, useState } from 'react';

// Form-only, bounded owner queries. Abort/ignore superseded lookups so a prior
// service or namespace cannot publish candidates into a new target context.
export function useCatalogOptions<T>(enabled: boolean, lookup: (signal: AbortSignal) => Promise<T[]>) {
  // A fresh activation must not reuse rows or denial from the previous opening,
  // even when its lookup callback is unchanged during the debounce interval.
  const request = useMemo(() => ({ enabled, lookup }), [enabled, lookup]);
  const [state, setState] = useState<{ request: typeof request; rows: T[]; error: unknown; loading: boolean }>();
  useEffect(() => {
    if (!request.enabled) return undefined;
    const controller = new AbortController();
    const timer = setTimeout(() => {
      setState({ request, rows: [], error: null, loading: true });
      void request.lookup(controller.signal).then((rows) => {
        if (!controller.signal.aborted) setState({ request, rows, error: null, loading: false });
      }).catch((error: unknown) => {
        if (!controller.signal.aborted) setState({ request, rows: [], error, loading: false });
      });
    }, 250);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [request]);
  return enabled && state?.request === request ? state : { rows: [] as T[], error: null, loading: enabled };
}
