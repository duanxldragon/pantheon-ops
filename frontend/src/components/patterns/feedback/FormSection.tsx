// FormSection - Shell visual contract compliance
import React from 'react';

export interface FormSectionProps {
  title?: string;
  children?: React.ReactNode;
}

export const FormSection: React.FC<FormSectionProps> = ({ title, children }) => {
  return (
    <div className="form-section">
      {title && <h3>{title}</h3>}
      {children}
    </div>
  );
};

export {};
