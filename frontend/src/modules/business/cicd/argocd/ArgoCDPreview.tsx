import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Input, Select, Space, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { AppModal, ChangeDiff } from '../../../../components';
import { isForbiddenRequestError, RequestError } from '../../../../api/request';
import { previewArgoCDApplication, rollbackArgoCDApplication, syncArgoCDApplication, type ArgoCDApplicationRow, type ArgoCDPreviewResponse, type ArgoCDPreviewRevision } from './api';

interface Props { application: ArgoCDApplicationRow; onClose: () => void; onAccepted: () => void | Promise<void> }
const prefix = 'business.cicd.argocd.preview';

export default function ArgoCDPreview(props: Readonly<Props>) {
  return <ApplicationReview key={props.application.id} {...props} />;
}

function ApplicationReview({ application, onClose, onAccepted }: Readonly<Props>) {
  const { t } = useTranslation();
  const [preview, setPreview] = useState<ArgoCDPreviewResponse | null>(null);
  const [history, setHistory] = useState<ArgoCDPreviewRevision[]>([]);
  const [selection, setSelection] = useState('sync');
  const [state, setState] = useState<'loading' | 'ready' | 'stale' | 'error' | 'forbidden'>('loading');
  const [generation, setGeneration] = useState(0);
  const [search, setSearch] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [accepted, setAccepted] = useState(false);
  const [refreshFailed, setRefreshFailed] = useState(false);
  const locked = useRef(false);
  const pending = useRef(false);
  const rollback = selection !== 'sync';
  const revision = rollback ? selection : application.sourceTargetRev;

  useEffect(() => {
    const controller = new AbortController();
    let expiry: ReturnType<typeof setTimeout> | undefined;
    void previewArgoCDApplication(application.id, { revision, rollback }, controller.signal).then((result) => {
      if (controller.signal.aborted) return;
      const remaining = new Date(result.expiresAt).getTime() - Date.now();
      if (!Number.isFinite(remaining) || remaining <= 0) { setState('stale'); return; }
      setPreview(result); setHistory(result.history); setState('ready');
      expiry = setTimeout(() => { if (!pending.current) { setPreview(null); setState('stale'); } }, remaining);
    }).catch((error: unknown) => {
      if (!controller.signal.aborted) setState(isForbiddenRequestError(error) ? 'forbidden' : 'error');
    });
    return () => { controller.abort(); if (expiry) clearTimeout(expiry); };
  }, [application.id, revision, rollback, generation]);

  const refresh = () => {
    if (locked.current || uncertain || accepted) return;
    setPreview(null); setState('loading'); setGeneration((value) => value + 1);
  };
  const chooseRevision = (value: string) => {
    if (locked.current || uncertain || accepted) return;
    setPreview(null); setState('loading'); setSelection(value);
  };
  const confirm = async () => {
    if (!preview || locked.current || accepted || (!uncertain && state !== 'ready')) return;
    if (!uncertain && new Date(preview.expiresAt).getTime() <= Date.now()) { setPreview(null); setState('stale'); return; }
    locked.current = true; pending.current = true; setSubmitting(true);
    try {
      try {
        const execute = rollback ? rollbackArgoCDApplication : syncArgoCDApplication;
        await execute(application.id, preview.targetVersion, preview.intentVersion);
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

  const labels = {
    loading: t('common.loading'), empty: t('common.noData'), error: t(`${prefix}.error`), forbidden: t(`${prefix}.forbidden`), stale: t(`${prefix}.stale`), partial: t(`${prefix}.partial`), ready: t(`${prefix}.ready`),
    before: t(`${prefix}.before`), after: t(`${prefix}.after`), sensitive: t(`${prefix}.sensitive`),
    unchangedHidden: t(`${prefix}.unchangedHidden`), expandUnchanged: t(`${prefix}.expandUnchanged`), collapseUnchanged: t(`${prefix}.collapseUnchanged`),
    kindLabels: { added: t(`${prefix}.added`), removed: t(`${prefix}.removed`), modified: t(`${prefix}.modified`), unchanged: t(`${prefix}.unchanged`), conflict: t(`${prefix}.conflict`) },
  };
  const matching = (preview?.changes ?? []).filter((change) => `${change.group}/${change.kind}/${change.namespace}/${change.name}`.toLowerCase().includes(search.toLowerCase()));
  return <AppModal visible size="detail" title={t(`${prefix}.title`, { name: application.name })} footer={null} onCancel={submitting ? undefined : onClose}>
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {accepted ? <Alert type="success" content={t(`${prefix}.accepted`)} /> : <>
        <label htmlFor="argocd-reviewed-revision">{t(`${prefix}.revision`)}</label>
        <Select id="argocd-reviewed-revision" aria-label={t(`${prefix}.revision`)} value={selection} onChange={chooseRevision} disabled={submitting || uncertain || state === 'loading'}
          options={[{ value: 'sync', label: t(`${prefix}.syncRevision`, { revision: application.sourceTargetRev }) }, ...history.map((entry) => ({ value: entry.revision, label: t(entry.current ? `${prefix}.currentRevision` : entry.rollbackAvailable ? `${prefix}.rollbackRevision` : `${prefix}.unsupportedRevision`, { revision: entry.revision }), disabled: entry.current || !entry.rollbackAvailable }))]} />
        {preview && <>
          <Typography.Paragraph>{t(`${prefix}.target`, { namespace: preview.namespace, count: preview.targetCount })}</Typography.Paragraph>
          <Typography.Paragraph style={{ overflowWrap: 'anywhere' }}>{t(`${prefix}.versions`, { current: preview.currentRevision || t(`${prefix}.unknown`), target: preview.targetVersion })}</Typography.Paragraph>
          {preview.historyTruncated && <Alert type="warning" content={t(`${prefix}.historyTruncated`)} />}
        </>}
        <Input.Search aria-label={t(`${prefix}.search`)} placeholder={t(`${prefix}.search`)} value={search} onChange={setSearch} allowClear />
        <ChangeDiff state={state} title={t(`${prefix}.changes`)} labels={labels} lines={matching.map((change) => ({ key: `${change.group}/${change.kind}/${change.namespace}/${change.name}`, label: `${change.kind} ${change.namespace}/${change.name}${change.change === 'retained' ? ` · ${t(`${prefix}.retained`)}` : ''}`, kind: change.change === 'retained' ? 'unchanged' : change.change, sensitive: change.valuesHidden, before: t(`${prefix}.sensitive`), after: t(`${prefix}.sensitive`) }))}
          riskSummary={<Space direction="vertical" size={8}>{preview?.riskKeys.map((key) => <Alert key={key} type="warning" content={t(key)} />)}</Space>} />
        {preview && <Typography.Paragraph>{t(rollback && preview.rollbackAvailable ? `${prefix}.rollbackReviewed` : `${prefix}.rollbackSeparate`)}</Typography.Paragraph>}
      </>}
      {uncertain && <Alert type="warning" content={t(`${prefix}.uncertain`)} />}
      {refreshFailed && <Alert type="warning" content={t(`${prefix}.refreshFailed`)} />}
      <Space wrap>
        <Button disabled={submitting} onClick={onClose}>{t(accepted || uncertain ? 'common.close' : 'common.cancel')}</Button>
        <Button disabled={accepted || submitting || uncertain || state === 'loading'} onClick={refresh}>{t(`${prefix}.refresh`)}</Button>
        <Button type="primary" loading={submitting} disabled={accepted || (!uncertain && state !== 'ready') || !preview} onClick={() => void confirm()}>{t(uncertain ? `${prefix}.retrySame` : rollback ? `${prefix}.confirmRollback` : `${prefix}.confirm`)}</Button>
      </Space>
    </Space>
  </AppModal>;
}
