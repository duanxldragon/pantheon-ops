import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Divider,
  Form,
  Grid,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Steps,
  Table,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import {
  IconCode,
  IconDelete,
  IconDownload,
  IconEdit,
  IconPlus,
  IconRefresh,
} from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { isRequestError } from '../../../api/request';
import { ensureOperationVerified } from '../../../api/request';
import PermissionAction from '../../../components/patterns/PermissionAction';
import { AppModal, PageContainer, PageHeader, showAppModalConfirm } from '../../../components';
import { usePermission } from '../../../hooks/usePermission';

import type { GenerateAndRegisterResp } from '../api';
import {
  createGeneratorDatasource,
  deleteGeneratorDatasource,
  generateAndRegisterModule,
  listGeneratorDatasources,
  listGeneratorTables,
  previewGeneratorTable,
  testGeneratorDatasource,
  updateGeneratorDatasource,
  type GeneratorDatasource,
  type GeneratorTableOption,
  type UpsertGeneratorDatasourcePayload,
} from '../api';
import { FieldEditor } from '../components/FieldEditor';
import { CodePreview } from '../components/CodePreview';
import './ModuleWizard.css';
import { ModuleExporter } from '../exporter';
import {
  buildDashboardQuickActionDescriptionKey,
  buildEnumOptionKey,
  buildFieldHelpTextKey,
  buildFieldLabelKey,
  buildFieldPlaceholderKey,
  buildMenuGroupTitleKey,
  buildModuleNamespace,
  buildPermissionTitleKey,
  buildAuditActionKey,
  buildTitleKey,
  buildMenuPreview,
  generateDefaultMenus,
  generateDefaultPermissions,
  getPageActions,
  inferBusinessContextFromName,
  inferModelName,
  inferMenuGroupDisplayName,
  normalizeMenuPath,
  normalizeBusinessContext,
  normalizeModulePath,
  isValidScopedModulePath,
  normalizeFields,
  PAGE_ACTION_TEMPLATE_DEFINITIONS,
  validateGeneratorCompleteness,
  type BusinessTableRole,
  type DataScopeMode,
  type GeneratorMenuPreviewNode,
  type ModuleField,
  type ModuleRelationType,
  type ModuleSchema,
  type ModuleScope,
  type PageActionKey,
  type PageActionTemplate,
  type TemplateLevel,
} from '../schema';
import { SECONDARY_VERIFY_CANCELLED_ERROR } from '../../../components/feedback/secondaryVerifyController';

const FormItem = Form.Item;
const { Row, Col } = Grid;

interface TranslationOverride {
  zh?: string;
  en?: string;
}

interface TranslationPreviewRow {
  key: string;
  zh: string;
  en: string;
}

function escapeCsvCell(value: string): string {
  const normalized = String(value ?? '');
  if (!/[",\n\r]/.test(normalized)) {
    return normalized;
  }
  return `"${normalized.replace(/"/g, '""')}"`;
}

function parseCsvRows(content: string): string[][] {
  const rows: string[][] = [];
  let current = '';
  let row: string[] = [];
  let inQuotes = false;

  for (let index = 0; index < content.length; index += 1) {
    const char = content[index];
    const next = content[index + 1];
    if (char === '"' && inQuotes && next === '"') {
      current += '"';
      index += 1;
      continue;
    }
    if (char === '"') {
      inQuotes = !inQuotes;
      continue;
    }
    if (char === ',' && !inQuotes) {
      row.push(current);
      current = '';
      continue;
    }
    if ((char === '\n' || char === '\r') && !inQuotes) {
      if (char === '\r' && next === '\n') {
        index += 1;
      }
      row.push(current);
      if (row.some((cell) => cell.trim() !== '')) {
        rows.push(row);
      }
      row = [];
      current = '';
      continue;
    }
    current += char;
  }

  row.push(current);
  if (row.some((cell) => cell.trim() !== '')) {
    rows.push(row);
  }
  return rows;
}

const ModuleWizard: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { hasPerm, isAdmin } = usePermission();
  const [currentStep, setCurrentStep] = useState(0);
  const [form] = Form.useForm<Partial<ModuleSchema>>();
  const [fields, setFields] = useState<ModuleField[]>([]);
  const [generatedFiles, setGeneratedFiles] = useState<ReturnType<ModuleExporter['generateAll']>>(
    [],
  );
  const [showPreview, setShowPreview] = useState(false);
  const [registering, setRegistering] = useState(false);
  const [registerResult, setRegisterResult] = useState<GenerateAndRegisterResp | null>(null);
  const [dynamicModuleDisabled, setDynamicModuleDisabled] = useState(false);
  const [datasources, setDatasources] = useState<GeneratorDatasource[]>([]);
  const [datasourceModalVisible, setDatasourceModalVisible] = useState(false);
  const [datasourceSaving, setDatasourceSaving] = useState(false);
  const [selectedDatasourceId, setSelectedDatasourceId] = useState('current');
  const [editingDatasourceId, setEditingDatasourceId] = useState<string | null>(null);
  const [tableOptions, setTableOptions] = useState<GeneratorTableOption[]>([]);
  const [tableLoading, setTableLoading] = useState(false);
  const [sourceMode, setSourceMode] = useState<'manual' | 'database'>('manual');
  const [selectedTableRole, setSelectedTableRole] = useState<BusinessTableRole>('main');
  const [templateVersion, setTemplateVersion] = useState<'v1'>('v1');
  const [dependencyModulesText, setDependencyModulesText] = useState('');
  const [relationContractsText, setRelationContractsText] = useState('');
  const [enableDataScope, setEnableDataScope] = useState(true);
  const [includeDashboardWidget, setIncludeDashboardWidget] = useState(true);
  const [dataScopeMode, setDataScopeMode] = useState<DataScopeMode>('dept');
  const [translationOverrides, setTranslationOverrides] = useState<
    Record<string, TranslationOverride>
  >({});
  const [datasourceForm] = Form.useForm<UpsertGeneratorDatasourcePayload>();
  const canManageDatasources = isAdmin || hasPerm('system:generator:datasource:manage');
  const actionLabel = (action: Exclude<PageActionKey, 'detail'>, locale: 'zh-CN' | 'en-US') =>
    t(`generator.pageActions.${action}`, { lng: locale });

  const selectedDatasource = datasources.find((item) => item.id === selectedDatasourceId);

  const loadDatasources = useCallback(async () => {
    const items = await listGeneratorDatasources();
    setDatasources(items);
    if (!items.some((item) => item.id === selectedDatasourceId)) {
      setSelectedDatasourceId(items[0]?.id || 'current');
    }
    return items;
  }, [selectedDatasourceId]);

  const loadTables = useCallback(async (datasourceId: string) => {
    setTableLoading(true);
    try {
      const items = await listGeneratorTables(datasourceId);
      setTableOptions(items);
    } catch {
      setTableOptions([]);
    } finally {
      setTableLoading(false);
    }
  }, []);

  const applyPreviewSuggestions = (preview: Awaited<ReturnType<typeof previewGeneratorTable>>) => {
    const currentName = normalizeModulePath(String(form.getFieldValue('name') || ''));
    const currentDisplayName = String(form.getFieldValue('displayName') || '').trim();
    const currentDisplayNameEn = String(form.getFieldValue('displayNameEn') || '').trim();
    const currentScope = String(form.getFieldValue('scope') || '').trim();
    const currentScopedValue =
      currentScope === 'system' || currentScope === 'business'
        ? currentScope
        : preview.suggestedScope;

    if (!currentName || !isValidScopedModulePath(currentScopedValue, currentName)) {
      form.setFieldValue('name', normalizeModulePath(preview.suggestedName));
    }
    if (!currentDisplayName) {
      form.setFieldValue('displayName', preview.suggestedTitle);
    }
    if (!currentDisplayNameEn) {
      form.setFieldValue('displayNameEn', preview.suggestedTitle);
    }
    if (!currentScope) {
      form.setFieldValue('scope', preview.suggestedScope);
    }
  };

  useEffect(() => {
    let active = true;
    const timer = globalThis.setTimeout(() => {
      void loadDatasources()
        .then((items) => {
          if (!active) {
            return;
          }
          const firstID =
            items.find((item) => item.id === selectedDatasourceId)?.id || items[0]?.id || 'current';
          setSelectedDatasourceId(firstID);
          return loadTables(firstID);
        })
        .catch(() => {
          if (active) {
            setDatasources([]);
            setTableOptions([]);
          }
        });
    }, 0);
    return () => {
      active = false;
      globalThis.clearTimeout(timer);
    };
  }, [loadDatasources, loadTables, selectedDatasourceId]);

  useEffect(() => {
    if (sourceMode !== 'database') {
      return;
    }
    const timer = globalThis.setTimeout(() => {
      form.setFieldValue('metadata.sourceDatasourceId' as keyof ModuleSchema, selectedDatasourceId);
      form.setFieldValue(
        'metadata.sourceDatasourceName' as keyof ModuleSchema,
        selectedDatasource?.name || '',
      );
      form.setFieldValue('metadata.sourceTable' as keyof ModuleSchema, undefined);
      void loadTables(selectedDatasourceId);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [form, loadTables, selectedDatasource?.name, selectedDatasourceId, sourceMode]);

  const getAllFormValues = () => form.getFields() as Partial<ModuleSchema>;

  const parseDependencyModules = () =>
    dependencyModulesText
      .split(/[\n,]/)
      .map((item) => item.trim())
      .filter(Boolean)
      .map((module) => ({ module, required: true }));

  const parseRelationContracts = () =>
    relationContractsText
      .split('\n')
      .map((item) => item.trim())
      .filter(Boolean)
      .map((item) => {
        const [
          name = '',
          type = 'lookup',
          targetModule = '',
          localField = '',
          targetField = '',
          junctionTable = '',
        ] = item.split('|').map((part) => part.trim());
        return {
          name,
          type: type as ModuleRelationType,
          targetModule,
          localField,
          targetField,
          junctionTable: junctionTable || undefined,
        };
      });

  const readMetadataValues = () => {
    const metadata = getAllFormValues().metadata;
    return {
      businessContext: metadata?.businessContext || undefined,
      businessContextTitle: metadata?.businessContextTitle || undefined,
      businessContextTitleEn: metadata?.businessContextTitleEn || undefined,
      tableRole: metadata?.tableRole || 'main',
      primaryTable: metadata?.primaryTable || undefined,
      relationFromField: metadata?.relationFromField || undefined,
      relationToField: metadata?.relationToField || undefined,
      boundedContext: metadata?.boundedContext || undefined,
      owner: metadata?.owner || undefined,
      summary: metadata?.summary || undefined,
      sourceMode: metadata?.sourceMode || sourceMode,
      sourceDatasourceId: metadata?.sourceDatasourceId || undefined,
      sourceDatasourceName: metadata?.sourceDatasourceName || undefined,
      sourceTable: metadata?.sourceTable || undefined,
    };
  };

  const buildSchema = (): ModuleSchema => {
    const values = getAllFormValues();
    const metadata = readMetadataValues();
    const name = normalizeModulePath(values.name || '');
    const displayName = values.displayName || '';
    const displayNameEn = values.displayNameEn || displayName;
    const scope = (values.scope as ModuleScope | undefined) || 'business';
    const parentMenu = normalizeMenuPath(values.parentMenu);
    const templateLevel = (values.templateLevel as TemplateLevel | undefined) || 'enterprise';
    const tableRole = (metadata.tableRole as BusinessTableRole | undefined) || 'main';
    const canAttachDashboardWidget = scope === 'business' && tableRole !== 'relation';
    const pageActionTemplate =
      (values.pageActionTemplate as PageActionTemplate | undefined) || 'standard';
    const pageActions =
      tableRole === 'relation'
        ? []
        : ((values.pageActions as PageActionKey[] | undefined) ??
          getPageActions({
            pageActionTemplate,
            enableExport: templateLevel === 'enterprise',
            enableImport: templateLevel === 'enterprise',
          }));
    const enableExport = pageActions.includes('export');
    const enableImport = pageActions.includes('import');
    const model = values.model || {
      tableName: scope === 'system' ? `system_${name}` : `biz_${name}`,
      fields: [],
    };
    const normalizedFields = normalizeFields(fields);
    const titleKey = buildTitleKey(scope, name);
    const dashboardQuickActionDescriptionKey = buildDashboardQuickActionDescriptionKey(scope, name);
    const moduleSegments = name.split('/').filter(Boolean);
    const businessContext = normalizeBusinessContext(
      metadata.businessContext || inferBusinessContextFromName(name),
    );
    const businessContextTitle =
      metadata.businessContextTitle || inferMenuGroupDisplayName(businessContext);
    const businessContextTitleEn = metadata.businessContextTitleEn || businessContextTitle;
    const zhTranslations = normalizedFields.reduce<Record<string, string>>(
      (acc, field) => {
        acc[buildFieldLabelKey(scope, name, field.name)] = field.label;
        if (field.placeholder) {
          acc[buildFieldPlaceholderKey(scope, name, field.name)] = field.placeholder;
        }
        if (field.helpText) {
          acc[buildFieldHelpTextKey(scope, name, field.name)] = field.helpText;
        }
        for (const item of field.enumOptions ?? []) {
          acc[buildEnumOptionKey(scope, name, field.name, item.value)] = item.label;
        }
        return acc;
      },
      {
        [titleKey]: displayName,
      },
    );
    const enTranslations = normalizedFields.reduce<Record<string, string>>(
      (acc, field) => {
        acc[buildFieldLabelKey(scope, name, field.name)] = field.labelEn || field.label;
        if (field.placeholder || field.placeholderEn) {
          acc[buildFieldPlaceholderKey(scope, name, field.name)] =
            field.placeholderEn || field.placeholder || '';
        }
        if (field.helpText || field.helpTextEn) {
          acc[buildFieldHelpTextKey(scope, name, field.name)] =
            field.helpTextEn || field.helpText || '';
        }
        for (const item of field.enumOptions ?? []) {
          acc[buildEnumOptionKey(scope, name, field.name, item.value)] = item.labelEn || item.label;
        }
        return acc;
      },
      {
        [titleKey]: displayNameEn,
      },
    );
    if (canAttachDashboardWidget && includeDashboardWidget) {
      zhTranslations[dashboardQuickActionDescriptionKey] = `进入${displayName}`;
      enTranslations[dashboardQuickActionDescriptionKey] = `Open ${displayNameEn}`;
    }

    moduleSegments.slice(0, -1).forEach((_, index) => {
      const groupSegments = moduleSegments.slice(0, index + 1);
      const groupTitleKey = buildMenuGroupTitleKey(scope, groupSegments);
      const groupDisplayName =
        index === 0 && groupSegments[0] === businessContext
          ? businessContextTitle
          : inferMenuGroupDisplayName(groupSegments[groupSegments.length - 1]);
      const groupDisplayNameEn =
        index === 0 && groupSegments[0] === businessContext
          ? businessContextTitleEn
          : groupDisplayName;
      zhTranslations[groupTitleKey] = zhTranslations[groupTitleKey] || groupDisplayName;
      enTranslations[groupTitleKey] = enTranslations[groupTitleKey] || groupDisplayNameEn;
    });

    pageActions
      .filter((action) => action !== 'detail')
      .forEach((action) => {
        const key = buildPermissionTitleKey(scope, name, action);
        zhTranslations[key] = `${actionLabel(action, 'zh-CN')}${displayName}`;
        enTranslations[key] = `${actionLabel(action, 'en-US')} ${displayNameEn}`;
      });

    zhTranslations[buildAuditActionKey(scope, name, 'create')] =
      `${actionLabel('create', 'zh-CN')}${displayName}`;
    zhTranslations[buildAuditActionKey(scope, name, 'update')] =
      `${actionLabel('update', 'zh-CN')}${displayName}`;
    zhTranslations[buildAuditActionKey(scope, name, 'delete')] =
      `${actionLabel('delete', 'zh-CN')}${displayName}`;
    enTranslations[buildAuditActionKey(scope, name, 'create')] =
      `${actionLabel('create', 'en-US')} ${displayNameEn}`;
    enTranslations[buildAuditActionKey(scope, name, 'update')] =
      `${actionLabel('update', 'en-US')} ${displayNameEn}`;
    enTranslations[buildAuditActionKey(scope, name, 'delete')] =
      `${actionLabel('delete', 'en-US')} ${displayNameEn}`;

    Object.entries(translationOverrides).forEach(([key, override]) => {
      if (Object.prototype.hasOwnProperty.call(zhTranslations, key) && override.zh !== undefined) {
        zhTranslations[key] = override.zh;
      }
      if (Object.prototype.hasOwnProperty.call(enTranslations, key) && override.en !== undefined) {
        enTranslations[key] = override.en;
      }
    });

    const schema: ModuleSchema = {
      name,
      templateVersion,
      displayName,
      description: values.description,
      displayNameEn,
      scope,
      templateLevel,
      parentMenu,
      pageActionTemplate,
      pageActions,
      dependencies: parseDependencyModules(),
      relations: parseRelationContracts(),
      dataScopeMode: enableDataScope ? dataScopeMode : 'none',
      metadata: {
        businessContext,
        businessContextTitle,
        businessContextTitleEn,
        tableRole,
        primaryTable: metadata.primaryTable,
        relationFromField: metadata.relationFromField,
        relationToField: metadata.relationToField,
        boundedContext: metadata.boundedContext,
        owner: metadata.owner,
        summary: metadata.summary,
        sourceMode: metadata.sourceMode,
        sourceDatasourceId: metadata.sourceDatasourceId,
        sourceDatasourceName: metadata.sourceDatasourceName,
        sourceTable: metadata.sourceTable,
      },
      model: {
        tableName: model.tableName,
        modelName: inferModelName({
          name,
          displayName,
          scope,
          templateLevel,
          pageActionTemplate,
          pageActions,
          model,
          menus: [],
          permissions: [],
          i18n: { namespace: '', translations: { zh: {}, en: {} } },
        } as ModuleSchema),
        fields: normalizedFields,
      },
      menus: [],
      permissions: [],
      i18n: {
        namespace: buildModuleNamespace(scope, name),
        translations: {
          zh: zhTranslations,
          en: enTranslations,
        },
      },
      enableExport,
      enableImport,
      enableAudit: templateLevel === 'enterprise',
      enableDataScope,
      includeDashboardWidget: canAttachDashboardWidget ? includeDashboardWidget : false,
    };

    schema.menus = generateDefaultMenus(schema);
    schema.permissions = generateDefaultPermissions(schema);
    return schema;
  };

  const getNormalizedNameAndScope = () => {
    const values = getAllFormValues();
    const normalizedName = normalizeModulePath(values.name || '');
    const scope = (values.scope as ModuleScope | undefined) || 'business';
    return { normalizedName, scope };
  };

  const syncModuleNameValidation = () => {
    const { normalizedName, scope } = getNormalizedNameAndScope();
    form.setFieldValue('name', normalizedName);
    if (!normalizedName || isValidScopedModulePath(scope, normalizedName)) {
      void form.validate(['name']).catch(() => undefined);
      return true;
    }
    void form.validate(['name']).catch(() => undefined);
    return false;
  };

  const previewSchema = currentStep >= 2 ? buildSchema() : null;
  const selectedActionTemplate =
    (getAllFormValues().pageActionTemplate as PageActionTemplate | undefined) || 'standard';

  const handleBasicInfoSubmit = async () => {
    try {
      let values = getAllFormValues();
      let metadata = readMetadataValues();
      const sourceMode = metadata.sourceMode;
      if (sourceMode === 'database' && !metadata.sourceTable) {
        message.error(t('generator.wizard.sourceTable.required'));
        return;
      }
      let tablePreview: Awaited<ReturnType<typeof previewGeneratorTable>> | null = null;
      if (sourceMode === 'database' && metadata.sourceTable) {
        tablePreview = await previewGeneratorTable(
          metadata.sourceTable,
          metadata.sourceDatasourceId,
        );
        applyPreviewSuggestions(tablePreview);
      }

      await form.validate();
      values = getAllFormValues();
      metadata = readMetadataValues();
      const normalizedName = normalizeModulePath(values.name || '');
      const scope = (values.scope as ModuleScope | undefined) || 'business';
      if (!isValidScopedModulePath(scope, normalizedName)) {
        form.setFieldValue('name', normalizedName);
        void form.validate(['name']).catch(() => undefined);
        return;
      }
      form.setFieldValue('name', normalizedName);
      form.setFieldValue('parentMenu', normalizeMenuPath(values.parentMenu));
      const templateLevel = (values.templateLevel as TemplateLevel | undefined) || 'enterprise';
      const pageActionTemplate =
        (values.pageActionTemplate as PageActionTemplate | undefined) || 'standard';
      let importedFields = fields;
      let tableName = values.scope === 'system' ? `system_${values.name}` : `biz_${values.name}`;
      if (sourceMode === 'database' && tablePreview) {
        importedFields = tablePreview.fields;
        tableName = tablePreview.tableName;
        setFields(tablePreview.fields);
      }
      const schema = {
        ...values,
        pageActionTemplate,
        pageActions: getPageActions({
          pageActionTemplate,
          enableExport: templateLevel === 'enterprise',
          enableImport: templateLevel === 'enterprise',
        }),
        model: {
          tableName,
          fields: importedFields,
        },
      };
      form.setFieldsValue(schema);
      setRegisterResult(null);
      setCurrentStep(1);
    } catch {
      message.error(t('generator.wizard.fillRequired'));
    }
  };

  const handleGenerate = () => {
    const schema = buildSchema();
    const issues = validateGeneratorCompleteness(schema);
    if (issues.some((issue) => issue.level === 'error')) {
      message.error(t('generator.validation.blocked'));
      return;
    }
    const exporter = new ModuleExporter(schema);
    setGeneratedFiles(exporter.generateAll());
    setRegisterResult(null);
    setCurrentStep(3);
  };

  const updateTranslationOverride = (key: string, locale: 'zh' | 'en', value: string) => {
    setTranslationOverrides((current) => ({
      ...current,
      [key]: {
        ...current[key],
        [locale]: value,
      },
    }));
  };

  const handleExportTranslations = () => {
    const schema = buildSchema();
    const keys = Array.from(
      new Set([
        ...Object.keys(schema.i18n.translations.zh),
        ...Object.keys(schema.i18n.translations.en),
      ]),
    ).sort();
    const csv = [
      ['key', 'zh-CN', 'en-US'].map(escapeCsvCell).join(','),
      ...keys.map((key) =>
        [key, schema.i18n.translations.zh[key] || '', schema.i18n.translations.en[key] || '']
          .map(escapeCsvCell)
          .join(','),
      ),
    ].join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${schema.name || 'module'}-i18n.csv`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleImportTranslations = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.csv,text/csv';
    input.onchange = () => {
      const file = input.files?.[0];
      if (!file) {
        return;
      }
      const reader = new FileReader();
      reader.onload = () => {
        try {
          const rows = parseCsvRows(String(reader.result || ''));
          const nextOverrides: Record<string, TranslationOverride> = {};
          rows.slice(1).forEach((row) => {
            const key = String(row[0] || '').trim();
            if (!key) {
              return;
            }
            nextOverrides[key] = {
              zh: row[1] ?? '',
              en: row[2] ?? '',
            };
          });
          setTranslationOverrides((current) => ({ ...current, ...nextOverrides }));
          message.success(t('generator.wizard.step3.translationPreview.importSuccess'));
        } catch {
          message.error(t('generator.wizard.step3.translationPreview.importError'));
        }
      };
      reader.readAsText(file, 'utf-8');
    };
    input.click();
  };

  const resetDatasourceForm = () => {
    setEditingDatasourceId(null);
    datasourceForm.setFieldsValue({
      name: '',
      driver: 'mysql',
      host: '',
      port: 3306,
      databaseName: '',
      username: '',
      password: '',
      status: 1,
      remark: '',
    });
  };

  const handleEditDatasource = (item: GeneratorDatasource) => {
    setEditingDatasourceId(item.id);
    datasourceForm.setFieldsValue({
      name: item.name,
      driver: item.driver || 'mysql',
      host: item.host || '',
      port: item.port || 3306,
      databaseName: item.databaseName,
      username: item.username || '',
      password: '',
      status: item.status,
      remark: item.remark || '',
    });
  };

  const handleSaveDatasource = async () => {
    try {
      const values = await datasourceForm.validate();
      await ensureOperationVerified();
      setDatasourceSaving(true);
      if (editingDatasourceId) {
        await updateGeneratorDatasource(editingDatasourceId, values);
        message.success(t('generator.datasource.saveSuccess'));
      } else {
        await createGeneratorDatasource(values);
        message.success(t('generator.datasource.createSuccess'));
      }
      const items = await loadDatasources();
      if (!editingDatasourceId) {
        const created = items[items.length - 1];
        if (created) {
          setSelectedDatasourceId(created.id);
        }
      }
      resetDatasourceForm();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      if (!isRequestError(error)) {
        return;
      }
      message.error(t('generator.datasource.saveError'));
    } finally {
      setDatasourceSaving(false);
    }
  };

  const handleTestDatasource = async (id: string) => {
    try {
      await ensureOperationVerified();
      await testGeneratorDatasource(id);
      message.success(t('generator.datasource.testSuccess'));
      await loadDatasources();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.datasource.testError'));
    }
  };

  const handleDeleteDatasource = async (id: string) => {
    try {
      await ensureOperationVerified();
      await deleteGeneratorDatasource(id);
      message.success(t('generator.datasource.deleteSuccess'));
      const items = await loadDatasources();
      if (selectedDatasourceId === id) {
        setSelectedDatasourceId(items[0]?.id || 'current');
      }
      resetDatasourceForm();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.datasource.deleteError'));
    }
  };

  const handleDownload = async () => {
    if (generatedFiles.length === 0) {
      return;
    }
    try {
      const exporter = new ModuleExporter(buildSchema());
      const blob = await exporter.exportAsZip();
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${buildSchema().name}-module.zip`;
      link.click();
      URL.revokeObjectURL(url);
      message.success(t('generator.wizard.downloadSuccess'));
    } catch {
      message.error(t('generator.wizard.downloadError'));
    }
  };

  const oneClickEnabled =
    currentStep === 3 && buildSchema().scope === 'business' && generatedFiles.length > 0;
  const canGenerateRegister = isAdmin || hasPerm('system:module:generate');
  const canOpenModuleManager = isAdmin || hasPerm('system:module:list');
  const summary = registerResult?.summary;
  const previewMenuTree = previewSchema ? buildMenuPreview(previewSchema) : [];
  const previewCompletenessIssues = previewSchema
    ? validateGeneratorCompleteness(previewSchema)
    : [];
  const previewBlockingIssue = previewCompletenessIssues.some((issue) => issue.level === 'error');
  const previewGeneratedFiles = previewSchema
    ? new ModuleExporter(previewSchema).generateAll()
    : [];
  const previewTranslationRows: TranslationPreviewRow[] = previewSchema
    ? Array.from(
        new Set([
          ...Object.keys(previewSchema.i18n.translations.zh),
          ...Object.keys(previewSchema.i18n.translations.en),
        ]),
      )
        .sort((a, b) => a.localeCompare(b))
        .map((key) => ({
          key,
          zh: previewSchema.i18n.translations.zh[key] || '',
          en: previewSchema.i18n.translations.en[key] || '',
        }))
    : [];
  const activationStatusKey = registerResult
    ? registerResult.module.status === 1
      ? 'generator.moduleManager.status.active'
      : registerResult.module.status === 2
        ? 'generator.moduleManager.status.uninstalled'
        : 'generator.moduleManager.status.pending'
    : '';

  const renderMenuPreview = (nodes: GeneratorMenuPreviewNode[]) => (
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
            {node.children.length > 0 ? renderMenuPreview(node.children) : null}
          </div>
        ))
      )}
    </div>
  );

  const submitGenerateAndRegister = async (overwrite = false) => {
    if (!previewSchema) {
      return;
    }
    if (previewSchema.scope !== 'business') {
      message.warning(t('generator.wizard.register.businessOnly'));
      return;
    }
    if (!isValidScopedModulePath(previewSchema.scope, previewSchema.name)) {
      void form.validate(['name']).catch(() => undefined);
      return;
    }
    try {
      await ensureOperationVerified();
      setRegistering(true);
      const result = await generateAndRegisterModule({
        schema: previewSchema,
        files: generatedFiles,
        overwrite,
      });
      setDynamicModuleDisabled(false);
      setRegisterResult(result);
      message.success(t('generator.wizard.register.success'));
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      if (isRequestError(error) && error.messageKey === 'module.dynamic.disabled') {
        setDynamicModuleDisabled(true);
        return;
      }
      if (
        isRequestError(error) &&
        (error.messageKey === 'module.generate.file_exists' ||
          error.messageKey === 'module.generate.already_exists') &&
        !overwrite
      ) {
        showAppModalConfirm({
          title: t('generator.wizard.register.overwriteTitle'),
          content: t('generator.wizard.register.overwriteContent'),
          onOk: () => {
            void submitGenerateAndRegister(true);
          },
        });
        return;
      }
      if (isRequestError(error)) {
        message.error(t(error.messageKey || 'request.failed'));
        return;
      }
      message.error(t('request.failed'));
    } finally {
      setRegistering(false);
    }
  };

  return (
    <PageContainer className="generator-wizard-page">
      <PageHeader title={t('generator.wizard.title')} />
      <Card className="page-panel generator-wizard-card">
        <Alert
          type="info"
          className="generator-wizard__section"
          content={t('generator.wizard.positioning')}
        />
        <Steps current={currentStep} className="generator-wizard__steps">
          <Steps.Step
            title={t('generator.wizard.step1.title')}
            description={t('generator.wizard.step1.desc')}
          />
          <Steps.Step
            title={t('generator.wizard.step2.title')}
            description={t('generator.wizard.step2.desc')}
          />
          <Steps.Step
            title={t('generator.wizard.step3.title')}
            description={t('generator.wizard.step3.desc')}
          />
          <Steps.Step
            title={t('generator.wizard.step4.title')}
            description={t('generator.wizard.step4.desc')}
          />
        </Steps>

        <Divider />

        {currentStep === 0 ? (
          <Form form={form} layout="vertical">
            <FormItem
              label={t('generator.wizard.moduleName')}
              field="name"
              rules={[
                { required: true, message: t('common.required') },
                {
                  validator: (value, callback) => {
                    const normalized = normalizeModulePath(String(value || ''));
                    const scope =
                      (form.getFieldValue('scope') as ModuleScope | undefined) || 'business';
                    if (!normalized || isValidScopedModulePath(scope, normalized)) {
                      callback();
                      return;
                    }
                    callback(t('module.generate.invalid_name'));
                  },
                },
              ]}
              extra={t('generator.wizard.moduleName.help')}
            >
              <Input
                placeholder="cmdb/host"
                onBlur={(event) => {
                  const normalizedName = normalizeModulePath(event.target.value);
                  form.setFieldValue('name', normalizedName);
                  if (!form.getFieldValue('metadata.businessContext' as keyof ModuleSchema)) {
                    form.setFieldValue(
                      'metadata.businessContext' as keyof ModuleSchema,
                      inferBusinessContextFromName(normalizedName),
                    );
                  }
                  syncModuleNameValidation();
                }}
              />
            </FormItem>
            <FormItem
              label={t('generator.wizard.sourceMode')}
              field="metadata.sourceMode"
              initialValue="manual"
            >
              <Select
                onChange={(value) => setSourceMode((value as 'manual' | 'database') || 'manual')}
              >
                <Select.Option value="manual">
                  {t('generator.wizard.sourceMode.manual')}
                </Select.Option>
                <Select.Option value="database">
                  {t('generator.wizard.sourceMode.database')}
                </Select.Option>
              </Select>
            </FormItem>
            {sourceMode === 'database' ? (
              <>
                <Card size="small" className="generator-wizard__section">
                  <Space direction="vertical" className="generator-wizard__full" size={12}>
                    <Space align="center" className="generator-wizard__toolbar">
                      <Typography.Text>{t('generator.datasource.selector')}</Typography.Text>
                      <Space>
                        <Button size="small" onClick={() => void loadTables(selectedDatasourceId)}>
                          <IconRefresh /> {t('common.refresh')}
                        </Button>
                        <PermissionAction
                          allowed={canManageDatasources}
                          tooltip={t('common.noPermissionAction')}
                        >
                          <Button
                            size="small"
                            onClick={() => {
                              resetDatasourceForm();
                              setDatasourceModalVisible(true);
                            }}
                          >
                            <IconPlus /> {t('generator.datasource.manage')}
                          </Button>
                        </PermissionAction>
                      </Space>
                    </Space>
                    <Select
                      value={selectedDatasourceId}
                      onChange={(value) => setSelectedDatasourceId(String(value))}
                      placeholder={t('generator.datasource.selectorPlaceholder')}
                    >
                      {datasources.map((item) => (
                        <Select.Option key={item.id} value={item.id}>
                          {item.name} · {item.databaseName}
                          {item.isCurrent ? ` · ${t('generator.datasource.currentTag')}` : ''}
                        </Select.Option>
                      ))}
                    </Select>
                    <Typography.Text type="secondary">
                      {t('generator.datasource.readonlyHint')}
                    </Typography.Text>
                  </Space>
                </Card>
                <FormItem
                  label={t('generator.wizard.sourceTable')}
                  field="metadata.sourceTable"
                  extra={t('generator.wizard.sourceTable.help')}
                >
                  <Select
                    allowClear
                    showSearch
                    loading={tableLoading}
                    placeholder={t('generator.wizard.sourceTable.placeholder')}
                    filterOption={(inputValue, option) =>
                      String(option?.props?.value || '')
                        .toLowerCase()
                        .includes(inputValue.toLowerCase())
                    }
                    onChange={(value) => {
                      const tableName = String(value || '').trim();
                      form.setFieldValue(
                        'metadata.sourceTable' as keyof ModuleSchema,
                        tableName || undefined,
                      );
                      if (!tableName) {
                        return;
                      }
                      void previewGeneratorTable(tableName, selectedDatasourceId)
                        .then((preview) => {
                          applyPreviewSuggestions(preview);
                        })
                        .catch(() => undefined);
                    }}
                  >
                    {tableOptions.map((item) => (
                      <Select.Option key={item.tableName} value={item.tableName}>
                        {item.tableName}
                        {item.comment ? ` · ${item.comment}` : ''}
                      </Select.Option>
                    ))}
                  </Select>
                </FormItem>
              </>
            ) : null}
            <FormItem
              label={t('generator.wizard.displayName')}
              field="displayName"
              rules={[{ required: true, message: t('common.required') }]}
            >
              <Input placeholder={t('generator.wizard.displayName.placeholder')} />
            </FormItem>
            <FormItem
              label={t('generator.wizard.displayNameEn')}
              field="displayNameEn"
              extra={t('generator.wizard.displayNameEn.help')}
            >
              <Input placeholder={t('generator.wizard.displayNameEn.placeholder')} />
            </FormItem>
            <FormItem
              label={t('generator.wizard.scope')}
              field="scope"
              initialValue="business"
              rules={[{ required: true, message: t('common.required') }]}
            >
              <Select
                onChange={() => {
                  syncModuleNameValidation();
                }}
              >
                <Select.Option value="business">
                  {t('generator.wizard.scope.business')}
                </Select.Option>
                <Select.Option value="system">{t('generator.wizard.scope.system')}</Select.Option>
              </Select>
            </FormItem>
            <Row gutter={16}>
              <Col xs={24} md={8}>
                <FormItem
                  label={t('generator.wizard.businessContext')}
                  field="metadata.businessContext"
                  extra={t('generator.wizard.businessContext.help')}
                >
                  <Input
                    placeholder={t('generator.wizard.businessContext.placeholder')}
                    onBlur={(event) => {
                      form.setFieldValue(
                        'metadata.businessContext' as keyof ModuleSchema,
                        normalizeBusinessContext(event.target.value),
                      );
                    }}
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={8}>
                <FormItem
                  label={t('generator.wizard.businessContextTitle')}
                  field="metadata.businessContextTitle"
                >
                  <Input placeholder={t('generator.wizard.businessContextTitle.placeholder')} />
                </FormItem>
              </Col>
              <Col xs={24} md={8}>
                <FormItem
                  label={t('generator.wizard.businessContextTitleEn')}
                  field="metadata.businessContextTitleEn"
                >
                  <Input placeholder={t('generator.wizard.businessContextTitleEn.placeholder')} />
                </FormItem>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} md={8}>
                <FormItem
                  label={t('generator.wizard.tableRole')}
                  field="metadata.tableRole"
                  initialValue="main"
                  extra={t('generator.wizard.tableRole.help')}
                >
                  <Select
                    onChange={(value) =>
                      setSelectedTableRole((value as BusinessTableRole) || 'main')
                    }
                  >
                    <Select.Option value="main">
                      {t('generator.wizard.tableRole.main')}
                    </Select.Option>
                    <Select.Option value="detail">
                      {t('generator.wizard.tableRole.detail')}
                    </Select.Option>
                    <Select.Option value="relation">
                      {t('generator.wizard.tableRole.relation')}
                    </Select.Option>
                    <Select.Option value="dictionary">
                      {t('generator.wizard.tableRole.dictionary')}
                    </Select.Option>
                  </Select>
                </FormItem>
              </Col>
              <Col xs={24} md={8}>
                <FormItem
                  label={t('generator.wizard.primaryTable')}
                  field="metadata.primaryTable"
                  extra={t('generator.wizard.primaryTable.help')}
                >
                  <Input placeholder={t('generator.wizard.primaryTable.placeholder')} />
                </FormItem>
              </Col>
              <Col xs={24} md={8}>
                <FormItem
                  label={t('generator.wizard.relationFields')}
                  extra={t('generator.wizard.relationFields.help')}
                >
                  <Space direction="vertical" className="generator-wizard__full">
                    <FormItem field="metadata.relationFromField" noStyle>
                      <Input
                        disabled={selectedTableRole !== 'relation'}
                        placeholder={t('generator.wizard.relationFromField.placeholder')}
                      />
                    </FormItem>
                    <FormItem field="metadata.relationToField" noStyle>
                      <Input
                        disabled={selectedTableRole !== 'relation'}
                        placeholder={t('generator.wizard.relationToField.placeholder')}
                      />
                    </FormItem>
                  </Space>
                </FormItem>
              </Col>
            </Row>
            <FormItem
              label={t('generator.wizard.templateLevel')}
              field="templateLevel"
              initialValue="enterprise"
              rules={[{ required: true, message: t('common.required') }]}
            >
              <Select>
                <Select.Option value="enterprise">
                  {t('generator.wizard.templateLevel.enterprise')}
                </Select.Option>
                <Select.Option value="basic">
                  {t('generator.wizard.templateLevel.basic')}
                </Select.Option>
              </Select>
            </FormItem>
            <FormItem
              label={t('generator.wizard.pageActionTemplate')}
              field="pageActionTemplate"
              initialValue="standard"
            >
              <Select>
                {PAGE_ACTION_TEMPLATE_DEFINITIONS.map((item) => (
                  <Select.Option key={item.key} value={item.key}>
                    {t(item.labelKey)}
                  </Select.Option>
                ))}
              </Select>
            </FormItem>
            <Card
              size="small"
              title={t('generator.wizard.p2plus.title')}
              className="generator-wizard__section"
            >
              <Row gutter={16}>
                <Col xs={24} md={8}>
                  <FormItem label={t('generator.wizard.templateVersion')}>
                    <Select
                      value={templateVersion}
                      onChange={(value) => setTemplateVersion((value as 'v1') || 'v1')}
                    >
                      <Select.Option value="v1">
                        {t('generator.wizard.templateVersion.v1')}
                      </Select.Option>
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={8}>
                  <FormItem
                    label={t('generator.wizard.enableDataScope')}
                    extra={t('generator.wizard.enableDataScope.help')}
                  >
                    <Select
                      value={enableDataScope ? 'enabled' : 'disabled'}
                      onChange={(value) => setEnableDataScope(value === 'enabled')}
                    >
                      <Select.Option value="enabled">{t('common.enabled')}</Select.Option>
                      <Select.Option value="disabled">{t('common.disabled')}</Select.Option>
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={8}>
                  <FormItem
                    label={t('generator.wizard.includeDashboardWidget')}
                    extra={t('generator.wizard.includeDashboardWidget.help')}
                  >
                    <Select
                      value={
                        ((form.getFieldValue('scope') as ModuleScope | undefined) || 'business') ===
                          'business' && selectedTableRole !== 'relation'
                          ? includeDashboardWidget
                            ? 'enabled'
                            : 'disabled'
                          : 'disabled'
                      }
                      disabled={
                        ((form.getFieldValue('scope') as ModuleScope | undefined) || 'business') !==
                          'business' || selectedTableRole === 'relation'
                      }
                      onChange={(value) => setIncludeDashboardWidget(value === 'enabled')}
                    >
                      <Select.Option value="enabled">{t('common.enabled')}</Select.Option>
                      <Select.Option value="disabled">{t('common.disabled')}</Select.Option>
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={8}>
                  <FormItem label={t('generator.wizard.dataScopeMode')}>
                    <Select
                      value={dataScopeMode}
                      disabled={!enableDataScope}
                      onChange={(value) => setDataScopeMode((value as DataScopeMode) || 'dept')}
                    >
                      <Select.Option value="dept">
                        {t('generator.wizard.dataScopeMode.dept')}
                      </Select.Option>
                      <Select.Option value="owner">
                        {t('generator.wizard.dataScopeMode.owner')}
                      </Select.Option>
                      <Select.Option value="tenant">
                        {t('generator.wizard.dataScopeMode.tenant')}
                      </Select.Option>
                      <Select.Option value="custom">
                        {t('generator.wizard.dataScopeMode.custom')}
                      </Select.Option>
                    </Select>
                  </FormItem>
                </Col>
              </Row>
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('generator.wizard.dependencies')}
                    extra={t('generator.wizard.dependencies.help')}
                  >
                    <Input.TextArea
                      value={dependencyModulesText}
                      onChange={setDependencyModulesText}
                      autoSize={{ minRows: 2, maxRows: 4 }}
                      placeholder={t('generator.wizard.dependencies.placeholder')}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('generator.wizard.relations')}
                    extra={t('generator.wizard.relations.help')}
                  >
                    <Input.TextArea
                      value={relationContractsText}
                      onChange={setRelationContractsText}
                      autoSize={{ minRows: 2, maxRows: 4 }}
                      placeholder={t('generator.wizard.relations.placeholder')}
                    />
                  </FormItem>
                </Col>
              </Row>
            </Card>
            <FormItem
              label={t('generator.wizard.parentMenu')}
              field="parentMenu"
              extra={t('generator.wizard.parentMenu.help')}
            >
              <Input
                placeholder={t('generator.wizard.parentMenu.placeholder')}
                onBlur={(event) => {
                  form.setFieldValue('parentMenu', normalizeMenuPath(event.target.value));
                }}
              />
            </FormItem>
            <FormItem label={t('generator.wizard.owner')} field="metadata.owner">
              <Input placeholder={t('generator.wizard.owner.placeholder')} />
            </FormItem>
            <FormItem label={t('generator.wizard.boundedContext')} field="metadata.boundedContext">
              <Input placeholder={t('generator.wizard.boundedContext.placeholder')} />
            </FormItem>
            <FormItem label={t('generator.wizard.summary')} field="metadata.summary">
              <Input.TextArea
                autoSize={{ minRows: 2, maxRows: 4 }}
                placeholder={t('generator.wizard.summary.placeholder')}
              />
            </FormItem>
            <Button type="primary" onClick={handleBasicInfoSubmit}>
              {t('common.next')}
            </Button>
          </Form>
        ) : null}

        {currentStep === 1 ? (
          <div>
            <Typography.Title heading={5}>{t('generator.wizard.step2.title')}</Typography.Title>
            <Typography.Text type="secondary" className="generator-wizard__description">
              {t('generator.wizard.step2.desc')}
            </Typography.Text>
            <FieldEditor fields={fields} onChange={setFields} />
            <Space className="generator-wizard__actions">
              <Button onClick={() => setCurrentStep(0)}>{t('common.previous')}</Button>
              <Button
                type="primary"
                onClick={() => setCurrentStep(2)}
                disabled={fields.length === 0}
              >
                {t('common.next')}
              </Button>
            </Space>
          </div>
        ) : null}

        {currentStep === 2 && previewSchema ? (
          <div>
            <Typography.Title heading={5}>{t('generator.wizard.step3.title')}</Typography.Title>
            <Typography.Text type="secondary" className="generator-wizard__description">
              {t('generator.wizard.step3.desc')}
            </Typography.Text>

            <Row gutter={16}>
              <Col xs={24} lg={12}>
                <Card
                  title={t('generator.wizard.step3.actions')}
                  className="generator-wizard__section"
                >
                  <Typography.Text type="secondary" className="generator-wizard__subdescription">
                    {t(
                      PAGE_ACTION_TEMPLATE_DEFINITIONS.find(
                        (item) => item.key === selectedActionTemplate,
                      )?.descriptionKey || 'generator.actionTemplates.standard.desc',
                    )}
                  </Typography.Text>
                  <Checkbox.Group
                    value={
                      (form.getFieldValue('pageActions' as keyof ModuleSchema) as
                        | PageActionKey[]
                        | undefined) ?? previewSchema.pageActions
                    }
                    disabled={previewSchema.metadata?.tableRole === 'relation'}
                    options={[
                      'view',
                      'detail',
                      'create',
                      'update',
                      'delete',
                      'export',
                      'import',
                    ].map((item) => ({
                      label: t(`generator.pageActions.${item}`),
                      value: item,
                    }))}
                    onChange={(value) => {
                      form.setFieldValue('pageActions', value as PageActionKey[]);
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
                        count: previewSchema.model.fields.filter(
                          (field) => field.validation?.unique,
                        ).length,
                      })}
                    </Tag>
                    <Tag color="purple">
                      {t('generator.wizard.step3.enumCount', {
                        count: previewSchema.model.fields.filter((field) => field.type === 'enum')
                          .length,
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

            <Card
              title={t('generator.wizard.step3.permissions')}
              className="generator-wizard__section"
            >
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
                  {renderMenuPreview(previewMenuTree)}
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
                    <Button size="small" onClick={handleExportTranslations}>
                      {t('generator.wizard.step3.translationPreview.export')}
                    </Button>
                    <Button size="small" onClick={handleImportTranslations}>
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
                          onChange={(nextValue) =>
                            updateTranslationOverride(record.key, 'zh', nextValue)
                          }
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
                          onChange={(nextValue) =>
                            updateTranslationOverride(record.key, 'en', nextValue)
                          }
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
              <Button onClick={() => setCurrentStep(1)}>{t('common.previous')}</Button>
              <Button type="primary" onClick={handleGenerate} disabled={previewBlockingIssue}>
                {t('generator.wizard.generate')}
              </Button>
            </Space>
          </div>
        ) : null}

        {currentStep === 3 && previewSchema ? (
          <div>
            <Typography.Title heading={5}>{t('generator.wizard.step4.title')}</Typography.Title>

            <Card className="generator-wizard__section">
              <Space wrap>
                <Typography.Text>
                  {t('generator.wizard.generatedFiles', { count: generatedFiles.length })}
                </Typography.Text>
                <Tag color="green">
                  {t('generator.wizard.totalLines', {
                    lines: generatedFiles.reduce(
                      (sum, file) => sum + file.content.split('\n').length,
                      0,
                    ),
                  })}
                </Tag>
                <Tag color="arcoblue">
                  {t('generator.wizard.step3.actionCount', {
                    count: previewSchema.pageActions?.length || 0,
                  })}
                </Tag>
              </Space>
            </Card>

            <Space className="generator-wizard__section">
              <PermissionAction
                allowed={canGenerateRegister}
                tooltip={t('common.noPermissionAction')}
              >
                <Button
                  type="primary"
                  status="success"
                  loading={registering}
                  disabled={!oneClickEnabled}
                  onClick={() => {
                    void submitGenerateAndRegister();
                  }}
                >
                  {t('generator.wizard.register.submit')}
                </Button>
              </PermissionAction>
              <Button type="primary" onClick={handleDownload}>
                <IconDownload /> {t('generator.wizard.download')}
              </Button>
              <Button onClick={() => setShowPreview(true)}>
                <IconCode /> {t('generator.wizard.preview')}
              </Button>
              <Button onClick={() => setCurrentStep(2)}>{t('common.previous')}</Button>
            </Space>

            <CodePreview
              visible={showPreview}
              files={generatedFiles}
              onClose={() => setShowPreview(false)}
            />

            {dynamicModuleDisabled ? (
              <Alert
                type="warning"
                title={t('generator.wizard.register.disabledTitle')}
                content={t('generator.wizard.register.disabledHint')}
                className="generator-wizard__section"
              />
            ) : null}

            {registerResult && summary ? (
              <div className="generator-wizard__result">
                <Alert
                  type="success"
                  title={t('generator.wizard.result.pendingActivation')}
                  content={t('generator.wizard.result.pendingActivationDesc')}
                  className="generator-wizard__section"
                />
                <Card
                  title={t('generator.wizard.result.title')}
                  className="generator-wizard__section"
                >
                  <Space direction="vertical" className="generator-wizard__full">
                    <Typography.Text>
                      {t('generator.wizard.result.moduleKey')}: {summary.moduleKey}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.parentMenu')}:{' '}
                      {summary.parentMenuPath || t('generator.wizard.result.parentMenu.topLevel')}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.routePath')}: {summary.routePath}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.routeName')}: {summary.routeName}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.componentKey')}: {summary.componentKey}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.permissionPrefix')}: {summary.permissionPrefix}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.backendPath')}: {summary.backendModulePath}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.frontendPath')}: {summary.frontendModulePath}
                    </Typography.Text>
                    <Typography.Text>
                      {t('generator.wizard.result.schemaPath')}: {summary.schemaPath}
                    </Typography.Text>
                    <Tag color={registerResult.module.status === 3 ? 'orange' : 'green'}>
                      {t(activationStatusKey)}
                    </Tag>
                  </Space>
                </Card>
                <Card
                  title={t('generator.wizard.result.contractTitle')}
                  className="generator-wizard__section"
                >
                  <Space direction="vertical" className="generator-wizard__full">
                    <Space wrap>
                      <Tag color="arcoblue">
                        {t('generator.wizard.result.templateVersion')}:{' '}
                        {summary.contract.templateVersion}
                      </Tag>
                      <Tag color={summary.contract.dataScopeEnabled ? 'green' : 'gray'}>
                        {t('generator.wizard.result.dataScope')}: {summary.contract.dataScopeMode}
                      </Tag>
                      <Tag color="blue">
                        {t('generator.wizard.result.dependencyCount', {
                          count: summary.contract.dependencyCount,
                        })}
                      </Tag>
                      <Tag color="orange">
                        {t('generator.wizard.result.relationCount', {
                          count: summary.contract.relationCount,
                        })}
                      </Tag>
                    </Space>
                    {(summary.contract.dependencies?.length || 0) > 0 ? (
                      <Space direction="vertical" className="generator-wizard__full">
                        <Typography.Text type="secondary">
                          {t('generator.wizard.result.dependencies')}
                        </Typography.Text>
                        {summary.contract.dependencies?.map((dependency) => (
                          <Typography.Text key={dependency.module} code>
                            {dependency.module}
                            {dependency.reason ? ` · ${dependency.reason}` : ''}
                          </Typography.Text>
                        ))}
                      </Space>
                    ) : null}
                    {(summary.contract.relations?.length || 0) > 0 ? (
                      <Space direction="vertical" className="generator-wizard__full">
                        <Typography.Text type="secondary">
                          {t('generator.wizard.result.relations')}
                        </Typography.Text>
                        {summary.contract.relations?.map((relation) => (
                          <Typography.Text key={`${relation.name}-${relation.targetModule}`} code>
                            {relation.name} · {relation.type} · {relation.targetModule}
                          </Typography.Text>
                        ))}
                      </Space>
                    ) : null}
                  </Space>
                </Card>
                <Card
                  title={t('generator.wizard.result.verifications')}
                  className="generator-wizard__section"
                >
                  <Space direction="vertical" className="generator-wizard__full">
                    {summary.verifications.map((item) => (
                      <Space
                        key={`${item.code}-${item.detail}`}
                        align="start"
                        className="generator-wizard__verification-row"
                      >
                        <Space>
                          <Tag
                            color={
                              item.status === 'pass'
                                ? 'green'
                                : item.status === 'warn'
                                  ? 'orange'
                                  : 'arcoblue'
                            }
                          >
                            {t(`generator.wizard.result.verificationStatus.${item.status}`)}
                          </Tag>
                          <Typography.Text>{t(item.messageKey)}</Typography.Text>
                        </Space>
                        <Typography.Text type="secondary">{item.detail}</Typography.Text>
                      </Space>
                    ))}
                  </Space>
                </Card>
                <Space wrap>
                  {canOpenModuleManager ? (
                    <Button onClick={() => navigate('/system/modules')}>
                      {t('generator.wizard.result.openModuleManager')}
                    </Button>
                  ) : null}
                  <Button
                    onClick={() => {
                      setRegisterResult(null);
                      setGeneratedFiles([]);
                      setCurrentStep(0);
                    }}
                  >
                    {t('generator.wizard.result.generateAnother')}
                  </Button>
                </Space>
              </div>
            ) : null}
          </div>
        ) : null}
      </Card>
      <AppModal
        title={t('generator.datasource.manageTitle')}
        visible={datasourceModalVisible}
        onCancel={() => setDatasourceModalVisible(false)}
        footer={null}
        size="xl"
      >
        <Space direction="vertical" className="generator-wizard__full" size={16}>
          <Table
            pagination={false}
            rowKey="id"
            data={datasources.filter((item) => !item.isCurrent)}
            columns={[
              { title: t('generator.datasource.name'), dataIndex: 'name' },
              { title: t('generator.datasource.databaseName'), dataIndex: 'databaseName' },
              { title: t('generator.datasource.host'), dataIndex: 'host' },
              {
                title: t('generator.datasource.status'),
                dataIndex: 'status',
                render: (value: number) => (
                  <Tag color={value === 1 ? 'green' : 'gray'}>
                    {value === 1
                      ? t('system.user.status.enabled')
                      : t('system.user.status.disabled')}
                  </Tag>
                ),
              },
              {
                title: t('common.action'),
                render: (_: unknown, record: GeneratorDatasource) => (
                  <Space>
                    <Button size="mini" type="text" onClick={() => handleEditDatasource(record)}>
                      <IconEdit /> {t('common.edit')}
                    </Button>
                    <Button
                      size="mini"
                      type="text"
                      onClick={() => void handleTestDatasource(record.id)}
                    >
                      <IconCode /> {t('generator.datasource.test')}
                    </Button>
                    <Popconfirm
                      title={t('generator.datasource.deleteConfirm')}
                      onOk={() => handleDeleteDatasource(record.id)}
                    >
                      <Button size="mini" type="text" status="danger">
                        <IconDelete /> {t('common.delete')}
                      </Button>
                    </Popconfirm>
                  </Space>
                ),
              },
            ]}
            noDataElement={t('generator.datasource.empty')}
          />
          <Card
            size="small"
            title={
              editingDatasourceId
                ? t('generator.datasource.editTitle')
                : t('generator.datasource.createTitle')
            }
          >
            <Form
              form={datasourceForm}
              layout="vertical"
              onSubmit={() => {
                void handleSaveDatasource();
              }}
            >
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <FormItem
                    field="name"
                    label={t('generator.datasource.name')}
                    rules={[{ required: true, message: t('common.required') }]}
                  >
                    <Input
                      placeholder={t('generator.datasource.namePlaceholder')}
                      onPressEnter={() => datasourceForm.submit()}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    field="driver"
                    label={t('generator.datasource.driver')}
                    initialValue="mysql"
                  >
                    <Select>
                      <Select.Option value="mysql">MySQL</Select.Option>
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    field="host"
                    label={t('generator.datasource.host')}
                    rules={[{ required: true, message: t('common.required') }]}
                  >
                    <Input
                      placeholder={t('generator.datasource.hostPlaceholder')}
                      onPressEnter={() => datasourceForm.submit()}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    field="port"
                    label={t('generator.datasource.port')}
                    initialValue={3306}
                    rules={[{ required: true, message: t('common.required') }]}
                  >
                    <InputNumber
                      placeholder={t('generator.datasource.portPlaceholder')}
                      className="generator-wizard__number-input"
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    field="databaseName"
                    label={t('generator.datasource.databaseName')}
                    rules={[{ required: true, message: t('common.required') }]}
                  >
                    <Input
                      placeholder={t('generator.datasource.databasePlaceholder')}
                      onPressEnter={() => datasourceForm.submit()}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    field="username"
                    label={t('generator.datasource.username')}
                    rules={[{ required: true, message: t('common.required') }]}
                  >
                    <Input
                      placeholder={t('generator.datasource.usernamePlaceholder')}
                      onPressEnter={() => datasourceForm.submit()}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    field="password"
                    label={t('generator.datasource.password')}
                    extra={
                      editingDatasourceId ? t('generator.datasource.passwordOptional') : undefined
                    }
                  >
                    <Input.Password
                      placeholder={t('generator.datasource.passwordPlaceholder')}
                      onPressEnter={() => datasourceForm.submit()}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    field="status"
                    label={t('generator.datasource.status')}
                    initialValue={1}
                  >
                    <Select>
                      <Select.Option value={1}>{t('system.user.status.enabled')}</Select.Option>
                      <Select.Option value={0}>{t('system.user.status.disabled')}</Select.Option>
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24}>
                  <FormItem field="remark" label={t('i18n.remark')}>
                    <Input.TextArea autoSize={{ minRows: 2, maxRows: 3 }} />
                  </FormItem>
                </Col>
              </Row>
              <Space>
                <Button onClick={resetDatasourceForm}>{t('common.reset')}</Button>
                <Button
                  type="primary"
                  loading={datasourceSaving}
                  onClick={() => void handleSaveDatasource()}
                >
                  {editingDatasourceId ? t('common.save') : t('common.create')}
                </Button>
              </Space>
            </Form>
          </Card>
        </Space>
      </AppModal>
    </PageContainer>
  );
};

export default ModuleWizard;
