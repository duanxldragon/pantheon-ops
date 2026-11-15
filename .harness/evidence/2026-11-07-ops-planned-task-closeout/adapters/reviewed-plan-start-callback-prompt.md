Read-only independent CODE follow-up. No tools. Max 150 words. Report proved P1/P2, otherwise APPROVE for this bounded change only. No release/project approval.
Follow-up to APPROVE with P2 onStarted rejection misreported as start failure. States accepted and postStartFailed start false. Once accepted, UI shows a success Alert; postStartFailed adds a refresh warning. ChangeDiff/plan hidden and confirm/refresh disabled when accepted. Callback type is void|Promise<void>. Unit verifies resolved start + rejected callback is never retried. Review supplied final logic, and whether the UI conditions address the finding.
const confirm = async () => {
if (!preview || submittingRef.current || new Date(preview.expiresAt).getTime() <= Date.now()) return;
submittingRef.current = true;
setSubmitting(true);
try {
try {
await startDeployTask(taskId, { ...payload, intentVersion: preview.intentVersion }, requestKey.current);
} catch (error) {
setPreview(null);
setState(isForbiddenRequestError(error) ? 'forbidden' : 'stale');
return;
}
setAccepted(true);
setPreview(null);
try { await onStarted(); } catch { setPostStartFailed(true); }
} finally { submittingRef.current = false; setSubmitting(false); }
};

RENDER CONDITIONS: const [accepted, setAccepted] = useState(false);
const [postStartFailed, setPostStartFailed] = useState(false);
{accepted && <Alert type="success" content={t('business.deploy.preview.startAccepted')} />}
{postStartFailed && <Alert type="warning" content={t('business.deploy.preview.postStartRefreshFailed')} />}
{!accepted && <ChangeDiff title={t('business.deploy.preview.changes')} state={state} labels={labels}
<Button disabled={submitting} onClick={onClose}>{t(accepted ? 'common.close' : 'common.cancel')}</Button>
<Button disabled={accepted || submitting || state === 'loading'} onClick={refresh}>{t('business.deploy.preview.refresh')}</Button>
<Button type="primary" loading={submitting} disabled={accepted || state !== 'ready' || !preview} onClick={() => void confirm()}>{t('business.deploy.preview.confirm')}</Button>
