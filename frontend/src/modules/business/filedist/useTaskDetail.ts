import { useCallback, useEffect, useState } from 'react';
import { getTaskDetail, getTaskEvents, getTaskTargets } from './api';
import type { DistributionEventRow, DistributionTargetRow, TaskDetailResponse } from './api';
import { isTaskActive } from './taskState';

interface DetailState {
  taskId: number;
  task: TaskDetailResponse | null;
  targets: DistributionTargetRow[];
  events: DistributionEventRow[];
  loading: boolean;
  error: unknown;
  refreshError: unknown;
}

const initialState = (taskId: number): DetailState => ({
  taskId,
  task: null,
  targets: [],
  events: [],
  loading: true,
  error: null,
  refreshError: null,
});

export function useTaskDetail(taskId: number) {
  const [state, setState] = useState<DetailState>(() => initialState(taskId));
  const [revision, setRevision] = useState(0);
  const refresh = useCallback(() => setRevision((value) => value + 1), []);

  useEffect(() => {
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    const load = async (background = false) => {
      if (!Number.isSafeInteger(taskId) || taskId <= 0) {
        setState({
          ...initialState(taskId),
          loading: false,
          error: new Error('business.filedist.taskMissing'),
        });
        return;
      }
      if (!background)
        setState((previous) =>
          Object.is(previous.taskId, taskId)
            ? { ...previous, loading: !previous.task, error: null, refreshError: null }
            : initialState(taskId),
        );
      let active = true;
      try {
        let failed = false;
        let firstError: unknown;
        const observe = <T>(request: Promise<T>) =>
          request.catch((error: unknown) => {
            if (!failed) {
              failed = true;
              firstError = error;
            }
            throw error;
          });
        const [taskResult, targetResult, eventResult] = await Promise.allSettled([
          observe(getTaskDetail(taskId, controller.signal)),
          observe(getTaskTargets(taskId, controller.signal)),
          observe(getTaskEvents(taskId, controller.signal)),
        ]);
        if (controller.signal.aborted) return;
        if (failed) throw firstError;
        if (
          taskResult.status !== 'fulfilled' ||
          targetResult.status !== 'fulfilled' ||
          eventResult.status !== 'fulfilled'
        )
          return;
        const task = taskResult.value,
          targets = targetResult.value,
          events = eventResult.value;
        active = isTaskActive(task.status);
        setState({
          taskId,
          task,
          targets: targets.items || [],
          events: events || [],
          loading: false,
          error: null,
          refreshError: null,
        });
      } catch (error) {
        if (controller.signal.aborted) return;
        setState((previous) =>
          Object.is(previous.taskId, taskId) && previous.task
            ? { ...previous, loading: false, refreshError: error }
            : { ...initialState(taskId), loading: false, error },
        );
      }
      // Wait until this request settles before starting the five-second timer.
      if (!controller.signal.aborted && active) timer = setTimeout(() => void load(true), 5000);
    };
    timer = setTimeout(() => void load(), 0);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [taskId, revision]);

  return { ...(Object.is(state.taskId, taskId) ? state : initialState(taskId)), refresh };
}
