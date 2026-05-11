import type { TFunction } from 'i18next';
import { showAppModalError, showAppModalSuccess } from '../components';
export { downloadFile } from './file';
import { apiRequest } from './request';

export interface ImportErrorItem {
  row: number;
  field: string;
  message: string;
}

export interface ImportResult {
  applied: boolean;
  created: number;
  updated: number;
  failed: number;
  errors: ImportErrorItem[];
}

interface ImportResultOptions {
  errorFileName?: string;
  autoDownloadErrors?: boolean;
}

function translateImportMessage(message: string, t: TFunction) {
  const duplicateMatch = message.match(/^import\.duplicate\.row\.(\d+)$/);
  if (duplicateMatch?.[1]) {
    return t('import.duplicate.row', { row: Number(duplicateMatch[1]) });
  }
  const ownerConflictMatch = message.match(/^import\.conflict\.owner\.(.+)$/);
  if (ownerConflictMatch?.[1]) {
    return t('import.conflict.owner', { module: ownerConflictMatch[1] });
  }
  return t(message, { defaultValue: t('request.failed') });
}

export function uploadImportFile(url: string, file: File) {
  const formData = new FormData();
  formData.append('file', file);
  return apiRequest<ImportResult>({
    url,
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
}

function downloadTextFile(filename: string, content: string) {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' });
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  window.URL.revokeObjectURL(url);
}

export function downloadImportErrors(
  result: ImportResult,
  t: TFunction,
  filename = t('common.importErrorFileName'),
) {
  if (!result.errors.length) {
    return;
  }
  const escapeCSV = (value: string | number) => `"${String(value).replace(/"/g, '""')}"`;
  const rows = [
    [
      t('common.importErrorRow'),
      t('common.importErrorField'),
      t('common.importErrorMessageKey'),
      t('common.importErrorMessage'),
    ],
    ...result.errors.map((item) => [
      item.row,
      item.field,
      item.message,
      translateImportMessage(item.message, t),
    ]),
  ];
  const content = `\uFEFF${rows.map((row) => row.map(escapeCSV).join(',')).join('\n')}`;
  downloadTextFile(filename, content);
}

export function showImportResult(
  result: ImportResult,
  t: TFunction,
  options?: ImportResultOptions,
) {
  if (result.applied && result.failed === 0) {
    showAppModalSuccess({
      title: t('common.import'),
      content: t('common.importSummary', {
        created: result.created,
        updated: result.updated,
        failed: result.failed,
      }),
    });
    return;
  }

  const shouldDownloadErrors = Boolean(options?.autoDownloadErrors && result.errors.length > 0);
  if (shouldDownloadErrors) {
    downloadImportErrors(result, t, options?.errorFileName || t('common.importErrorFileName'));
  }

  const lines = result.errors
    .slice(0, 8)
    .map(
      (item) =>
        `${t('common.row', { count: item.row })} · ${item.field} · ${translateImportMessage(item.message, t)}`,
    );
  const tailLine = shouldDownloadErrors
    ? [
        t('common.importErrorFileDownloaded', {
          filename: options?.errorFileName || t('common.importErrorFileName'),
        }),
      ]
    : [];
  showAppModalError({
    title: t('common.importFailed'),
    content: [
      t('common.importSummary', {
        created: result.created,
        updated: result.updated,
        failed: result.failed,
      }),
      ...lines,
      ...tailLine,
    ].join('\n'),
  });
}
