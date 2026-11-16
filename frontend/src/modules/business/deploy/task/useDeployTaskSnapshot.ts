import { useCallback, useEffect, useRef, useState } from 'react';
import { isForbiddenRequestError, RequestError } from '../../../../api/request';
import { getDeployTaskDetail, type DeployTaskRow } from '../api';

interface Snapshot {
  id?: number;
  task: DeployTaskRow | null;
  error: unknown;
  loading: boolean;
  stale: boolean;
}

// A departed request/stream generation cannot clear or replace the new task.
export function useDeployTaskSnapshot(id: number | undefined) {
  const [snapshot, setSnapshot] = useState<Snapshot>({ id, task: null, error: null, loading: true, stale: false });
  const generation = useRef<{ id?: number; disposed: boolean; denied: boolean; request: number; running: boolean; pending: boolean } | null>(null);
  const loadData = useCallback(async function loadSnapshot(options?: { silent?: boolean }) {
    const session = generation.current;
    if (!id || !session || session.id !== id || session.disposed || session.denied) return;
    if (session.running) { session.pending = true; return; }
    session.pending = false;
    session.running = true;
    const request = ++session.request;
    if (!options?.silent) setSnapshot((previous) => ({ ...previous, id, loading: true, error: null }));
    try {
      const task = await getDeployTaskDetail(id);
      if (session.disposed || session.request !== request) return;
      setSnapshot({ id, task, error: null, loading: false, stale: false });
    } catch (error) {
      if (session.disposed || session.request !== request) return;
      if (isForbiddenRequestError(error)) { session.denied = true; session.pending = false; }
      if (isForbiddenRequestError(error) || !options?.silent) {
        setSnapshot({ id, task: null, error, loading: false, stale: false });
      } else {
        setSnapshot((previous) => ({ ...previous, loading: false, stale: true }));
      }
    } finally {
      session.running = false;
      // Keep the last event received during this request, without overlapping reads.
      if (session.pending && !session.disposed && !session.denied && session.request === request && generation.current === session) {
        void loadSnapshot({ silent: true });
      }
    }
  }, [id]);
  const deny = useCallback(() => {
    const session = generation.current;
    if (!session || session.id !== id || session.disposed) return;
    // Fence any in-flight successful refresh after an explicit stream denial.
    ++session.request;
    session.denied = true;
    session.pending = false;
    setSnapshot({ id, task: null, error: new RequestError({ kind: 'forbidden', status: 403, code: 403 }), loading: false, stale: false });
  }, [id]);
  useEffect(() => {
    const session = { id, disposed: false, denied: false, request: 0, running: false, pending: false };
    generation.current = session;
    queueMicrotask(() => { if (!session.disposed) void loadData(); });
    return () => { session.disposed = true; session.pending = false; };
  }, [id, loadData]);
  const current = snapshot.id === id ? snapshot : { task: null, error: null, loading: true, stale: false };
  return { ...current, loadData, deny };
}
