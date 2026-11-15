import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Input, Space, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { AppModal, ChangeDiff } from '../../../../components';
import { isForbiddenRequestError, RequestError } from '../../../../api/request';
import { createK8sRelease, previewK8sRelease, previewK8sRollback, rollbackK8sRelease, type K8sReleasePayload, type K8sReleasePreviewResponse } from '../api';

type Intent = { operation: 'create'; payload: K8sReleasePayload } | { operation: 'rollback'; releaseId: number };
interface Props { intent: Intent; onClose: (returnToForm: boolean) => void; onAccepted: () => void | Promise<void> }

export default function ReleasePreview(props: Readonly<Props>) {
 return <ReleaseReview key={JSON.stringify(props.intent)} {...props} />;
}

function ReleaseReview({ intent, onClose, onAccepted }: Readonly<Props>) {
 const { t } = useTranslation();
 const [preview, setPreview] = useState<K8sReleasePreviewResponse | null>(null);
 const [state, setState] = useState<'loading' | 'ready' | 'stale' | 'error' | 'forbidden'>('loading');
 const [generation, setGeneration] = useState(0);
 const [submitting, setSubmitting] = useState(false);
 const [uncertain, setUncertain] = useState(false);
 const [accepted, setAccepted] = useState(false);
 const [refreshFailed, setRefreshFailed] = useState(false);
 const [search, setSearch] = useState('');
 const locked = useRef(false);
 const pending = useRef(false);
 const requestKey = useRef(crypto.randomUUID());
 useEffect(() => {
  const controller = new AbortController();
  let expiry: ReturnType<typeof setTimeout> | undefined;
  const load = intent.operation === 'create' ? previewK8sRelease({ ...intent.payload, idempotencyKey: requestKey.current }, controller.signal)
   : previewK8sRollback(intent.releaseId, requestKey.current, controller.signal);
  void load.then((result) => {
   if (controller.signal.aborted) return;
   const remaining = new Date(result.expiresAt).getTime() - Date.now();
   if (!Number.isFinite(remaining) || remaining <= 0) { setState('stale'); return; }
   setPreview(result); setState('ready');
   expiry = setTimeout(() => { if (!pending.current) { setPreview(null); setState('stale'); } }, remaining);
  }).catch((error: unknown) => { if (!controller.signal.aborted) setState(isForbiddenRequestError(error) ? 'forbidden' : 'error'); });
  return () => { controller.abort(); if (expiry) clearTimeout(expiry); };
 }, [intent, generation]);
 const refresh = () => {
  if (locked.current || uncertain || accepted) return;
  setPreview(null); setState('loading'); requestKey.current = crypto.randomUUID(); setGeneration((value) => value + 1);
 };
 const confirm = async () => {
  if (!preview || locked.current || accepted) return;
  if (!uncertain && new Date(preview.expiresAt).getTime() <= Date.now()) { setPreview(null); setState('stale'); return; }
  locked.current = true; pending.current = true; setSubmitting(true);
  try {
   try {
    if (intent.operation === 'create') await createK8sRelease({ ...intent.payload, idempotencyKey: requestKey.current, intentVersion: preview.intentVersion });
    else await rollbackK8sRelease(intent.releaseId, requestKey.current, preview.intentVersion);
   } catch (error) {
    if (isForbiddenRequestError(error)) { pending.current = false; setUncertain(false); setPreview(null); setState('forbidden'); }
    else if (error instanceof RequestError && (error.code === 409 || error.status === 409)) { pending.current = false; setUncertain(false); setPreview(null); setState('stale'); }
    else { setUncertain(true); }
    return;
   }
   setAccepted(true); setUncertain(false); pending.current = false; setPreview(null);
   try { await onAccepted(); } catch { setRefreshFailed(true); }
  } finally { locked.current = false; setSubmitting(false); }
 };
 const prefix = 'business.k8s.release.preview';
 const labels = {
  loading: t('common.loading'), empty: t('common.noData'), error: t(`${prefix}.error`), forbidden: t(`${prefix}.forbidden`), stale: t(`${prefix}.stale`), partial: t(`${prefix}.partial`), ready: t(`${prefix}.ready`),
  before: t(`${prefix}.before`), after: t(`${prefix}.after`), sensitive: t(`${prefix}.sensitive`),
  unchangedHidden: t(`${prefix}.unchangedHidden`), expandUnchanged: t(`${prefix}.expandUnchanged`), collapseUnchanged: t(`${prefix}.collapseUnchanged`),
  kindLabels: { added: t(`${prefix}.added`), removed: t(`${prefix}.removed`), modified: t(`${prefix}.modified`), unchanged: t(`${prefix}.unchanged`), conflict: t(`${prefix}.conflict`) },
 };
 return <AppModal visible size="detail" title={t(`${prefix}.title`)} footer={null} onCancel={submitting ? undefined : () => onClose(!accepted && !uncertain)}>
  <Space direction="vertical" size={16} style={{ width: '100%' }}>
   {accepted ? <Alert type="success" content={t(`${prefix}.accepted`)} /> : <>
    {preview && <Typography.Paragraph>{t(`${prefix}.target`, { environment: preview.environment, namespace: preview.namespace, workload: preview.workloadName, count: preview.targetCount })}</Typography.Paragraph>}
    <Input.Search aria-label={t(`${prefix}.search`)} placeholder={t(`${prefix}.search`)} value={search} onChange={setSearch} allowClear />
    <ChangeDiff state={state} title={t(`${prefix}.changes`)} labels={labels} lines={(preview?.changes ?? []).filter((line) => t(`${prefix}.field.${line.field}`).toLowerCase().includes(search.toLowerCase())).map((line) => ({ key: line.id, label: t(`${prefix}.field.${line.field}`), kind: line.kind, before: line.before === '[hidden]' ? t(`${prefix}.sensitive`) : line.before, after: line.after }))}
     riskSummary={<Space direction="vertical" size={8}>{preview?.riskKeys.map((key) => <Alert key={key} type="warning" content={t(key)} />)}</Space>} />
    {preview && <Typography.Paragraph>{t(preview.rollbackAvailable ? `${prefix}.rollbackImage` : `${prefix}.rollbackUnavailable`)}</Typography.Paragraph>}
   </>}
   {uncertain && <Alert type="warning" content={t(`${prefix}.uncertain`)} />}
   {refreshFailed && <Alert type="warning" content={t(`${prefix}.refreshFailed`)} />}
   <Space wrap>
    <Button disabled={submitting} onClick={() => onClose(!accepted && !uncertain)}>{t(accepted || uncertain ? 'common.close' : 'common.cancel')}</Button>
    <Button disabled={accepted || submitting || uncertain || state === 'loading'} onClick={refresh}>{t(`${prefix}.refresh`)}</Button>
    <Button type="primary" loading={submitting} disabled={accepted || (!uncertain && state !== 'ready') || !preview} onClick={() => void confirm()}>{t(uncertain ? `${prefix}.retrySame` : `${prefix}.confirm`)}</Button>
   </Space>
  </Space>
 </AppModal>;
}
