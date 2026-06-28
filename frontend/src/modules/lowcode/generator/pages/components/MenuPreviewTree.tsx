import React from 'react';
import { Space, Tag, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import type { GeneratorMenuPreviewNode } from '../../schema';

interface MenuPreviewTreeProps {
  nodes: GeneratorMenuPreviewNode[];
}

const MenuPreviewTree: React.FC<MenuPreviewTreeProps> = ({ nodes }) => {
  const { t } = useTranslation();

  if (nodes.length === 0) {
    return (
      <Typography.Text type="secondary">
        {t('generator.wizard.step3.menuPreview.empty')}
      </Typography.Text>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {nodes.map((node) => (
        <div key={node.key}>
          <Space wrap>
            <Tag
              color={node.type === 'M' ? 'arcoblue' : node.type === 'C' ? 'green' : 'orange'}
            >
              {node.type}
            </Tag>
            <Typography.Text>{node.titleKey}</Typography.Text>
            {node.path ? (
              <Typography.Text type="secondary">{node.path}</Typography.Text>
            ) : null}
          </Space>
          {node.children.length > 0 && (
            <div style={{ marginLeft: 24, marginTop: 8 }}>
              <MenuPreviewTree nodes={node.children} />
            </div>
          )}
        </div>
      ))}
    </div>
  );
};

export default MenuPreviewTree;
