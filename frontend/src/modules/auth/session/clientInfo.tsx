import { Space, Typography } from '@arco-design/web-react';

type ClientInfoLike = {
  browser?: string;
  os?: string;
  device?: string;
  userAgent?: string;
};

export function formatClientSummary(info?: ClientInfoLike | null) {
  if (!info) {
    return '-';
  }

  const segments = [info.device, info.browser, info.os].filter(
    (item) => item && item !== 'Unknown',
  );
  if (segments.length > 0) {
    return segments.join(' · ');
  }

  return info.userAgent || '-';
}

export function renderClientInfo(info?: ClientInfoLike | null) {
  if (!info) {
    return '-';
  }

  return (
    <Space direction="vertical" size={2}>
      <span>{formatClientSummary(info)}</span>
      {info.userAgent ? (
        <Typography.Text
          type="secondary"
          ellipsis={{ showTooltip: true }}
          style={{ maxWidth: 360 }}
        >
          {info.userAgent}
        </Typography.Text>
      ) : null}
    </Space>
  );
}
