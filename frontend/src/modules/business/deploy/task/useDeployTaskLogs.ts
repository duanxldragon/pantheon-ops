import { useEffect, useRef, useState } from 'react';
import { isForbiddenRequestError } from '../../../../api/request';
import { getDeployTaskLogs, type DeployTaskLogFilter, type DeployTaskLogRow } from '../api';
import type { OperationalDataState } from '../../../../components/operational';

interface LogSnapshot {
  sourceKey: string;
  rows: DeployTaskLogRow[];
  state: OperationalDataState;
  truncated: boolean;
}

export function useDeployTaskLogs(taskId: number, active: boolean, filter: DeployTaskLogFilter = {}) {
  const { hostId, stepId, attemptNo, firstError } = filter;
  const sourceKey = `${taskId}:${hostId ?? ''}:${attemptNo ?? ''}:${stepId ?? ''}:${firstError ? 1 : 0}`;
  const [snapshot, setSnapshot] = useState<LogSnapshot>({ sourceKey, rows: [], state: 'loading', truncated: false });
  const source = useRef({ sourceKey, cursor: 0 });
  const [reloadKey, setReloadKey] = useState(0);
  useEffect(() => {
    if (source.current.sourceKey !== sourceKey) source.current = { sourceKey, cursor: 0 };
    const session = source.current;
    let disposed = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const controller = new AbortController();
    const load = async () => {
      try {
        const response = await getDeployTaskLogs(taskId, session.cursor, controller.signal, { hostId, stepId, attemptNo, firstError });
        if (disposed) return;
        const incoming = response.items.filter((row) => Number.isSafeInteger(row.sequence) && row.sequence > session.cursor);
        if (!firstError) session.cursor = Math.max(session.cursor, ...incoming.map((row) => row.sequence));
        setSnapshot((previous) => {
          const sameSource = previous.sourceKey === sourceKey;
          const merged = [...(sameSource && !firstError ? previous.rows : []), ...incoming.map((row) => ({ ...row, content: row.content.slice(0, 16384) }))];
          return { sourceKey, rows: merged.slice(-1000), state: 'ready', truncated: (sameSource && previous.truncated) || !!response.truncated || merged.length > 1000 };
        });
        if (incoming.length === 500 || active) timer = setTimeout(() => { void load(); }, incoming.length === 500 ? 100 : 5000);
      } catch (error) {
        if (disposed) return;
        if (isForbiddenRequestError(error)) {
          session.cursor = 0;
          setSnapshot({ sourceKey, rows: [], state: 'forbidden', truncated: false });
          return;
        }
        setSnapshot((previous) => ({ sourceKey, rows: previous.sourceKey === sourceKey ? previous.rows : [],
          state: previous.sourceKey === sourceKey && previous.rows.length > 0 ? 'stale' : 'error', truncated: previous.sourceKey === sourceKey && previous.truncated }));
        if (active) timer = setTimeout(() => { void load(); }, 5000);
      }
    };
    void load();
    return () => { disposed = true; controller.abort(); if (timer) clearTimeout(timer); };
  }, [taskId, hostId, stepId, attemptNo, firstError, sourceKey, active, reloadKey]);
  return { rows: snapshot.sourceKey === sourceKey ? snapshot.rows : [],
    state: snapshot.sourceKey === sourceKey ? snapshot.state : 'loading' as OperationalDataState,
    truncated: snapshot.sourceKey === sourceKey && snapshot.truncated,
    reload: () => setReloadKey((value) => value + 1) };
}
