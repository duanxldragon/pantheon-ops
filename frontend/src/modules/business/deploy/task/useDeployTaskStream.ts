import { useEffect, useRef, useState } from 'react';
import { getDeployTaskEventStreamUrl } from '../api';

// The browser retains Last-Event-ID while this EventSource reconnects. A task
// snapshot update must not recreate the connection or discard that cursor.
export function useDeployTaskStream(taskId: number | undefined, active: boolean, refresh: () => void, onForbidden?: () => void) {
  const [connected, setConnected] = useState(false);
  const callback = useRef(refresh);
  const forbidden = useRef(onForbidden);
  useEffect(() => { forbidden.current = onForbidden; }, [onForbidden]);
  useEffect(() => { callback.current = refresh; }, [refresh]);
  useEffect(() => {
    if (!taskId || !active) return;
    let disposed = false;
    const source = new EventSource(getDeployTaskEventStreamUrl(taskId), { withCredentials: true });
    source.onopen = () => { if (!disposed) setConnected(true); };
    source.addEventListener('forbidden', () => {
      if (disposed) return;
      disposed = true;
      source.close();
      setConnected(false);
      forbidden.current?.();
    });
    source.addEventListener('task', () => {
      if (disposed) return;
      setConnected(true);
      callback.current();
    });
    source.onerror = () => { if (!disposed) setConnected(false); };
    return () => { disposed = true; source.close(); setConnected(false); };
  }, [taskId, active]);
  return connected;
}
