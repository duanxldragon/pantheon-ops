import React from 'react';
import { Space, Typography } from '@arco-design/web-react';

interface FormSectionProps {
  title?: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
}

const FormSection: React.FC<FormSectionProps> = ({ title, description, children }) => (
  <Space direction="vertical" size={12} className="form-section">
    {title ? (
      <Space direction="vertical" size={2} className="form-section__header">
        <Typography.Text className="form-section__title">{title}</Typography.Text>
        {description ? (
          <Typography.Text type="secondary" className="form-section__description">
            {description}
          </Typography.Text>
        ) : null}
      </Space>
    ) : null}
    {children}
  </Space>
);

export default FormSection;
