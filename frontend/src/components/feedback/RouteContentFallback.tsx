import React from 'react';
import { Card, Skeleton, Space } from '@arco-design/web-react';
import PageContainer from '../patterns/layout/PageContainer';

const RouteContentFallback: React.FC = () => (
  <PageContainer className="route-content-fallback" data-testid="route-content-fallback">
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card className="page-panel">
        <Skeleton
          animation
          text={{
            rows: 2,
            width: ['28%', '48%'],
          }}
        />
      </Card>
      <Card className="page-panel">
        <Skeleton
          animation
          text={{
            rows: 8,
            width: ['100%', '100%', '92%', '100%', '88%', '100%', '96%', '72%'],
          }}
        />
      </Card>
    </Space>
  </PageContainer>
);

export default RouteContentFallback;
