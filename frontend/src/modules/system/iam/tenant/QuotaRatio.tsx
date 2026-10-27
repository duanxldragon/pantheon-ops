import React from 'react';
import { Typography } from '@arco-design/web-react';
import { formatQuotaLimit, isUnlimitedQuota } from './quota';
import './QuotaRatio.css';

/**
 * Usage against a plan ceiling, the one shape both tenant-ops pages report.
 *
 * Exceeding a ceiling is highlighted because the platform console reports
 * occupancy without enforcing it: quota is a measurement, not a write gate
 * (TENANT_CONTRACT_V1 §2.4), so an over-limit tenant is a real state an
 * operator has to see.
 */
interface QuotaRatioProps {
  usage: number;
  limit: number;
  unlimitedText: string;
}

/**
 * Which highlight the ratio gets, in one place so "over" and "exactly at the
 * ceiling" cannot drift apart between the two pages that share this cell.
 */
function quotaTone(usage: number, limit: number): string {
  if (isUnlimitedQuota(limit)) {
    return '';
  }
  if (usage > limit) {
    return 'tenant-quota-is-over';
  }
  if (usage >= limit) {
    return 'tenant-quota-is-full';
  }
  return '';
}

const QuotaRatio: React.FC<QuotaRatioProps> = ({ usage, limit, unlimitedText }) => {
  const tone = quotaTone(usage, limit);

  return (
    <Typography.Text className={tone ? `tenant-quota ${tone}` : 'tenant-quota'}>
      {`${usage} / ${formatQuotaLimit(limit, unlimitedText)}`}
    </Typography.Text>
  );
};

export default QuotaRatio;
