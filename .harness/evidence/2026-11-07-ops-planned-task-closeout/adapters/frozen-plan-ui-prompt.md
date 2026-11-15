Independent CODE review. Read-only, no tools. Supplied source only. Report proved P0/P1/P2 defects and minimal fix; otherwise APPROVE. Max 180 words. State scope limits. No release, production or project-completion approval. Current/frozen version binding was previously tested; do not infer missing helpers are defective.

FILE pantheon-ops\frontend\src\modules\business\deploy\task\DeployTaskPreview.tsx
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Collapse, Input, Pagination, Space, Typography } from '@arco-design/web-react';
import { AppModal, ChangeDiff } from '../../../../components';
import { isForbiddenRequestError } from '../../../../api/request';
import { previewDeployTask, startDeployTask, type DeployTaskPreviewPlan, type DeployTaskPreviewResponse, type StartDeployTaskPayload } from '../api';
import '@arco-design/web-react/es/Collapse/style/css.js';
import '../deploy.css';

interface Props {
taskId: number;
payload: StartDeployTaskPayload;
onClose: () => void;
onStarted: () => void | Promise<void>;
}
export default function DeployTaskPreview(props: Readonly<Props>) {
const { taskId, payload } = props;
const intentKey = JSON.stringify([taskId, payload.credentialRefId, payload.timeoutSeconds, payload.hostFingerprint]);
return <DeployTaskReview key={intentKey} {...props} />;
}

function DeployTaskReview({ taskId, payload, onClose, onStarted }: Readonly<Props>) {
const { t } = useTranslation();
const [preview, setPreview] = useState<DeployTaskPreviewResponse | null>(null);
const [state, setState] = useState<'loading' | 'ready' | 'error' | 'forbidden' | 'stale'>('loading');
const [generation, setGeneration] = useState(0);
const [keyword, setKeyword] = useState('');
const [page, setPage] = useState(1);
const [submitting, setSubmitting] = useState(false);
const submittingRef = useRef(false);
const requestKey = useRef(crypto.randomUUID());
useEffect(() => {
const controller = new AbortController();
let expiry: ReturnType<typeof setTimeout> | undefined;
void previewDeployTask(taskId, { credentialRefId: payload.credentialRefId, timeoutSeconds: payload.timeoutSeconds, hostFingerprint: payload.hostFingerprint }, controller.signal).then((result) => {
if (controller.signal.aborted) return;
const remaining = new Date(result.expiresAt).getTime() - Date.now();
if (!Number.isFinite(remaining) || remaining <= 0) { setState('stale'); return; }
setPreview(result);
setState('ready');
expiry = setTimeout(() => { setPreview(null); setState('stale'); }, remaining);
}).catch((error: unknown) => {
if (!controller.signal.aborted) setState(isForbiddenRequestError(error) ? 'forbidden' : 'error');
});
return () => { controller.abort(); if (expiry) clearTimeout(expiry); };
}, [taskId, payload.credentialRefId, payload.timeoutSeconds, payload.hostFingerprint, generation]);
const refresh = () => { setPreview(null); setState('loading'); setPage(1); requestKey.current = crypto.randomUUID(); setGeneration((value) => value + 1); };
const confirm = async () => {
if (!preview || submittingRef.current || new Date(preview.expiresAt).getTime() <= Date.now()) return;
submittingRef.current = true;
setSubmitting(true);
try {
await startDeployTask(taskId, { ...payload, intentVersion: preview.intentVersion }, requestKey.current);
await onStarted();
} catch (error) {
setPreview(null);
setState(isForbiddenRequestError(error) ? 'forbidden' : 'stale');
} finally { submittingRef.current = false; setSubmitting(false); }
};
const filteredChanges = (preview?.changes || []).filter((line) => `${line.field} ${line.before || ''} ${line.after || ''}`.toLocaleLowerCase().includes(keyword.toLocaleLowerCase()));
const labels = {
loading: t('common.loading'), empty: t('common.noData'), error: t('business.deploy.preview.loadFailed'),
forbidden: t('business.deploy.preview.forbidden'), stale: t('business.deploy.preview.stale'),
partial: t('business.deploy.preview.partial'), ready: t('business.deploy.preview.ready'),
kindLabels: { added: t('business.deploy.preview.added'), removed: t('business.deploy.preview.removed'), modified: t('business.deploy.preview.modified'), unchanged: t('business.deploy.preview.unchanged'), conflict: t('business.deploy.preview.conflict') },
before: t('business.deploy.preview.before'), after: t('business.deploy.preview.after'), sensitive: t('business.deploy.preview.sensitive'),
unchangedHidden: t('business.deploy.preview.unchangedHidden'), expandUnchanged: t('business.deploy.preview.expandUnchanged'), collapseUnchanged: t('business.deploy.preview.collapseUnchanged'),
};
return <AppModal title={t('business.deploy.preview.title')} visible size="detail" onCancel={submitting ? undefined : onClose} footer={null}>
<Space direction="vertical" size={16} style={{ width: '100%' }}>
{preview && <Typography.Paragraph>{t('business.deploy.preview.scopeSummary', { scope: preview.businessScopeName, count: preview.targetCount })}</Typography.Paragraph>}
{preview && <Input.Search value={keyword} aria-label={t('business.deploy.preview.search')} placeholder={t('business.deploy.preview.search')} allowClear onChange={(value) => { setKeyword(value); setPage(1); }} />}
<ChangeDiff title={t('business.deploy.preview.changes')} state={state} labels={labels}
lines={filteredChanges.slice((page - 1) * 100, page * 100).map((line) => ({ key: line.id, label: line.field, kind: line.kind, before: line.before || '—', after: line.after || '—' }))}
riskSummary={<Space direction="vertical" size={8}>{preview?.riskKeys.map((key) => <Alert key={key} type="warning" content={t(key)} />)}</Space>} />
{preview?.plan && <FrozenPlan plan={preview.plan} keyword={keyword} />}
{filteredChanges.length > 100 && <Pagination simple current={page} pageSize={100} total={filteredChanges.length} onChange={setPage} />}
<Space wrap>
<Button disabled={submitting} onClick={onClose}>{t('common.cancel')}</Button>
<Button disabled={submitting || state === 'loading'} onClick={refresh}>{t('business.deploy.preview.refresh')}</Button>
<Button type="primary" loading={submitting} disabled={state !== 'ready' || !preview} onClick={() => void confirm()}>{t('business.deploy.preview.confirm')}</Button>
</Space>
</Space>
</AppModal>;
}

function FrozenPlan({ plan, keyword }: Readonly<{ plan: DeployTaskPreviewPlan; keyword: string }>) {
const { t } = useTranslation();
const query = keyword.toLocaleLowerCase();
const matches = (value: string) => value.toLocaleLowerCase().includes(query);
const steps = plan.steps.flatMap((step) => {
const headerMatches = matches(`${step.name} ${step.action} ${step.packageName} ${step.packageVersion}`);
const parameters = headerMatches ? step.parameters : step.parameters.filter((parameter) => matches(`${parameter.key} ${parameter.hidden ? '' : parameter.value || ''}`));
return headerMatches || parameters.length ? [{ ...step, parameters }] : [];
});
return <section className="deploy-preview-plan" aria-label={t('business.deploy.preview.frozenPlan')}>
<Typography.Title heading={6}>{t('business.deploy.preview.frozenPlan')}</Typography.Title>
<Typography.Paragraph>{plan.templateId
? t('business.deploy.preview.templateSummary', { name: plan.templateName || '—', version: plan.templateVersion || '—' })
: t('business.deploy.preview.packagePlan')}</Typography.Paragraph>
<Typography.Paragraph type="secondary">{t('business.deploy.preview.parametersUnknown')}</Typography.Paragraph>
<Typography.Paragraph>{t('business.deploy.preview.planCounts', { steps: plan.stepCount, parameters: plan.parameterCount })}</Typography.Paragraph>
{plan.truncated && <Alert type="warning" content={t('business.deploy.preview.planTruncated')} />}
{steps.length === 0 ? <Typography.Paragraph>{t('common.noData')}</Typography.Paragraph> : <Collapse defaultActiveKey={[steps[0].id]}>
{steps.map((step) => <Collapse.Item key={step.id} name={step.id} header={step.name || t('business.deploy.preview.unnamedStep')}>
<Space direction="vertical" size={8} style={{ width: '100%' }}>
{step.packageName && <Typography.Text>{t('business.deploy.preview.packageSummary', { name: step.packageName, version: step.packageVersion || '—' })}</Typography.Text>}
<Typography.Text>{t('business.deploy.preview.stepAction', { action: t(`business.deploy.task.action.${step.action}`, { defaultValue: step.action || '—' }) })}</Typography.Text>
<Typography.Text>{t('business.deploy.preview.failurePolicy', { policy: t(`business.deploy.preview.policy.${step.failurePolicy || 'fail_fast'}`, { defaultValue: '—' }) })}</Typography.Text>
{step.parameters.length ? <dl className="deploy-preview-plan__parameters">
{step.parameters.map((parameter) => <div className="deploy-preview-plan__parameter" key={parameter.key}>
<dt>{parameter.key}</dt><dd>{parameter.hidden ? t('business.deploy.preview.sensitive') : parameter.value || '—'}</dd>
</div>)}
</dl> : <Typography.Text type="secondary">{t('business.deploy.preview.noParameters')}</Typography.Text>}
</Space>
</Collapse.Item>)}
</Collapse>}
  </section>;
}
