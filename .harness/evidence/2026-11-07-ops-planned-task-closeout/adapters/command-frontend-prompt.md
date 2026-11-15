Read-only source review; no tools. Report proved P1/P2 defects with minimal fix, otherwise APPROVE supplied scope only. Max 220 words. This business O2 repair requires public HTTP viewed task versions. List/Get add stateVersion from taskCommandVersion(task, grouped task-level MAX(sequence)) after visibility/paging. Task schema is unchanged (datetime(3)); SHA includes task status/facts and task-level event sequence; log/host/step sequences excluded. targetTransactionService constructs a new service with transaction-bound CMDB/BizScope readers, asyncExecution=false and empty worker maps. Original dispatcher cancellation is captured under task lock and invoked post-commit; resume/retry use original dispatcher post-commit. Tests on real MySQL cover state matrix, same-version concurrency, ABA, scope before validation, context cancellation, chunked HTTP missing/stale/version, event/host failures, post-commit cancel. Omitted engine/lease/dependency/permission rules are unchanged. UI API errors expose numeric code/status. Alerts persist until next explicit action; 409 refresh does not replay; failed batch selections retained. Review only supplied source, not whole repository.
(id: number, stateVersion: string) {
return apiRequest<DeployTaskRow>({
url: `/business/deploy/tasks/${id}/cancel`,
method: 'post',
data: { stateVersion },
headers: { 'X-Idempotency-Key': crypto.randomUUID() },
});
}

function controlDeployTask(id: number, action: 'pause' | 'resume' | 'retry', stateVersion: string) {
return apiRequest<DeployTaskRow>({
url: `/business/deploy/tasks/${id}/${action}`,
method: 'post',
data: { stateVersion },
headers: { 'X-Idempotency-Key': crypto.randomUUID() },
});
}

export const pauseDeployTask = (id: number, stateVersion: string) => controlDeployTask(id, 'pause', stateVersion);
export const resumeDeployTask = (id: number, stateVersion: string) => controlDeployTask(id, 'resume', stateVersion);
export const retryDeployTask = (id: number, stateVersion: string) => controlDeployTask(id, 'retry', stateVersion);

export function isDeployTaskCommandConflict(error: unknown) {
if (!error || typeof error !== 'object') return false;
const value = error as { code?: number; status?: number };
return value.code === 409 || value.status === 409;
}

DeployTaskDetail.tsx
const handleCancelTask = useCallback(async () => {
if (!task) {
return;
}
setSubmitting(true);
try {
setCommandConflict(false);
await cancelDeployTask(task.id, task.stateVersion);
Message.success(t('business.deploy.task.cancelSuccess'));
await loadData();
} catch (error) {
if (!isDeployTaskCommandConflict(error)) throw error;
setCommandConflict(true);
await loadData();
} finally {
setSubmitting(false);
}
}, [loadData, t, task]);

const handleTaskControl = useCallback(
async (action: 'pause' | 'resume' | 'retry') => {
if (!task) return;
setSubmitting(true);
try {
const controls = {
pause: pauseDeployTask,
resume: resumeDeployTask,
retry: retryDeployTask,
};
setCommandConflict(false);
await controls[action](task.id, task.stateVersion);
Message.success(t(`business.deploy.task.${action}Success`));
await loadData();
} catch (error) {
if (!isDeployTaskCommandConflict(error)) throw error;
setCommandConflict(true);
await loadData();
} finally {
setSubmitting(false);
}
},
[loadData, t, task],
);

DeployTaskList.tsx
const handleCancel = async (task: DeployTaskRow) => {
setSubmitting(true);
setCommandFeedback(null);
try {
await cancelDeployTask(task.id, task.stateVersion);
Message.success(t('business.deploy.task.cancelSuccess'));
} catch (error) {
if (!isDeployTaskCommandConflict(error)) throw error;
setCommandFeedback('conflict');
} finally {
try { await loadData(); } finally { setSubmitting(false); }
}
};

const handleDelete = async (id: number) => {
await deleteDeployTask(id);
Message.success(t('common.deleteSuccess'));
setSelectedRowKeys((current) => current.filter((rowKey) => Number(rowKey) !== id));
await loadData();
};

const handleBatchCancel = async () => {
if (selectedRowKeys.length === 0) {
return;
}
setSubmitting(true);
try {
const candidates = data.filter(
(item) =>
selectedRowKeys.includes(item.id) && ['pending', 'running'].includes(item.status),
);
setCommandFeedback(null);
const results = await Promise.allSettled(candidates.map((item) => cancelDeployTask(item.id, item.stateVersion)));
const succeeded = candidates.filter((_, index) => results[index].status === 'fulfilled').map((item) => item.id);
setSelectedRowKeys((current) => current.filter((key) => !succeeded.includes(Number(key))));
if (results.some((result) => result.status === 'rejected')) {
setCommandFeedback('partial');
} else if (candidates.length > 0) {
Message.success(t('business.deploy.task.cancelSuccess'));
}
await loadData();
} finally {
setSubmitting(false);
}
};
