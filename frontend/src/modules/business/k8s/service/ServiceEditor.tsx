import { Button, Input, InputNumber, Select, Space, Typography } from '@arco-design/web-react';
import { IconMinus, IconPlus } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import type { K8sServiceRow } from '../api';
import { emptyServicePort, type ServiceDraft } from './serviceDraft';
import './service-editor.css';

export default function ServiceEditor({
  draft,
  onChange,
  editing,
  disabled,
  validation,
}: {
  draft: ServiceDraft;
  onChange: (draft: ServiceDraft) => void;
  editing: K8sServiceRow | null;
  disabled: boolean;
  validation: string | null;
}) {
  const { t } = useTranslation();
  const updatePort = (index: number, patch: Partial<ServiceDraft['ports'][number]>) =>
    onChange({
      ...draft,
      ports: draft.ports.map((port, current) => (current === index ? { ...port, ...patch } : port)),
    });
  const fieldLabel = (id: string, key: string) => <label htmlFor={id}>{t(key)}</label>;
  return (
    <div className="k8s-service-editor" aria-busy={disabled}>
      {editing ? (
        <Typography.Paragraph>{t('business.k8s.service.selectorUpdateHint')}</Typography.Paragraph>
      ) : null}
      <div className="k8s-service-editor__grid">
        <div className="k8s-service-editor__field">
          {fieldLabel('k8s-service-name', 'business.k8s.service.field.name')}
          <Input
            id="k8s-service-name"
            value={draft.name}
            disabled={disabled || Boolean(editing)}
            onChange={(name) => onChange({ ...draft, name })}
          />
        </div>
        <div className="k8s-service-editor__field">
          {fieldLabel('k8s-service-namespace', 'business.k8s.service.field.namespace')}
          <Input
            id="k8s-service-namespace"
            value={draft.namespace}
            disabled={disabled || Boolean(editing)}
            onChange={(namespace) => onChange({ ...draft, namespace })}
          />
        </div>
      </div>
      <div className="k8s-service-editor__field">
        {fieldLabel('k8s-service-type', 'business.k8s.service.field.type')}
        <Select
          id="k8s-service-type"
          aria-label={t('business.k8s.service.field.type')}
          value={draft.type}
          disabled={disabled}
          onChange={(type) => onChange({ ...draft, type })}
        >
          {['ClusterIP', 'NodePort', 'LoadBalancer'].map((type) => (
            <Select.Option key={type} value={type}>
              {type}
            </Select.Option>
          ))}
        </Select>
      </div>
      <div className="k8s-service-editor__field">
        {fieldLabel('k8s-service-selector', 'business.k8s.service.field.selector')}
        <Input.TextArea
          id="k8s-service-selector"
          value={draft.selector}
          disabled={disabled}
          placeholder={t('business.k8s.service.selectorPlaceholder')}
          autoSize={{ minRows: 2, maxRows: 4 }}
          onChange={(selector) => onChange({ ...draft, selector })}
        />
        <Typography.Text type="secondary">{t('business.k8s.service.selectorHint')}</Typography.Text>
      </div>
      <div>
        <Typography.Text bold>{t('business.k8s.service.field.ports')}</Typography.Text>
        <Typography.Paragraph type="secondary">
          {t('business.k8s.service.portsHint')}
        </Typography.Paragraph>
        {draft.ports.map((port, index) => (
          <fieldset key={index} className="k8s-service-editor__port" disabled={disabled}>
            <legend>{t('business.k8s.service.portNumber', { number: index + 1 })}</legend>
            <div className="k8s-service-editor__grid">
              <div className="k8s-service-editor__field">
                {fieldLabel(`k8s-port-name-${index}`, 'business.k8s.service.field.portName')}
                <Input
                  id={`k8s-port-name-${index}`}
                  value={port.name}
                  disabled={disabled}
                  onChange={(name) => updatePort(index, { name })}
                />
              </div>
              <div className="k8s-service-editor__field">
                {fieldLabel(`k8s-port-${index}`, 'business.k8s.service.field.port')}
                <InputNumber
                  id={`k8s-port-${index}`}
                  min={1}
                  max={65535}
                  value={port.port}
                  disabled={disabled}
                  onChange={(value) => updatePort(index, { port: Number(value) })}
                />
              </div>
              <div className="k8s-service-editor__field">
                {fieldLabel(`k8s-target-port-${index}`, 'business.k8s.service.field.targetPort')}
                <Input
                  id={`k8s-target-port-${index}`}
                  value={port.targetPort}
                  disabled={disabled}
                  onChange={(targetPort) => updatePort(index, { targetPort })}
                />
              </div>
              <div className="k8s-service-editor__field">
                {fieldLabel(`k8s-protocol-${index}`, 'business.k8s.service.field.protocol')}
                <Select
                  id={`k8s-protocol-${index}`}
                  aria-label={t('business.k8s.service.field.protocol')}
                  value={port.protocol}
                  disabled={disabled}
                  onChange={(protocol) => updatePort(index, { protocol })}
                >
                  {['TCP', 'UDP', 'SCTP'].map((protocol) => (
                    <Select.Option key={protocol} value={protocol}>
                      {protocol}
                    </Select.Option>
                  ))}
                </Select>
              </div>
              {draft.type !== 'ClusterIP' ? (
                <div className="k8s-service-editor__field">
                  {fieldLabel(`k8s-node-port-${index}`, 'business.k8s.service.field.nodePort')}
                  <InputNumber
                    id={`k8s-node-port-${index}`}
                    min={1}
                    max={65535}
                    value={port.nodePort || undefined}
                    disabled={disabled}
                    onChange={(value) =>
                      updatePort(index, { nodePort: value == null ? undefined : Number(value) })
                    }
                  />
                </div>
              ) : null}
            </div>
            <Button
              type="text"
              size="small"
              icon={<IconMinus />}
              disabled={disabled || draft.ports.length === 1}
              aria-label={t('business.k8s.service.removePortNumber', { number: index + 1 })}
              onClick={() =>
                onChange({ ...draft, ports: draft.ports.filter((_, current) => current !== index) })
              }
            >
              {t('business.k8s.service.removePort')}
            </Button>
          </fieldset>
        ))}
        <Button
          type="secondary"
          icon={<IconPlus />}
          disabled={disabled}
          onClick={() => onChange({ ...draft, ports: [...draft.ports, emptyServicePort()] })}
        >
          {t('business.k8s.service.addPort')}
        </Button>
      </div>
      {editing ? (
        <Space wrap>
          <Typography.Text type="secondary">
            {t('business.k8s.service.field.resourceVersion')}
          </Typography.Text>
          <Typography.Text code>{editing.resourceVersion}</Typography.Text>
        </Space>
      ) : null}
      {validation ? (
        <Typography.Paragraph type="error" role="alert">
          {t(validation)}
        </Typography.Paragraph>
      ) : null}
    </div>
  );
}
