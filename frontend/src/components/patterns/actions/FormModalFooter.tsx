import React from 'react';
import SubmitBar from './SubmitBar';

interface FormModalFooterProps {
  onCancel: () => void;
  onSubmit: () => void;
  loading?: boolean;
  submitText: React.ReactNode;
}

const FormModalFooter: React.FC<FormModalFooterProps> = ({
  onCancel,
  onSubmit,
  loading,
  submitText,
}) => (
  <SubmitBar
    onCancel={onCancel}
    onSubmit={onSubmit}
    loading={loading}
    submitText={submitText}
  />
);

export default FormModalFooter;
