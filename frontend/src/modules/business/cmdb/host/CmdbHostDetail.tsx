import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Descriptions,
  Tag,
  Space,
  Button,
  Form,
  Input,
  Select,
  Message,
  Typography,
} from '@arco-design/web-react';
import { IconLeft, IconCode } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import FormSection from '../../../../components/patterns/FormSection';
import SubmitBar from '../../../../components/patterns/SubmitBar';
import { getHostDetail, collectHostConfig } from './api';
import type { HostRow } from './api';
import { usePermission } from '../../../../hooks/usePermission';
import '../../../system/list-page.css';
import '../cmdb.css';

export default function CmdbHostDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [searchParams] = useSearchParams();

  const [host, setHost] = useState<HostRow | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [collectVisible, setCollectVisible] = useState(searchParams.get('collect') === '1');
  const [collecting, setCollecting] = useState(false);
  const [collectForm] = Form.useForm();

  const canCollect = hasPerm('business:cmdb:host:collect');

  const hostStats = useMemo(
    () =>
      host
        ? [
            {
              key: 'status',
              label: t('business.cmdb.host.status'),
              value: t(`business.cmdb.host.status.${host.status}`),
              hint: t('business.cmdb.host.hero.statusHint'),
            },
            {
              key: 'os',
              label: t('business.cmdb.host.os'),
              value: t(`business.cmdb.host.os.${host.os}`),
              hint: t('business.cmdb.host.hero.osHint'),
            },
            {
              key: 'labels',
              label: t('business.cmdb.host.labels'),
              value: host.labelValues?.length || 0,
              hint: t('business.cmdb.host.hero.labelsHint'),
            },
            {
              key: 'components',
              label: t('business.cmdb.host.installedComponents'),
              value: host.installedComponents?.length || 0,
              hint: t('business.cmdb.host.hero.componentsHint'),
            },
          ]
        : [],
    [host, t],
  );

  const loadDetail = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const result = await getHostDetail(Number(id));
      setHost(result);
    } catch (err) {
      setError(err);
      setHost(null);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadDetail();
    });
  }, [loadDetail]);

  const handleCollect = async () => {
    if (!id) return;
    const values = await collectForm.validate();
    setCollecting(true);
    try {
      const result = await collectHostConfig(Number(id), values);
      setHost(result);
      setCollectVisible(false);
      Message.success(t('business.cmdb.host.collectSuccess'));
    } catch {
      // handled by request interceptor
    } finally {
      setCollecting(false);
    }
  };

  if (loading) {
    return (
      <PageContainer>
        <PageLoading />
      </PageContainer>
    );
  }

  if (error || !host) {
    return (
      <PageContainer>
        <PageHeader title={t('operations.cmdb.host.detail')} />
        <PageError description={t('common.loadFailedDesc')} onRetry={loadDetail} />
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <PageHeader
        title={host.hostname}
        extra={
          <Space>
            {canCollect && host.os === 'linux' && (
              <Button icon={<IconCode />} onClick={() => setCollectVisible(true)}>
                {t('business.cmdb.host.collect')}
              </Button>
            )}
            <Button icon={<IconLeft />} onClick={() => navigate('/operations/cmdb/host')}>
              {t('common.back')}
            </Button>
          </Space>
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero cmdb-page__hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">
                {t('business.cmdb.host.hero.eyebrow')}
              </span>
              <Typography.Title heading={5} className="system-page-hero__title cmdb-page__hero-title">
                {host.hostname}
              </Typography.Title>
            </div>
          </div>
          <div className="cmdb-page__hero-grid">
            {hostStats.map((item) => (
              <div key={item.key} className="cmdb-page__hero-metric">
                <span className="cmdb-page__hero-label">{item.label}</span>
                <span className="cmdb-page__hero-value">{item.value}</span>
                <span className="cmdb-page__hero-hint">{item.hint}</span>
              </div>
            ))}
          </div>
        </Card>
        <Card className="page-panel">
          <FormSection title={t('business.cmdb.host.baseInfo')}>
            <Descriptions
              column={2}
              data={[
                { label: t('business.cmdb.host.hostname'), value: host.hostname },
                { label: t('business.cmdb.host.ip'), value: host.ip },
                { label: t('business.cmdb.host.sshPort'), value: host.sshPort || 22 },
                { label: t('business.cmdb.host.os'), value: t(`business.cmdb.host.os.${host.os}`) },
                { label: t('business.cmdb.host.status'), value: t(`business.cmdb.host.status.${host.status}`) },
                { label: t('business.cmdb.host.owner'), value: host.owner || '-' },
                { label: t('business.cmdb.host.remark'), value: host.remark || '-' },
              ]}
            />
          </FormSection>
        </Card>
        <Card className="page-panel">
          <FormSection title={t('business.cmdb.host.systemConfig')}>
            <Descriptions
              column={2}
              data={[
                { label: t('business.cmdb.host.os'), value: t(`business.cmdb.host.os.${host.os}`) },
                { label: t('business.cmdb.host.osVersion'), value: host.osVersion || '-' },
                { label: t('business.cmdb.host.cpuCores'), value: host.cpuCores || '-' },
                { label: t('business.cmdb.host.memoryGb'), value: host.memoryGb ? `${host.memoryGb}` : '-' },
                { label: t('business.cmdb.host.diskGb'), value: host.diskGb ? `${host.diskGb}` : '-' },
                { label: t('business.cmdb.host.sshPort'), value: host.sshPort || 22 },
              ]}
            />
          </FormSection>
        </Card>
        <Card className="page-panel">
          <FormSection title={t('business.cmdb.host.labels')}>
            {host.labelValues?.length ? (
              <Space wrap>
                {host.labelValues.map((l, i) => (
                  <Tag key={i}>
                    {l.key}={l.val}
                  </Tag>
                ))}
              </Space>
            ) : (
              <PageEmpty description={t('business.cmdb.host.labelsEmpty')} />
            )}
          </FormSection>
        </Card>
        <Card className="page-panel">
          <FormSection title={t('business.cmdb.host.installedComponents')}>
            {host.installedComponents?.length ? (
              <Space wrap>
                {host.installedComponents.map((c, i) => (
                  <Tag key={i} color="arcoblue">
                    {c.name} {c.version}
                  </Tag>
                ))}
              </Space>
            ) : (
              <PageEmpty description={t('business.cmdb.host.componentsEmpty')} />
            )}
          </FormSection>
        </Card>
      </Space>
      <AppModal
        visible={collectVisible}
        onCancel={() => setCollectVisible(false)}
        title={t('business.cmdb.collect.modalTitle')}
        footer={null}
      >
        <Form form={collectForm} layout="vertical" onSubmit={handleCollect}>
          <Form.Item
            label={t('business.cmdb.host.collectSshUser')}
            field="sshUser"
            rules={[{ required: true }]}
          >
            <Input placeholder={t('business.cmdb.host.collectSshUserPlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('business.cmdb.host.collectAuthMode')}
            field="authMode"
            rules={[{ required: true }]}
            initialValue="password"
          >
            <Select>
              <Select.Option value="password">
                {t('business.cmdb.collect.authMode.password')}
              </Select.Option>
              <Select.Option value="private_key">
                {t('business.cmdb.collect.authMode.privateKey')}
              </Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            label={t('business.cmdb.collect.hostFingerprint')}
            field="hostFingerprint"
            rules={[{ required: true }]}
          >
            <Input placeholder={t('business.cmdb.collect.hostFingerprintPlaceholder')} />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, next) => prev.authMode !== next.authMode}>
            {(values) =>
              values.authMode === 'private_key' ? (
                <Form.Item
                  label={t('business.cmdb.host.collectPrivateKey')}
                  field="sshPrivateKey"
                  rules={[{ required: true }]}
                >
                  <Input.TextArea
                    rows={4}
                    placeholder={t('business.cmdb.host.collectPrivateKeyPlaceholder')}
                  />
                </Form.Item>
              ) : (
                <Form.Item
                  label={t('business.cmdb.host.collectPassword')}
                  field="sshPassword"
                  rules={[{ required: true }]}
                >
                  <Input.Password />
                </Form.Item>
              )
            }
          </Form.Item>
          <div style={{ color: 'var(--text-tertiary)', marginBottom: 16 }}>
            {t('business.cmdb.collect.hint')}
          </div>
          <SubmitBar
            onCancel={() => setCollectVisible(false)}
            loading={collecting}
            submitText={t('business.cmdb.collect.start')}
          />
        </Form>
      </AppModal>
    </PageContainer>
  );
}
