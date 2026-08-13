import axios from 'axios';
import { message } from '../components/feedback/message';
import i18n from 'i18next';

interface DownloadFileOptions {
  url: string;
  method?: 'get' | 'post';
  data?: unknown;
  params?: Record<string, unknown>;
  filename?: string;
}

function parseFilename(contentDisposition?: string, fallbackName?: string) {
  if (!contentDisposition) {
    return fallbackName || 'download.csv';
  }
  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(contentDisposition);
  if (utf8Match?.[1]) {
    return decodeURIComponent(utf8Match[1]);
  }
  const plainMatch = /filename="?([^"]+)"?/i.exec(contentDisposition);
  if (plainMatch?.[1]) {
    return plainMatch[1];
  }
  return fallbackName || 'download.csv';
}

const I18N_KEY_PATTERN = /^[a-z0-9_]+(?:\.[a-z0-9_]+)+$/i;

function saveBlob(blob: Blob, filename: string) {
  const url = globalThis.URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  globalThis.URL.revokeObjectURL(url);
}

// 与后端 impexp.sanitizeCSVCell 对齐：以 = + - @ 或制表符/回车开头的非数值
// 单元格在 Excel 中会被当作公式执行（日志里的用户名等字段可被攻击者写入），
// 统一前置单引号中和；合法数值（-5、+3.14）保持原样。
function neutralizeCsvFormula(value: string) {
  if (!value) {
    return value;
  }
  if (!/^[=+\-@\t\r]/.test(value)) {
    return value;
  }
  if (!Number.isNaN(Number(value))) {
    return value;
  }
  return `'${value}`;
}

export function downloadCsvFile(filename: string, headers: string[], rows: string[][]) {
  const escaped = (value: string) => {
    const normalized = neutralizeCsvFormula(String(value ?? ''));
    if (/[",\r\n]/.test(normalized)) {
      return `"${normalized.replaceAll('"', '""')}"`;
    }
    return normalized;
  };
  const csv = [
    headers.map(escaped).join(','),
    ...rows.map((row) => row.map(escaped).join(',')),
  ].join('\r\n');
  const blob = new Blob(['\uFEFF', csv], { type: 'text/csv;charset=utf-8' });
  saveBlob(blob, filename || 'export.csv');
}

export async function downloadFile(options: DownloadFileOptions) {
  const response = await axios.request<Blob>({
    baseURL: '/api/v1',
    url: options.url,
    method: options.method || 'get',
    data: options.data,
    params: options.params,
    responseType: 'blob',
    timeout: 30000,
    withCredentials: true,
    headers: {
      'Accept-Language': localStorage.getItem('pantheon_lang') || 'zh-CN',
    },
    validateStatus: () => true,
  });

  const contentTypeHeader = response.headers['content-type'];
  let contentType = '';
  if (Array.isArray(contentTypeHeader)) {
    contentType = contentTypeHeader.join(',');
  } else if (typeof contentTypeHeader === 'string') {
    contentType = contentTypeHeader;
  }
  if (response.status >= 400 || contentType.includes('application/json')) {
    try {
      const text = await response.data.text();
      const payload = JSON.parse(text);
      const messageKey = payload?.message || 'request.failed';
      const resolvedKey = I18N_KEY_PATTERN.test(String(messageKey)) ? messageKey : 'request.failed';
      message.error(i18n.t(resolvedKey, { defaultValue: resolvedKey }));
      throw new Error(messageKey);
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('request.failed', { cause: error });
    }
  }

  const filename = parseFilename(
    String(response.headers['content-disposition'] || ''),
    options.filename,
  );
  saveBlob(response.data, filename);
}
