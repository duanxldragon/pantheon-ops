import React from 'react';
import { Space, Typography } from '@arco-design/web-react';
import {
  formatDateTime,
  formatRelativeTime,
  isValidDateValue,
  type DateValue,
} from '../../core/format/dateTime';

interface DateTimeMetaProps {
  value?: DateValue;
  fallback?: React.ReactNode;
  showRelative?: boolean;
  direction?: 'vertical' | 'horizontal';
  className?: string;
}

const DateTimeMeta: React.FC<DateTimeMetaProps> = ({
  value,
  fallback = '-',
  showRelative = true,
  direction = 'vertical',
  className,
}) => {
  if (!isValidDateValue(value)) {
    const text =
      value === null || value === undefined || value === '' ? fallback : formatDateTime(value);
    return <>{text}</>;
  }

  if (!showRelative) {
    return <Typography.Text className={className}>{formatDateTime(value)}</Typography.Text>;
  }

  return (
    <Space direction={direction} size={direction === 'vertical' ? 2 : 6} className={className}>
      <Typography.Text>{formatDateTime(value)}</Typography.Text>
      <Typography.Text type="secondary">{formatRelativeTime(value)}</Typography.Text>
    </Space>
  );
};

export default DateTimeMeta;
