import React from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Grid,
  Input,
  Space,
  Table,
  Tag,
  Typography,
} from '@arco-design/web-react';

const { Row, Col } = Grid;
import { useTranslation } from 'react-i18next';

import {
  PAGE_ACTION_TEMPLATE_DEFINITIONS,
  type GeneratorCompletenessIssue,
  type GeneratorMenuPreviewNode,
  type ModuleSchema,
  type PageActionKey,
  type PageActionTemplate,
} from '../schema';
import type { GeneratedFile } from '../exporter';

export interface TranslationPreviewRow {
  key: string;
  zh: string;
  en: string;
}

interface Step3PreviewProps {
  previewSchema: ModuleSchema;
  previewMenuTree: GeneratorMenuPreviewNode[];
  previewCompletenessIssues: GeneratorCompletenessIssue[];
  previewGeneratedFiles: GeneratedFile[];
  previewTranslationRows: TranslationPreviewRow[];
  selectedActionTemplate: PageActionTemplate;
  pageActions: PageActionKey[];
  onPageActionsChange: (actions: PageActionKey[]) => void;
  updateTranslationOverride: (key: string, locale: 'zh' | 'en', value: string) => void;
  onExportTranslations: () => void;
  onImportTranslations: () => void;
  onGenerate: () => void;
  previewBlockingIssue: boolean;
  onPrevious: () => void;
}

function renderMenuPreview(
  nodes: GeneratorMenuPreviewNode[],
  t: (key: string) => string,
): React.ReactNode {
  return (
    <div className="generator-wizard__menu-tree">
      {nodes.length === 0 ? (
        <Typography.Text type="secondary">
          {t('generator.wizard.step3.menuPreview.empty')}
        </Typography.Text>
      ) : (
        nodes.map((node) => (
          <div key={node.key} className="generator-wizard__menu-node">
            <Space wrap>
              <Tag color={node.type === 'M' ? 'arcoblue' : node.type === 'C' ? 'green' : 'orange'}>
                {node.type}
              </Tag>
              <Typography.Text>{node.titleKey}</Typography.Text>
              {node.path ? <Typography.Text type="secondary">{node.path}</Typography.Text> : null}
            </Space>
            {node.children.length > 0 ? renderMenuPreview(node.children, t) : null}
          </div>
        ))
      )}
    </div>
  );
}

const Step3Preview: React.FC<Step3PreviewProps> = ({
  previewSchema,
  previewMenuTree,
  previewCompletenessIssues,
  previewGeneratedFiles,
  previewTranslationRows,
  selectedActionTemplate,
  pageActions,
  onPageActionsChange,
  updateTranslationOverride,
  onExportTranslations,
  onImportTranslations,
  onGenerate,
  previewBlockingIssue,
  onPrevious,
}) => {
  const { t } = useTranslation();

  return (
    <div>
      <Typography.Title heading={5}>{t('generator.wizard.step3.title')}</Typography.Title>
      <Typography.Text type="secondary" className="generator-wizard__description">
        {t('generator.wizard.step3.desc')}
      </Typography.Text>

      <Row gutter={16}>
        <Col xs={24} lg={12}>
          <Card title={t('generator.wizard.step3.actions')} className="generator-wizard__section">
            <Typography.Text type="secondary" className="generator-wizard__subdescription">
              {t(
                PAGE_ACTION_TEMPLATE_DEFINITIONS.find((item) => item.key === selectedActionTemplate)
                  ?.descriptionKey || 'generator.actionTemplates.standard.desc',
              )}
            </Typography.Text>
            <Checkbox.Group
              value={pageActions}
              disabled={previewSchema.metadata?.tableRole === 'relation'}
              options={['view', 'detail', 'create', 'update', 'delete', 'export', 'import'].map(
                (item) => ({
                  label: t(`generator.pageActions.${item}`),
                  value: item,
                }),
              )}
              onChange={(value) => {
                onPageActionsChange(value as PageActionKey[]);
              }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={t('generator.wizard.step3.dataPolicies')}
            className="generator-wizard__section"
          >
            <Space wrap>
              <Tag color="green">
                {t('generator.wizard.step3.fieldCount', {
                  count: previewSchema.model.fields.length,
                })}
              </Tag>
              <Tag color="arcoblue">
                {t('generator.wizard.step3.uniqueCount', {
                  count: previewSchema.model.fields.filter((field) => field.validation?.unique)
                    .length,
                })}
              </Tag>
              <Tag color="purple">
                {t('generator.wizard.step3.enumCount', {
                  count: previewSchema.model.fields.filter((field) => field.type === 'enum').length,
                })}
              </Tag>
            </Space>
            <div className="generator-wizard__enum-list">
              {previewSchema.model.fields
                .filter((field) => field.type === 'enum')
                .map((field) => (
                  <div key={field.name} className="generator-wizard__enum-item">
                    <Typography.Text>{field.label}</Typography.Text>
                    <Typography.Text type="secondary">
                      {' '}
                      · {field.dictCode || t('generator.fieldEditor.enumInline')}
                    </Typography.Text>
                  </div>
                ))}
            </div>
          </Card>
        </Col>
      </Row>

      <Card title={t('generator.wizard.step3.permissions')} className="generator-wizard__section">
        <Space wrap>
          {previewSchema.permissions.length === 0 ? (
            <Typography.Text type="secondary">
              {t('generator.wizard.step3.permissions.empty')}
            </Typography.Text>
          ) : (
            previewSchema.permissions.map((permission) => (
              <Tag key={permission.key}>{permission.key}</Tag>
            ))
          )}
        </Space>
      </Card>

      <Row gutter={16}>
        <Col xs={24} lg={12}>
          <Card
            title={t('generator.wizard.step3.menuPreview')}
            className="generator-wizard__section"
          >
            <Typography.Text type="secondary" className="generator-wizard__subdescription">
              {t('generator.wizard.step3.menuPreview.desc')}
            </Typography.Text>
            {renderMenuPreview(previewMenuTree, t)}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={t('generator.wizard.step3.i18nCompleteness')}
            className="generator-wizard__section"
          >
            <Typography.Text type="secondary" className="generator-wizard__subdescription">
              {t('generator.wizard.step3.i18nCompleteness.desc')}
            </Typography.Text>
            <Space direction="vertical" className="generator-wizard__full">
              {previewCompletenessIssues.length === 0 ? (
                <Alert type="success" content={t('generator.validation.passed')} />
              ) : (
                previewCompletenessIssues.map((issue) => (
                  <Alert
                    key={`${issue.code}-${issue.detail || ''}`}
                    type={issue.level === 'error' ? 'error' : 'warning'}
                    content={`${t(issue.messageKey)}${issue.detail ? `: ${issue.detail}` : ''}`}
                  />
                ))
              )}
            </Space>
          </Card>
        </Col>
      </Row>

      <Card
        title={t('generator.wizard.step3.translationPreview')}
        className="generator-wizard__section"
      >
        <Space direction="vertical" className="generator-wizard__full" size={12}>
          <Space align="center" className="generator-wizard__toolbar">
            <Typography.Text type="secondary">
              {t('generator.wizard.step3.translationPreview.desc')}
            </Typography.Text>
            <Space wrap>
              <Button size="small" onClick={onExportTranslations}>
                {t('generator.wizard.step3.translationPreview.export')}
              </Button>
              <Button size="small" onClick={onImportTranslations}>
                {t('generator.wizard.step3.translationPreview.import')}
              </Button>
            </Space>
          </Space>
          <Table
            rowKey="key"
            data={previewTranslationRows}
            pagination={{ pageSize: 8, sizeCanChange: true }}
            columns={[
              {
                title: t('generator.wizard.step3.translationPreview.key'),
                dataIndex: 'key',
                width: 320,
                render: (value: string) => <Typography.Text code>{value}</Typography.Text>,
              },
              {
                title: t('generator.wizard.step3.translationPreview.zh'),
                dataIndex: 'zh',
                render: (value: string, record: TranslationPreviewRow) => (
                  <Input
                    value={value}
                    className="generator-wizard__translation-input"
                    onChange={(nextValue) => updateTranslationOverride(record.key, 'zh', nextValue)}
                  />
                ),
              },
              {
                title: t('generator.wizard.step3.translationPreview.en'),
                dataIndex: 'en',
                render: (value: string, record: TranslationPreviewRow) => (
                  <Input
                    value={value}
                    className="generator-wizard__translation-input"
                    onChange={(nextValue) => updateTranslationOverride(record.key, 'en', nextValue)}
                  />
                ),
              },
            ]}
          />
        </Space>
      </Card>

      <Card
        title={t('generator.wizard.step3.generationImpact')}
        className="generator-wizard__section"
      >
        <Space wrap className="generator-wizard__section">
          <Tag color="arcoblue">
            {t('generator.wizard.step3.impact.files', {
              count: previewGeneratedFiles.length,
            })}
          </Tag>
          <Tag color="green">{previewSchema.model.tableName}</Tag>
          <Tag color={previewSchema.metadata?.tableRole === 'relation' ? 'orange' : 'purple'}>
            {t(`generator.wizard.tableRole.${previewSchema.metadata?.tableRole || 'main'}`)}
          </Tag>
          <Tag color={previewSchema.enableDataScope ? 'green' : 'gray'}>
            {t(
              previewSchema.enableDataScope
                ? 'generator.wizard.dataScope.enabledTag'
                : 'generator.wizard.dataScope.disabledTag',
            )}
          </Tag>
          <Tag color={previewSchema.includeDashboardWidget ? 'arcoblue' : 'gray'}>
            {t(
              previewSchema.includeDashboardWidget
                ? 'generator.wizard.dashboardWidget.enabledTag'
                : 'generator.wizard.dashboardWidget.disabledTag',
            )}
          </Tag>
          <Tag color="blue">
            {t('generator.wizard.step3.impact.dependencies', {
              count: previewSchema.dependencies?.length || 0,
            })}
          </Tag>
          <Tag color="orange">
            {t('generator.wizard.step3.impact.relations', {
              count: previewSchema.relations?.length || 0,
            })}
          </Tag>
        </Space>
        <div className="generator-wizard__impact-list">
          {previewGeneratedFiles.slice(0, 8).map((file) => (
            <Typography.Text key={file.path} code className="generator-wizard__impact-item">
              {file.path}
            </Typography.Text>
          ))}
          {previewGeneratedFiles.length > 8 ? (
            <Typography.Text type="secondary">
              {t('generator.wizard.step3.impact.more', {
                count: previewGeneratedFiles.length - 8,
              })}
            </Typography.Text>
          ) : null}
        </div>
      </Card>

      <Space>
        <Button onClick={onPrevious}>{t('common.previous')}</Button>
        <Button type="primary" onClick={onGenerate} disabled={previewBlockingIssue}>
          {t('generator.wizard.generate')}
        </Button>
      </Space>
    </div>
  );
};

export default Step3Preview;
