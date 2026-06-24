import React from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
  type FormInstance,
} from '@arco-design/web-react';
import { IconRefresh } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { FormSection, SubmitBar } from '../../../../components';
import { pantheonThemeOptions } from '../../../../core/theme/theme';
import { SUPPORTED_LOCALES } from '../../../../i18n';
import type { SettingItem } from './api';
import {
  auditRetentionDaySettingKeys,
  auditRetentionSettingKeys,
  integerSettingKeys,
  isSameStringArray,
  normalizeAuditRetentionTagValues,
  parseAuditRetentionSettingValue,
  parseDefaultFieldValue,
  recommendedAuditRetentionDayOptions,
  recommendedAuditRetentionOptions,
  resolveAuditRetentionDefaultValues,
  type SettingFormValue,
} from './useSettingCatalog';

const FormItem = Form.Item;

interface SettingGroupFormProps {
  form: FormInstance<Record<string, SettingFormValue>>;
  activeGroupKey: string;
  activeGroupItems: SettingItem[];
  canUpdateSetting: boolean;
  canRefreshCache: boolean;
  refreshingCache: boolean;
  submittingGroup: string | null;
  onRefreshCache: () => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const SettingGroupForm: React.FC<SettingGroupFormProps> = ({
  form,
  activeGroupKey,
  activeGroupItems,
  canUpdateSetting,
  canRefreshCache,
  refreshingCache,
  submittingGroup,
  onRefreshCache,
  onSubmit,
  onCancel,
}) => {
  const { t } = useTranslation();

  const formatDefaultValueLabel = (item: SettingItem) => {
    if (auditRetentionSettingKeys.has(item.settingKey)) {
      const defaultValues = resolveAuditRetentionDefaultValues(item);
      return defaultValues
        .map((option) => t('common.keepRecentDays', { count: Number(option) }))
        .join(' / ');
    }
    if (item.settingKey === 'upload.storage_driver') {
      return t(
        `system.setting.option.upload.storage_driver.${item.defaultValue}`,
        item.defaultValue,
      );
    }
    if (item.settingKey === 'i18n.default_language') {
      return t(`app.language.${item.defaultValue}`, item.defaultValue);
    }
    if (item.settingKey === 'ui.default_theme') {
      const matchedTheme = pantheonThemeOptions.find((theme) => theme.key === item.defaultValue);
      return matchedTheme ? t(matchedTheme.labelKey) : item.defaultValue;
    }
    if (item.settingKey === 'platform.app_mode') {
      return t(`system.setting.option.platform.app_mode.${item.defaultValue}`, item.defaultValue);
    }
    if (item.valueType === 'boolean') {
      return item.defaultValue === 'true' ? t('common.yes') : t('common.no');
    }
    if (item.defaultValue === '') {
      return t('system.setting.defaultValueEmpty');
    }
    return item.defaultValue;
  };

  const renderField = (item: SettingItem) => {
    const label = t(`system.setting.item.${item.settingKey}`, item.settingKey);
    const isWideField =
      auditRetentionSettingKeys.has(item.settingKey) ||
      item.valueType === 'json' ||
      item.isEncrypted === 1;
    const fieldClassName = isWideField
      ? 'setting-page__field setting-page__field--full'
      : 'setting-page__field';
    const remark = t(item.remark, '');
    const help = (
      <Space direction="vertical" size={4}>
        {remark ? <span>{remark}</span> : null}
        {item.isEncrypted === 1 ? (
          <Space size={8} wrap>
            <Tag color="red">{t('system.setting.encrypted')}</Tag>
            <Typography.Text type="secondary">
              {item.hasValue === 1
                ? t('system.setting.leaveEmptyToKeep')
                : t('system.setting.encryptedEmptyHint')}
            </Typography.Text>
          </Space>
        ) : (
          <Space size={12} wrap>
            <Button
              type="text"
              size="small"
              onClick={() => {
                form.setFieldValue(item.settingKey, parseDefaultFieldValue(item));
              }}
            >
              {t('system.setting.restoreDefault')}
            </Button>
            <Typography.Text type="secondary">
              {t('system.setting.defaultValueHint', { value: formatDefaultValueLabel(item) })}
            </Typography.Text>
          </Space>
        )}
      </Space>
    );

    if (item.settingKey === 'ui.default_theme') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={pantheonThemeOptions.map((theme) => ({
              label: `${t(theme.labelKey)} · ${t(theme.descriptionKey)}`,
              value: theme.key,
            }))}
          />
        </FormItem>
      );
    }
    if (item.settingKey === 'platform.app_mode') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={[
              {
                label: t('system.setting.option.platform.app_mode.enterprise'),
                value: 'enterprise',
              },
              { label: t('system.setting.option.platform.app_mode.consumer'), value: 'consumer' },
              { label: t('system.setting.option.platform.app_mode.hybrid'), value: 'hybrid' },
            ]}
          />
        </FormItem>
      );
    }
    if (item.settingKey === 'upload.storage_driver') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={[
              { label: t('system.setting.option.upload.storage_driver.local'), value: 'local' },
              { label: t('system.setting.option.upload.storage_driver.s3'), value: 's3' },
            ]}
          />
        </FormItem>
      );
    }
    if (item.settingKey === 'i18n.default_language') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={SUPPORTED_LOCALES.map((locale) => ({
              label: t(`app.language.${locale}`),
              value: locale,
            }))}
          />
        </FormItem>
      );
    }
    if (auditRetentionSettingKeys.has(item.settingKey)) {
      const savedValues = parseAuditRetentionSettingValue(item.settingValue);
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
          rules={[
            {
              required: true,
              type: 'array',
              minLength: 1,
              message: t('system.setting.audit.retentionRequired'),
            },
            {
              validator: (_value, callback) => {
                const currentValue = form.getFieldValue(item.settingKey);
                const normalized = normalizeAuditRetentionTagValues(
                  Array.isArray(currentValue) ? currentValue : [],
                );
                if (normalized.length === 0) {
                  callback(t('system.setting.audit.retentionRequired'));
                  return;
                }
                if (normalized.length !== (Array.isArray(currentValue) ? currentValue.length : 0)) {
                  callback(t('system.setting.audit.retentionInvalid'));
                  return;
                }
                callback();
              },
            },
          ]}
        >
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            <Select
              mode="multiple"
              allowCreate
              placeholder={t('system.setting.audit.retentionPlaceholder')}
              options={recommendedAuditRetentionOptions.map((option) => ({
                label: t('common.keepRecentDays', { count: option }),
                value: String(option),
              }))}
              onChange={(value) => {
                form.setFieldValue(
                  item.settingKey,
                  normalizeAuditRetentionTagValues(value as Array<string | number>),
                );
              }}
            />
            <FormItem noStyle shouldUpdate>
              {() => {
                const currentValue = normalizeAuditRetentionTagValues(
                  Array.isArray(form.getFieldValue(item.settingKey))
                    ? (form.getFieldValue(item.settingKey) as Array<string | number>)
                    : [],
                );
                const dirty = !isSameStringArray(currentValue, savedValues);
                if (!dirty) {
                  return null;
                }
                return <Tag color="orange">{t('system.setting.audit.unsavedChanges')}</Tag>;
              }}
            </FormItem>
          </Space>
        </FormItem>
      );
    }

    if (item.valueType === 'boolean') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
          triggerPropName="checked"
        >
          <Switch checkedText={t('common.yes')} uncheckedText={t('common.no')} />
        </FormItem>
      );
    }
    if (item.valueType === 'number') {
      if (auditRetentionDaySettingKeys.has(item.settingKey)) {
        return (
          <FormItem
            key={item.settingKey}
            className={fieldClassName}
            field={item.settingKey}
            label={label}
            extra={help}
            rules={[{ required: true, message: t('common.requiredField', { field: label }) }]}
          >
            <Select
              options={recommendedAuditRetentionDayOptions.map((option) => ({
                label: t('system.setting.audit.retentionDaysOption', { count: option }),
                value: option,
              }))}
            />
          </FormItem>
        );
      }
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
          rules={[{ required: true, message: t('common.requiredField', { field: label }) }]}
        >
          <InputNumber
            style={{ width: '100%' }}
            precision={integerSettingKeys.has(item.settingKey) ? 0 : undefined}
            min={integerSettingKeys.has(item.settingKey) ? 1 : undefined}
          />
        </FormItem>
      );
    }
    if (item.isEncrypted === 1) {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Input.Password
            placeholder={
              item.hasValue === 1
                ? t('system.setting.leaveEmptyToKeep')
                : t('system.setting.encryptedPlaceholder')
            }
            onPressEnter={() => form.submit()}
          />
        </FormItem>
      );
    }
    if (item.valueType === 'json') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Input.TextArea autoSize={{ minRows: 4, maxRows: 10 }} />
        </FormItem>
      );
    }
    return (
      <FormItem
        key={item.settingKey}
        className={fieldClassName}
        field={item.settingKey}
        label={label}
        extra={help}
      >
        <Input onPressEnter={() => form.submit()} />
      </FormItem>
    );
  };

  return (
    <Card className="page-panel setting-page__config-card">
      <Form
        form={form}
        layout="vertical"
        onSubmit={() => {
          onSubmit();
        }}
      >
        <Space
          direction="vertical"
          size={10}
          className="dialog-form-stack setting-page__form-stack"
          style={{ marginTop: 10 }}
        >
          <FormSection
            title={t(`system.setting.group.${activeGroupKey}`)}
            description={t(`system.setting.groupHint.${activeGroupKey}`, '')}
          >
            <div className="setting-page__field-grid">{activeGroupItems.map(renderField)}</div>
          </FormSection>
          <Typography.Text type="secondary" className="setting-page__save-hint">
            {t('system.setting.saveHint')}
          </Typography.Text>
          <div className="setting-page__actions">
            <Space className="setting-page__meta-actions">
              <Button
                size="small"
                icon={<IconRefresh />}
                loading={refreshingCache}
                onClick={() => {
                  onRefreshCache();
                }}
                disabled={!canRefreshCache}
              >
                {t('system.setting.cache.refresh')}
              </Button>
            </Space>
            <SubmitBar
              loading={submittingGroup === activeGroupKey}
              submitDisabled={!canUpdateSetting}
              onCancel={onCancel}
              onSubmit={() => {
                onSubmit();
              }}
            />
          </div>
        </Space>
      </Form>
    </Card>
  );
};

export default SettingGroupForm;
