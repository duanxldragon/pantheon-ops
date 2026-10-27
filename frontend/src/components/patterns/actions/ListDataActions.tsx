import React from 'react';
import { Button } from '@arco-design/web-react';
import { IconDownload, IconRefresh } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import ImportCsvButton from './ImportCsvButton';

export interface ListDataActionsProps {
  canExport?: boolean;
  canImport?: boolean;
  canDownloadTemplate?: boolean;
  onExport?: () => void;
  onDownloadTemplate?: () => void;
  onImport?: (file: File) => void;
  onRefresh?: () => void;
  templateIcon?: boolean;
  size?: 'mini' | 'small' | 'default' | 'large';
}

const ListDataActions: React.FC<ListDataActionsProps> = ({
  canExport = true,
  canImport = true,
  canDownloadTemplate = true,
  onExport,
  onDownloadTemplate,
  onImport,
  onRefresh,
  templateIcon = false,
  size,
}) => {
  const { t } = useTranslation();
  return (
    <>
      {onRefresh ? (
        <Button size={size} icon={<IconRefresh />} onClick={onRefresh}>
          {t('common.refresh')}
        </Button>
      ) : null}
      {onExport ? (
        <Button size={size} icon={<IconDownload />} onClick={onExport} disabled={!canExport}>
          {t('common.export')}
        </Button>
      ) : null}
      {onDownloadTemplate ? (
        <Button
          size={size}
          icon={templateIcon ? <IconDownload /> : undefined}
          onClick={onDownloadTemplate}
          disabled={!canDownloadTemplate}
        >
          {t('common.downloadTemplate')}
        </Button>
      ) : null}
      {onImport ? (
        <ImportCsvButton disabled={!canImport} onSelect={onImport}>
          {t('common.import')}
        </ImportCsvButton>
      ) : null}
    </>
  );
};

export default ListDataActions;
