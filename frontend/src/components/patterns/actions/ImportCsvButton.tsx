import React, { useRef } from 'react';
import { Button } from '@arco-design/web-react';
import { IconUpload } from '@arco-design/web-react/icon';

interface ImportCsvButtonProps {
  disabled?: boolean;
  loading?: boolean;
  children: React.ReactNode;
  onSelect: (file: File) => void;
}

const ImportCsvButton: React.FC<ImportCsvButtonProps> = ({
  disabled,
  loading,
  children,
  onSelect,
}) => {
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <>
      <Button
        icon={<IconUpload />}
        disabled={disabled}
        loading={loading}
        onClick={() => inputRef.current?.click()}
      >
        {children}
      </Button>
      <input
        ref={inputRef}
        type="file"
        accept=".csv,text/csv"
        style={{ display: 'none' }}
        onChange={(event) => {
          const file = event.target.files?.[0];
          event.target.value = '';
          if (file) {
            onSelect(file);
          }
        }}
      />
    </>
  );
};

export default ImportCsvButton;
