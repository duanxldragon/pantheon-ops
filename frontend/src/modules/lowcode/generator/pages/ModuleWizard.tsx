import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Form,
  Grid,
  Input,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { IconCode, IconDownload, IconPlus, IconRefresh } from '@arco-design/web-react/icon';
import { message } from '../../../../components/feedback/message';
import { isRequestError, ensureOperationVerified } from '../../../../api/request';
import PermissionAction from '../../../../components/patterns/PermissionAction';
import {
  AppTable,
  AppModal,
  buildStandardPagination,
  GovernanceSummaryBar,
  ListHeaderActions,
  PageContainer,
  showAppModalConfirm,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { auditPendingActivations, getModuleStatus } from '../../../lowcode/dynamicmodule/api';

import {
  createGeneratorDatasource,
  deleteGeneratorDatasource,
  downloadGeneratedSource,
  generateAndRegisterModule,
  listGeneratorDatasources,
  listGeneratorTables,
  previewGeneratorTable,
  previewGeneratedFiles as requestPreviewGeneratedFiles,
  testGeneratorDatasource,
  updateGeneratorDatasource,
  type GenerateAndRegisterResp,
  type GeneratedFile,
  type GeneratorDatasource,
  type GeneratorTableOption,
  type UpsertGeneratorDatasourcePayload,
} from '../api';
import { FieldEditor } from '../../components/FieldEditor';
import { CodePreview } from '../../components/CodePreview';
import DatasourceManagerModal from './components/DatasourceManagerModal';
import MenuPreviewTree from './components/MenuPreviewTree';
import '../../../system/components/shared/list-page.css';
import './ModuleWizard.css';
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
  type ModuleField,
  type ModuleListLayoutConfig,
  type ModuleRelationType,
  type ModuleSchema,
  type ModuleScope,
  type PageActionKey,
  type PageActionTemplate,
  type TemplateLevel,
} from '../schema';
import { SECONDARY_VERIFY_CANCELLED_ERROR } from '../../../../components/feedback/secondaryVerifyController';

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

interface ResolvedModuleMetadata {
  businessContext?: string;
  businessContextTitle?: string;
  businessContextTitleEn?: string;
  tableRole: BusinessTableRole;
  primaryTable?: string;
  relationFromField?: string;
  relationToField?: string;
  boundedContext?: string;
  owner?: string;
  summary?: string;
  sourceMode: 'manual' | 'database';
  sourceDatasourceId?: string;
  sourceDatasourceName?: string;
  sourceTable?: string;
  autoRecycle: boolean;
}

interface BuildModuleSchemaInput {
  values: Partial<ModuleSchema>;
  metadata: ResolvedModuleMetadata;
  fields: ModuleField[];
  templateVersion: 'v1';
  dependencyModulesText: string;
  relationContractsText: string;
  enableDataScope: boolean;
  includeDashboardWidget: boolean;
  listLayout: ModuleListLayoutConfig;
  dataScopeMode: DataScopeMode;
  translationOverrides: Record<string, TranslationOverride>;
  actionLabel: (action: Exclude<PageActionKey, 'detail'>, locale: 'zh-CN' | 'en-US') => string;
}

interface CsvParseState {
  current: string;
  row: string[];
  rows: string[][];
  inQuotes: boolean;
}

type RegistrationErrorKind = 'cancelled' | 'disabled' | 'overwrite' | 'request' | 'unknown';

function resolvePreferredDatasourceId(
  items: Array<Pick<GeneratorDatasource, 'id' | 'isCurrent'>>,
  selectedDatasourceId: string,
): string {
  return (
    items.find((item) => item.id === selectedDatasourceId)?.id ||
    items.find((item) => item.isCurrent)?.id ||
    items[0]?.id ||
    'current'
  );
}

function mergeTranslationOverrides(
  current: Record<string, TranslationOverride>,
  rows: string[][],
): Record<string, TranslationOverride> {
  return rows.reduce<Record<string, TranslationOverride>>(
    (acc, row) => {
      const [key = '', zh = '', en = ''] = row;
      const normalizedKey = key.trim();
      if (!normalizedKey) {
        return acc;
      }
      acc[normalizedKey] = {
        ...acc[normalizedKey],
        zh,
        en,
      };
      return acc;
    },
    { ...current },
  );
}

function escapeCsvCell(value: string): string {
  const normalized = String(value ?? '');
  if (!/[",\n\r]/.test(normalized)) {
    return normalized;
  }
  return `"${normalized.replaceAll('"', '""')}"`;
}

function verificationStatusColor(status: string): string {
  if (status === 'pass') {
    return 'green';
  }
  if (status === 'warn') {
    return 'orange';
  }
  return 'arcoblue';
}

function moduleActivationStatusKey(status: number): string {
  if (status === 1) {
    return 'generator.moduleManager.status.active';
  }
  if (status === 2) {
    return 'generator.moduleManager.status.uninstalled';
  }
  return 'generator.moduleManager.status.pending';
}

function appendCsvRow(state: CsvParseState) {
  state.row.push(state.current);
  if (state.row.some((cell) => cell.trim() !== '')) {
    state.rows.push(state.row);
  }
  state.row = [];
  state.current = '';
}

function consumeCsvCharacter(
  state: CsvParseState,
  char: string,
  next: string | undefined,
): boolean {
  if (char === '"' && state.inQuotes && next === '"') {
    state.current += '"';
    return true;
  }
  if (char === '"') {
    state.inQuotes = !state.inQuotes;
    return false;
  }
  if (char === ',' && !state.inQuotes) {
    state.row.push(state.current);
    state.current = '';
    return false;
  }
  if ((char === '\n' || char === '\r') && !state.inQuotes) {
    appendCsvRow(state);
    return char === '\r' && next === '\n';
  }
  state.current += char;
  return false;
}

function parseCsvRows(content: string): string[][] {
  const state: CsvParseState = {
    current: '',
    row: [],
    rows: [],
    inQuotes: false,
  };

  for (let index = 0; index < content.length; index += 1) {
    if (consumeCsvCharacter(state, content[index], content[index + 1])) {
      index += 1;
    }
  }

  appendCsvRow(state);
  return state.rows;
}

function readFileText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result;
      resolve(typeof result === 'string' ? result : '');
    };
    reader.onerror = () => {
      reject(reader.error || new Error('Failed to read file'));
    };
    reader.readAsText(file, 'utf-8');
  });
}

function parseDependencyModules(value: string) {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .map((module) => ({ module, required: true }));
}

function parseRelationContracts(value: string) {
  return value
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
        targetLabelField = '',
        lookupApi = '',
        lookupValueField = '',
        junctionTable = '',
      ] = item.split('|').map((part) => part.trim());
      return {
        name,
        type: type as ModuleRelationType,
        targetModule,
        localField,
        targetField,
        targetLabelField: targetLabelField || undefined,
        lookupApi: lookupApi || undefined,
        lookupValueField: lookupValueField || undefined,
        junctionTable: junctionTable || undefined,
      };
    });
}

function buildFieldTranslations(
  scope: ModuleScope,
  name: string,
  fields: ModuleField[],
  locale: 'zh' | 'en',
  titleKey: string,
  title: string,
): Record<string, string> {
  return fields.reduce<Record<string, string>>(
    (translations, field) => {
      const isEnglish = locale === 'en';
      translations[buildFieldLabelKey(scope, name, field.name)] =
        (isEnglish ? field.labelEn : field.label) || field.label;
      const placeholder = isEnglish ? field.placeholderEn || field.placeholder : field.placeholder;
      if (placeholder) {
        translations[buildFieldPlaceholderKey(scope, name, field.name)] = placeholder;
      }
      const helpText = isEnglish ? field.helpTextEn || field.helpText : field.helpText;
      if (helpText) {
        translations[buildFieldHelpTextKey(scope, name, field.name)] = helpText;
      }
      for (const item of field.enumOptions ?? []) {
        translations[buildEnumOptionKey(scope, name, field.name, item.value)] =
          (isEnglish ? item.labelEn : item.label) || item.label;
      }
      return translations;
    },
    { [titleKey]: title },
  );
}

function addDashboardTranslations(
  translations: { zh: Record<string, string>; en: Record<string, string> },
  enabled: boolean,
  key: string,
  displayName: string,
  displayNameEn: string,
) {
  if (!enabled) {
    return;
  }
  translations.zh[key] = `进入${displayName}`;
  translations.en[key] = `Open ${displayNameEn}`;
}

function addMenuGroupTranslations(
  translations: { zh: Record<string, string>; en: Record<string, string> },
  scope: ModuleScope,
  moduleSegments: string[],
  businessContext: string,
  businessContextTitle: string,
  businessContextTitleEn: string,
) {
  moduleSegments.slice(0, -1).forEach((_, index) => {
    const groupSegments = moduleSegments.slice(0, index + 1);
    const groupTitleKey = buildMenuGroupTitleKey(scope, groupSegments);
    const isBusinessContext = index === 0 && groupSegments[0] === businessContext;
    const groupDisplayName = isBusinessContext
      ? businessContextTitle
      : inferMenuGroupDisplayName(groupSegments.at(-1) ?? '');
    const groupDisplayNameEn = isBusinessContext ? businessContextTitleEn : groupDisplayName;
    translations.zh[groupTitleKey] = translations.zh[groupTitleKey] || groupDisplayName;
    translations.en[groupTitleKey] = translations.en[groupTitleKey] || groupDisplayNameEn;
  });
}

function addActionTranslations(
  translations: { zh: Record<string, string>; en: Record<string, string> },
  scope: ModuleScope,
  name: string,
  pageActions: PageActionKey[],
  displayName: string,
  displayNameEn: string,
  actionLabel: BuildModuleSchemaInput['actionLabel'],
) {
  pageActions
    .filter((action) => action !== 'detail')
    .forEach((action) => {
      const key = buildPermissionTitleKey(scope, name, action);
      translations.zh[key] = `${actionLabel(action, 'zh-CN')}${displayName}`;
      translations.en[key] = `${actionLabel(action, 'en-US')} ${displayNameEn}`;
    });

  for (const action of ['create', 'update', 'delete'] as const) {
    const key = buildAuditActionKey(scope, name, action);
    translations.zh[key] = `${actionLabel(action, 'zh-CN')}${displayName}`;
    translations.en[key] = `${actionLabel(action, 'en-US')} ${displayNameEn}`;
  }
}

function applyTranslationOverrides(
  translations: { zh: Record<string, string>; en: Record<string, string> },
  overrides: Record<string, TranslationOverride>,
) {
  Object.entries(overrides).forEach(([key, override]) => {
    if (Object.hasOwn(translations.zh, key) && override.zh !== undefined) {
      translations.zh[key] = override.zh;
    }
    if (Object.hasOwn(translations.en, key) && override.en !== undefined) {
      translations.en[key] = override.en;
    }
  });
}

function buildListLayout(
  requested: ModuleListLayoutConfig,
  metadata: ResolvedModuleMetadata,
  fields: ModuleField[],
  pageActions: PageActionKey[],
  enableExport: boolean,
  enableImport: boolean,
): ModuleListLayoutConfig {
  const hasSearchableFields = fields.some((field) => field.searchable);
  const hasVisibleListFields = fields.some((field) => field.visibleInList !== false);
  return {
    governance:
      requested.governance !== false &&
      Boolean(metadata.primaryTable || metadata.relationFromField || metadata.relationToField),
    search: hasSearchableFields && requested.search !== false,
    headerActions:
      (enableExport || enableImport || pageActions.includes('create')) &&
      requested.headerActions !== false,
    batchActions:
      pageActions.some((action) => ['update', 'delete'].includes(action)) &&
      requested.batchActions !== false,
    rowActions:
      hasVisibleListFields &&
      pageActions.some((action) => ['view', 'detail', 'update', 'delete'].includes(action)) &&
      requested.rowActions !== false,
  };
}

function buildModuleSchema(input: BuildModuleSchemaInput): ModuleSchema {
  const { values, metadata } = input;
  const name = normalizeModulePath(values.name || '');
  const displayName = values.displayName || '';
  const displayNameEn = values.displayNameEn || displayName;
  const scope = (values.scope as ModuleScope | undefined) || 'business';
  const templateLevel = (values.templateLevel as TemplateLevel | undefined) || 'enterprise';
  const tableRole = metadata.tableRole || 'main';
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
  const normalizedFields = normalizeFields(input.fields);
  const titleKey = buildTitleKey(scope, name);
  const businessContext = normalizeBusinessContext(
    metadata.businessContext || inferBusinessContextFromName(name),
  );
  const businessContextTitle =
    metadata.businessContextTitle || inferMenuGroupDisplayName(businessContext);
  const businessContextTitleEn = metadata.businessContextTitleEn || businessContextTitle;
  const translations = {
    zh: buildFieldTranslations(scope, name, normalizedFields, 'zh', titleKey, displayName),
    en: buildFieldTranslations(scope, name, normalizedFields, 'en', titleKey, displayNameEn),
  };
  addDashboardTranslations(
    translations,
    canAttachDashboardWidget && input.includeDashboardWidget,
    buildDashboardQuickActionDescriptionKey(scope, name),
    displayName,
    displayNameEn,
  );
  addMenuGroupTranslations(
    translations,
    scope,
    name.split('/').filter(Boolean),
    businessContext,
    businessContextTitle,
    businessContextTitleEn,
  );
  addActionTranslations(
    translations,
    scope,
    name,
    pageActions,
    displayName,
    displayNameEn,
    input.actionLabel,
  );
  applyTranslationOverrides(translations, input.translationOverrides);

  const schema: ModuleSchema = {
    name,
    templateVersion: input.templateVersion,
    displayName,
    description: values.description,
    displayNameEn,
    scope,
    templateLevel,
    parentMenu: normalizeMenuPath(values.parentMenu),
    pageActionTemplate,
    pageActions,
    dependencies: parseDependencyModules(input.dependencyModulesText),
    relations: parseRelationContracts(input.relationContractsText),
    dataScopeMode: input.enableDataScope ? input.dataScopeMode : 'none',
    listLayout: buildListLayout(
      input.listLayout,
      metadata,
      normalizedFields,
      pageActions,
      enableExport,
      enableImport,
    ),
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
      autoRecycle: metadata.autoRecycle,
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
      translations,
    },
    enableExport,
    enableImport,
    enableAudit: templateLevel === 'enterprise',
    enableDataScope: input.enableDataScope,
    includeDashboardWidget: canAttachDashboardWidget ? input.includeDashboardWidget : false,
  };

  schema.menus = generateDefaultMenus(schema);
  schema.permissions = generateDefaultPermissions(schema);
  return schema;
}

function classifyRegistrationError(error: unknown, overwrite: boolean): RegistrationErrorKind {
  if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
    return 'cancelled';
  }
  if (isRequestError(error) && error.messageKey === 'module.dynamic.disabled') {
    return 'disabled';
  }
  if (
    isRequestError(error) &&
    (error.messageKey === 'module.generate.file_exists' ||
      error.messageKey === 'module.generate.already_exists') &&
    !overwrite
  ) {
    return 'overwrite';
  }
  return isRequestError(error) ? 'request' : 'unknown';
}

function resolvePreviewSchema(currentStep: number, buildSchema: () => ModuleSchema) {
  return currentStep >= 2 ? buildSchema() : null;
}

function schemaKey(schema: ModuleSchema | null): string {
  return schema ? JSON.stringify(schema) : '';
}

function resolvePreviewMenuTree(schema: ModuleSchema | null) {
  return schema ? buildMenuPreview(schema) : [];
}

function resolvePreviewCompletenessIssues(schema: ModuleSchema | null) {
  return schema ? validateGeneratorCompleteness(schema) : [];
}

function resolvePreviewGeneratedFiles(
  generatedSchemaKey: string,
  previewSchemaKey: string,
  generatedFiles: GeneratedFile[],
) {
  return generatedSchemaKey === previewSchemaKey ? generatedFiles : [];
}

function buildTranslationPreviewRows(schema: ModuleSchema | null): TranslationPreviewRow[] {
  if (!schema) {
    return [];
  }
  return Array.from(
    new Set([
      ...Object.keys(schema.i18n.translations.zh),
      ...Object.keys(schema.i18n.translations.en),
    ]),
  )
    .sort((a, b) => a.localeCompare(b))
    .map((key) => ({
      key,
      zh: schema.i18n.translations.zh[key] || '',
      en: schema.i18n.translations.en[key] || '',
    }));
}

function renderWhen(condition: boolean, node: React.ReactNode): React.ReactNode {
  return condition ? node : null;
}

function renderSchemaStep(
  currentStep: number,
  expectedStep: number,
  schema: ModuleSchema | null,
  render: (schema: ModuleSchema) => React.ReactNode,
): React.ReactNode {
  if (currentStep !== expectedStep || !schema) {
    return null;
  }
  return render(schema);
}

function renderRegistrationResult(
  result: GenerateAndRegisterResp | null,
  summary: GenerateAndRegisterResp['summary'] | undefined,
  render: (
    result: GenerateAndRegisterResp,
    summary: GenerateAndRegisterResp['summary'],
  ) => React.ReactNode,
): React.ReactNode {
  if (!result || !summary) {
    return null;
  }
  return render(result, summary);
}

function choose<T>(condition: boolean, whenTrue: T, whenFalse: T): T {
  return condition ? whenTrue : whenFalse;
}

function valueOr<T>(value: T | null | undefined | false | '', fallback: T): T {
  return value || fallback;
}

function hasAccess(isAdmin: boolean, hasPermission: boolean): boolean {
  return isAdmin || hasPermission;
}

function buildWizardStepClass(index: number, currentStep: number): string {
  const activeClass = index === currentStep ? ' generator-wizard__step-card--active' : '';
  const doneClass = index < currentStep ? ' generator-wizard__step-card--done' : '';
  return `generator-wizard__step-card${activeClass}${doneClass}`;
}

function resolveSelectedActionTemplate(values: Partial<ModuleSchema>): PageActionTemplate {
  return (values.pageActionTemplate as PageActionTemplate | undefined) || 'standard';
}

function isOneClickRegistrationEnabled(
  currentStep: number,
  schemaScope: ModuleScope,
  generatedSchemaKey: string,
  previewSchemaKey: string,
  generatedFileCount: number,
): boolean {
  return (
    currentStep === 3 &&
    schemaScope === 'business' &&
    generatedSchemaKey === previewSchemaKey &&
    generatedFileCount > 0
  );
}

const ModuleWizard: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { hasPerm, isAdmin } = usePermission();
  const [currentStep, setCurrentStep] = useState(0);
  const [form] = Form.useForm<Partial<ModuleSchema>>();
  const [fields, setFields] = useState<ModuleField[]>([]);
  const [generatedFiles, setGeneratedFiles] = useState<GeneratedFile[]>([]);
  const [generatedSchemaKey, setGeneratedSchemaKey] = useState('');
  const [showPreview, setShowPreview] = useState(false);
  const [registering, setRegistering] = useState(false);
  const [auditingActivation, setAuditingActivation] = useState(false);
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
  const [listLayout, setListLayout] = useState<ModuleListLayoutConfig>({
    governance: true,
    search: true,
    headerActions: true,
    batchActions: true,
    rowActions: true,
  });
  const [translationPreviewPagination, setTranslationPreviewPagination] = useState({
    current: 1,
    pageSize: 8,
  });
  const [dataScopeMode, setDataScopeMode] = useState<DataScopeMode>('dept');
  const [translationOverrides, setTranslationOverrides] = useState<
    Record<string, TranslationOverride>
  >({});
  const [datasourceForm] = Form.useForm<UpsertGeneratorDatasourcePayload>();
  const canManageDatasources = hasAccess(isAdmin, hasPerm('system:generator:datasource:manage'));
  const actionLabel = (action: Exclude<PageActionKey, 'detail'>, locale: 'zh-CN' | 'en-US') =>
    t(`generator.pageActions.${action}`, { lng: locale });

  const selectedDatasource = datasources.find((item) => item.id === selectedDatasourceId);

  const loadDatasources = useCallback(async () => {
    const items = await listGeneratorDatasources();
    setDatasources(items);
    const nextSelectedDatasourceId = resolvePreferredDatasourceId(items, selectedDatasourceId);
    if (nextSelectedDatasourceId !== selectedDatasourceId) {
      setSelectedDatasourceId(nextSelectedDatasourceId);
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
      loadDatasources()
        .then((items) => {
          if (!active) {
            return;
          }
          const nextSelectedDatasourceId = resolvePreferredDatasourceId(
            items,
            selectedDatasourceId,
          );
          if (nextSelectedDatasourceId !== selectedDatasourceId) {
            setSelectedDatasourceId(nextSelectedDatasourceId);
          }
          return loadTables(nextSelectedDatasourceId);
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
      loadTables(selectedDatasourceId);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [form, loadTables, selectedDatasource?.name, selectedDatasourceId, sourceMode]);

  const getAllFormValues = () => form.getFields() as Partial<ModuleSchema>;

  const readMetadataValues = (): ResolvedModuleMetadata => {
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
      autoRecycle: Boolean(metadata?.autoRecycle),
    };
  };

  const buildSchema = (): ModuleSchema =>
    buildModuleSchema({
      values: getAllFormValues(),
      metadata: readMetadataValues(),
      fields,
      templateVersion,
      dependencyModulesText,
      relationContractsText,
      enableDataScope,
      includeDashboardWidget,
      listLayout,
      dataScopeMode,
      translationOverrides,
      actionLabel,
    });

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

  const previewSchema = resolvePreviewSchema(currentStep, buildSchema);
  const previewSchemaKey = schemaKey(previewSchema);
  const selectedActionTemplate = resolveSelectedActionTemplate(getAllFormValues());

  const handleBasicInfoSubmit = async () => {
    try {
      const metadata = readMetadataValues();
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
      const values = getAllFormValues();
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
        listLayout,
      };
      form.setFieldsValue(schema);
      setRegisterResult(null);
      setCurrentStep(1);
    } catch {
      message.error(t('generator.wizard.fillRequired'));
    }
  };

  const handleGenerate = async () => {
    const schema = buildSchema();
    const issues = validateGeneratorCompleteness(schema);
    if (issues.some((issue) => issue.level === 'error')) {
      message.error(t('generator.validation.blocked'));
      return;
    }
    try {
      const result = await requestPreviewGeneratedFiles({ schema });
      setGeneratedFiles(result.files);
      setGeneratedSchemaKey(JSON.stringify(schema));
      setRegisterResult(null);
      setCurrentStep(3);
    } catch (error) {
      if (isRequestError(error)) {
        message.error(t(error.messageKey || 'request.failed'));
        return;
      }
      message.error(t('request.failed'));
    }
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
    ).sort((a, b) => a.localeCompare(b));
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
    const importTranslationOverrides = async (file: File) => {
      const rows = parseCsvRows(await readFileText(file));
      setTranslationOverrides((current) => mergeTranslationOverrides(current, rows.slice(1)));
    };
    input.onchange = () => {
      const file = input.files?.[0];
      if (!file) {
        return;
      }
      importTranslationOverrides(file)
        .then(() => {
          message.success(t('generator.wizard.step3.translationPreview.importSuccess'));
        })
        .catch(() => {
          message.error(t('generator.wizard.step3.translationPreview.importError'));
        });
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

  const showDatasourceRequestError = useCallback(
    (error: unknown, fallbackKey: string) => {
      if (isRequestError(error)) {
        message.error(t(error.messageKey || fallbackKey));
        return;
      }
      message.error(t(fallbackKey));
    },
    [t],
  );

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
      showDatasourceRequestError(error, 'generator.datasource.saveError');
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
      showDatasourceRequestError(error, 'generator.datasource.testError');
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
      showDatasourceRequestError(error, 'generator.datasource.deleteError');
    }
  };

  const handleDownload = async () => {
    if (generatedFiles.length === 0) {
      return;
    }
    try {
      await downloadGeneratedSource({ schema: buildSchema() });
      message.success(t('generator.wizard.downloadSuccess'));
    } catch {
      message.error(t('generator.wizard.downloadError'));
    }
  };

  const oneClickEnabled = isOneClickRegistrationEnabled(
    currentStep,
    buildSchema().scope,
    generatedSchemaKey,
    previewSchemaKey,
    generatedFiles.length,
  );
  const canGenerateRegister = hasAccess(isAdmin, hasPerm('system:module:generate'));
  const canOpenModuleManager = hasAccess(isAdmin, hasPerm('system:module:list'));
  const summary = registerResult?.summary;
  const previewMenuTree = resolvePreviewMenuTree(previewSchema);
  const previewCompletenessIssues = resolvePreviewCompletenessIssues(previewSchema);
  const previewBlockingIssue = previewCompletenessIssues.some((issue) => issue.level === 'error');
  const previewGeneratedFiles = resolvePreviewGeneratedFiles(
    generatedSchemaKey,
    previewSchemaKey,
    generatedFiles,
  );
  const previewTranslationRows = buildTranslationPreviewRows(previewSchema);
  const translationPreviewPage = translationPreviewPagination.current;
  const translationPreviewPageSize = translationPreviewPagination.pageSize;
  const pagedPreviewTranslationRows = previewTranslationRows.slice(
    (translationPreviewPage - 1) * translationPreviewPageSize,
    translationPreviewPage * translationPreviewPageSize,
  );
  const activationStatusKey = moduleActivationStatusKey(registerResult?.module.status ?? 0);

  useEffect(() => {
    const totalPages = Math.max(
      1,
      Math.ceil(previewTranslationRows.length / translationPreviewPageSize),
    );
    if (translationPreviewPage > totalPages) {
      const timer = globalThis.setTimeout(() => {
        setTranslationPreviewPagination((current) => ({ ...current, current: totalPages }));
      }, 0);
      return () => globalThis.clearTimeout(timer);
    }
    return undefined;
  }, [previewTranslationRows.length, translationPreviewPage, translationPreviewPageSize]);

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
        overwrite,
      });
      setDynamicModuleDisabled(false);
      setRegisterResult(result);
      message.success(t('generator.wizard.register.success'));
    } catch (error) {
      switch (classifyRegistrationError(error, overwrite)) {
        case 'cancelled':
          return;
        case 'disabled':
          setDynamicModuleDisabled(true);
          return;
        case 'overwrite':
          showAppModalConfirm({
            title: t('generator.wizard.register.overwriteTitle'),
            content: t('generator.wizard.register.overwriteContent'),
            onOk: () => {
              submitGenerateAndRegister(true);
            },
          });
          return;
        case 'request':
          message.error(
            t(isRequestError(error) ? error.messageKey || 'request.failed' : 'request.failed'),
          );
          return;
        default:
          message.error(t('request.failed'));
      }
    } finally {
      setRegistering(false);
    }
  };

  const handleAuditActivation = async () => {
    if (!summary?.moduleKey) {
      return;
    }
    try {
      await ensureOperationVerified();
      setAuditingActivation(true);
      const result = await auditPendingActivations();
      const latest = await getModuleStatus(summary.moduleKey);
      setRegisterResult((current) =>
        current
          ? {
              ...current,
              module: {
                ...current.module,
                status: latest.status,
              },
            }
          : current,
      );
      message.success(
        t('generator.wizard.result.activationAuditSuccess', {
          activated: result.summary.activatedModules,
          pending: result.summary.pendingModules,
        }),
      );
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.wizard.result.activationAuditError'));
    } finally {
      setAuditingActivation(false);
    }
  };

  const wizardSteps = [
    {
      title: t('generator.wizard.step1.title'),
      description: t('generator.wizard.step1.desc'),
    },
    {
      title: t('generator.wizard.step2.title'),
      description: t('generator.wizard.step2.desc'),
    },
    {
      title: t('generator.wizard.step3.title'),
      description: t('generator.wizard.step3.desc'),
    },
    {
      title: t('generator.wizard.step4.title'),
      description: t('generator.wizard.step4.desc'),
    },
  ];
  return (
    <PageContainer className="generator-wizard-page">
      <Space
        direction="vertical"
        size={12}
        className="system-page-template generator-wizard-page__content"
      >
        <GovernanceSummaryBar
          eyebrow={t('generator.wizard.header.eyebrow')}
          title={t('generator.wizard.header.title')}
          description={t('generator.wizard.header.description')}
          metrics={[
            {
              key: 'steps',
              label: t('generator.wizard.header.steps'),
              value: `${currentStep + 1}/${wizardSteps.length}`,
            },
            {
              key: 'fields',
              label: t('generator.wizard.header.fields'),
              value: fields.length,
            },
            {
              key: 'files',
              label: t('generator.wizard.header.files'),
              value: generatedFiles.length,
            },
          ]}
        />
        <Card className="page-panel generator-wizard-card">
          {renderWhen(
            canOpenModuleManager,
            <div className="system-list__work-actions">
              <ListHeaderActions
                primary={
                  <Button size="small" onClick={() => navigate('/system/modules')}>
                    {t('generator.wizard.openRegistry')}
                  </Button>
                }
              />
            </div>,
          )}
          <div className="generator-wizard__steps generator-wizard__step-grid">
            {wizardSteps.map((step, index) => (
              <div key={step.title} className={buildWizardStepClass(index, currentStep)}>
                <span className="generator-wizard__step-index">{index + 1}</span>
                <span className="generator-wizard__step-title">{step.title}</span>
                <span className="generator-wizard__step-desc">{step.description}</span>
              </div>
            ))}
          </div>

          <div className="generator-wizard__content-divider" />

          {renderWhen(
            currentStep === 0,
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
                          <Button size="small" onClick={() => loadTables(selectedDatasourceId)}>
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
                        String(
                          (option as React.ReactElement<{ value?: string | number }> | null)?.props
                            ?.value || '',
                        )
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
                        previewGeneratorTable(tableName, selectedDatasourceId)
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
                          ((form.getFieldValue('scope') as ModuleScope | undefined) ||
                            'business') === 'business' &&
                          selectedTableRole !== 'relation' &&
                          includeDashboardWidget
                            ? 'enabled'
                            : 'disabled'
                        }
                        disabled={
                          ((form.getFieldValue('scope') as ModuleScope | undefined) ||
                            'business') !== 'business' || selectedTableRole === 'relation'
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
                <Card
                  size="small"
                  className="generator-wizard__list-layout-card"
                  title={t('generator.wizard.listLayout.title', 'List Layout')}
                >
                  <Row gutter={16}>
                    <Col xs={24} md={12}>
                      <FormItem label={t('generator.wizard.listLayout.governance', 'Governance')}>
                        <Select
                          value={listLayout.governance ? 'enabled' : 'disabled'}
                          onChange={(value) =>
                            setListLayout((current) => ({
                              ...current,
                              governance: value === 'enabled',
                            }))
                          }
                        >
                          <Select.Option value="enabled">{t('common.enabled')}</Select.Option>
                          <Select.Option value="disabled">{t('common.disabled')}</Select.Option>
                        </Select>
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12}>
                      <FormItem label={t('generator.wizard.listLayout.search', 'Search')}>
                        <Select
                          value={listLayout.search ? 'enabled' : 'disabled'}
                          onChange={(value) =>
                            setListLayout((current) => ({
                              ...current,
                              search: value === 'enabled',
                            }))
                          }
                        >
                          <Select.Option value="enabled">{t('common.enabled')}</Select.Option>
                          <Select.Option value="disabled">{t('common.disabled')}</Select.Option>
                        </Select>
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12}>
                      <FormItem
                        label={t('generator.wizard.listLayout.headerActions', 'Header Actions')}
                      >
                        <Select
                          value={listLayout.headerActions ? 'enabled' : 'disabled'}
                          onChange={(value) =>
                            setListLayout((current) => ({
                              ...current,
                              headerActions: value === 'enabled',
                            }))
                          }
                        >
                          <Select.Option value="enabled">{t('common.enabled')}</Select.Option>
                          <Select.Option value="disabled">{t('common.disabled')}</Select.Option>
                        </Select>
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12}>
                      <FormItem
                        label={t('generator.wizard.listLayout.batchActions', 'Batch Actions')}
                      >
                        <Select
                          value={listLayout.batchActions ? 'enabled' : 'disabled'}
                          onChange={(value) =>
                            setListLayout((current) => ({
                              ...current,
                              batchActions: value === 'enabled',
                            }))
                          }
                        >
                          <Select.Option value="enabled">{t('common.enabled')}</Select.Option>
                          <Select.Option value="disabled">{t('common.disabled')}</Select.Option>
                        </Select>
                      </FormItem>
                    </Col>
                  </Row>
                </Card>
                <Card
                  size="small"
                  className="generator-wizard__lifecycle-card"
                  title={t('generator.wizard.lifecycle.title')}
                >
                  <Space direction="vertical" className="generator-wizard__full" size={10}>
                    <Typography.Text type="secondary">
                      {t('generator.wizard.lifecycle.desc')}
                    </Typography.Text>
                    <FormItem
                      field="metadata.autoRecycle"
                      triggerPropName="checked"
                      initialValue={false}
                    >
                      <Checkbox>{t('generator.wizard.lifecycle.autoRecycle')}</Checkbox>
                    </FormItem>
                    <Alert
                      type={
                        form.getFieldValue('metadata.autoRecycle' as keyof ModuleSchema)
                          ? 'warning'
                          : 'info'
                      }
                      content={t(
                        form.getFieldValue('metadata.autoRecycle' as keyof ModuleSchema)
                          ? 'generator.wizard.lifecycle.autoRecycleHint'
                          : 'generator.wizard.lifecycle.standardHint',
                      )}
                    />
                  </Space>
                </Card>
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
                      <Typography.Text type="secondary">
                        {t('generator.wizard.relations.columns')}
                      </Typography.Text>
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
              <FormItem
                label={t('generator.wizard.boundedContext')}
                field="metadata.boundedContext"
              >
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
            </Form>,
          )}

          {renderWhen(
            currentStep === 1,
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
            </div>,
          )}

          {renderSchemaStep(currentStep, 2, previewSchema, (stepSchema) => (
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
                          PageActionKey[] | undefined) ?? stepSchema.pageActions
                      }
                      disabled={stepSchema.metadata?.tableRole === 'relation'}
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
                          count: stepSchema.model.fields.length,
                        })}
                      </Tag>
                      <Tag color="arcoblue">
                        {t('generator.wizard.step3.uniqueCount', {
                          count: stepSchema.model.fields.filter((field) => field.validation?.unique)
                            .length,
                        })}
                      </Tag>
                      <Tag color="purple">
                        {t('generator.wizard.step3.enumCount', {
                          count: stepSchema.model.fields.filter((field) => field.type === 'enum')
                            .length,
                        })}
                      </Tag>
                    </Space>
                    <div className="generator-wizard__enum-list">
                      {stepSchema.model.fields
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
                  {stepSchema.permissions.length === 0 ? (
                    <Typography.Text type="secondary">
                      {t('generator.wizard.step3.permissions.empty')}
                    </Typography.Text>
                  ) : (
                    stepSchema.permissions.map((permission) => (
                      <Tag key={`${permission.type}:${permission.key}`}>{permission.key}</Tag>
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
                    <MenuPreviewTree nodes={previewMenuTree} />
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
                        previewCompletenessIssues.map((issue) => {
                          const detailSuffix = issue.detail ? `: ${issue.detail}` : '';
                          return (
                            <Alert
                              key={`${issue.code}-${issue.detail || ''}`}
                              type={issue.level === 'error' ? 'error' : 'warning'}
                              content={`${t(issue.messageKey)}${detailSuffix}`}
                            />
                          );
                        })
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
                  <AppTable<TranslationPreviewRow>
                    className="system-list__table"
                    rowKey="key"
                    data={pagedPreviewTranslationRows}
                    pagination={buildStandardPagination(t, {
                      current: translationPreviewPagination.current,
                      pageSize: translationPreviewPagination.pageSize,
                      total: previewTranslationRows.length,
                      sizeOptions: [8, 16, 32, 64],
                      onChange: (page, pageSize) => {
                        setTranslationPreviewPagination({
                          current: page,
                          pageSize: pageSize || translationPreviewPagination.pageSize,
                        });
                      },
                    })}
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
                  <Tag color="green">{stepSchema.model.tableName}</Tag>
                  <Tag color={stepSchema.metadata?.tableRole === 'relation' ? 'orange' : 'purple'}>
                    {t(`generator.wizard.tableRole.${stepSchema.metadata?.tableRole || 'main'}`)}
                  </Tag>
                  <Tag color={stepSchema.enableDataScope ? 'green' : 'gray'}>
                    {t(
                      stepSchema.enableDataScope
                        ? 'generator.wizard.dataScope.enabledTag'
                        : 'generator.wizard.dataScope.disabledTag',
                    )}
                  </Tag>
                  <Tag color={stepSchema.includeDashboardWidget ? 'arcoblue' : 'gray'}>
                    {t(
                      stepSchema.includeDashboardWidget
                        ? 'generator.wizard.dashboardWidget.enabledTag'
                        : 'generator.wizard.dashboardWidget.disabledTag',
                    )}
                  </Tag>
                  <Tag color="blue">
                    {t('generator.wizard.step3.impact.dependencies', {
                      count: stepSchema.dependencies?.length || 0,
                    })}
                  </Tag>
                  <Tag color="orange">
                    {t('generator.wizard.step3.impact.relations', {
                      count: stepSchema.relations?.length || 0,
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
          ))}

          {renderSchemaStep(currentStep, 3, previewSchema, (stepSchema) => (
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
                      count: stepSchema.pageActions?.length || 0,
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
                      submitGenerateAndRegister();
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

              {renderWhen(
                dynamicModuleDisabled,
                <Alert
                  type="warning"
                  title={t('generator.wizard.register.disabledTitle')}
                  content={t('generator.wizard.register.disabledHint')}
                  className="generator-wizard__section"
                />,
              )}

              {renderRegistrationResult(registerResult, summary, (result, resultSummary) => (
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
                        {t('generator.wizard.result.moduleKey')}: {resultSummary.moduleKey}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.parentMenu')}:{' '}
                        {valueOr(
                          resultSummary.parentMenuPath,
                          t('generator.wizard.result.parentMenu.topLevel'),
                        )}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.routePath')}: {resultSummary.routePath}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.routeName')}: {resultSummary.routeName}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.componentKey')}: {resultSummary.componentKey}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.permissionPrefix')}:{' '}
                        {resultSummary.permissionPrefix}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.backendPath')}:{' '}
                        {resultSummary.backendModulePath}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.frontendPath')}:{' '}
                        {resultSummary.frontendModulePath}
                      </Typography.Text>
                      <Typography.Text>
                        {t('generator.wizard.result.schemaPath')}: {resultSummary.schemaPath}
                      </Typography.Text>
                      <Tag color={choose(result.module.status === 3, 'orange', 'green')}>
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
                          {resultSummary.contract.templateVersion}
                        </Tag>
                        <Tag color={choose(Boolean(result.module.autoRecycle), 'orange', 'gray')}>
                          {choose(
                            Boolean(result.module.autoRecycle),
                            t('generator.wizard.lifecycle.autoRecycleTag'),
                            t('generator.wizard.lifecycle.standardTag'),
                          )}
                        </Tag>
                        <Tag
                          color={choose(resultSummary.contract.dataScopeEnabled, 'green', 'gray')}
                        >
                          {t('generator.wizard.result.dataScope')}:{' '}
                          {resultSummary.contract.dataScopeMode}
                        </Tag>
                        <Tag color="blue">
                          {t('generator.wizard.result.dependencyCount', {
                            count: resultSummary.contract.dependencyCount,
                          })}
                        </Tag>
                        <Tag color="orange">
                          {t('generator.wizard.result.relationCount', {
                            count: resultSummary.contract.relationCount,
                          })}
                        </Tag>
                      </Space>
                      {renderWhen(
                        (resultSummary.contract.dependencies?.length || 0) > 0,
                        <Space direction="vertical" className="generator-wizard__full">
                          <Typography.Text type="secondary">
                            {t('generator.wizard.result.dependencies')}
                          </Typography.Text>
                          {resultSummary.contract.dependencies?.map((dependency) => (
                            <Typography.Text key={dependency.module} code>
                              {dependency.module}
                              {choose(Boolean(dependency.reason), ` · ${dependency.reason}`, '')}
                            </Typography.Text>
                          ))}
                        </Space>,
                      )}
                      {renderWhen(
                        (resultSummary.contract.relations?.length || 0) > 0,
                        <Space direction="vertical" className="generator-wizard__full">
                          <Typography.Text type="secondary">
                            {t('generator.wizard.result.relations')}
                          </Typography.Text>
                          {resultSummary.contract.relations?.map((relation) => (
                            <Typography.Text key={`${relation.name}-${relation.targetModule}`} code>
                              {relation.name} · {relation.type} · {relation.targetModule} ·{' '}
                              {relation.localField} → {relation.targetField}
                              {choose(
                                Boolean(relation.targetLabelField),
                                ` · label:${relation.targetLabelField}`,
                                '',
                              )}
                              {choose(
                                Boolean(relation.lookupApi),
                                ` · api:${relation.lookupApi}`,
                                '',
                              )}
                              {choose(
                                Boolean(relation.lookupValueField),
                                ` · value:${relation.lookupValueField}`,
                                '',
                              )}
                            </Typography.Text>
                          ))}
                        </Space>,
                      )}
                    </Space>
                  </Card>
                  <Card
                    title={t('generator.wizard.result.verifications')}
                    className="generator-wizard__section"
                  >
                    <Space direction="vertical" className="generator-wizard__full">
                      {resultSummary.verifications.map((item) => (
                        <Space
                          key={`${item.code}-${item.detail}`}
                          align="start"
                          className="generator-wizard__verification-row"
                        >
                          <Space>
                            <Tag color={verificationStatusColor(item.status)}>
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
                    {renderWhen(
                      canOpenModuleManager,
                      <Button
                        loading={auditingActivation}
                        disabled={result.module.status === 1}
                        onClick={() => {
                          handleAuditActivation();
                        }}
                      >
                        {t('generator.wizard.result.checkActivation')}
                      </Button>,
                    )}
                    {renderWhen(
                      canOpenModuleManager,
                      <Button onClick={() => navigate('/system/modules')}>
                        {t('generator.wizard.result.openModuleManager')}
                      </Button>,
                    )}
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
              ))}
            </div>
          ))}
        </Card>
      </Space>
      <AppModal
        title={t('generator.datasource.manageTitle')}
        visible={datasourceModalVisible}
        onCancel={() => setDatasourceModalVisible(false)}
        footer={null}
        size="xl"
      >
        <DatasourceManagerModal
          editingId={editingDatasourceId}
          saving={datasourceSaving}
          form={datasourceForm}
          items={datasources}
          onEditItem={handleEditDatasource}
          onDeleteItem={handleDeleteDatasource}
          onTestItem={handleTestDatasource}
          onSave={handleSaveDatasource}
          onReset={resetDatasourceForm}
        />
      </AppModal>
    </PageContainer>
  );
};

export default ModuleWizard;
