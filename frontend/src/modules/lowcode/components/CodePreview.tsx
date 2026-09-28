/**
 * 模块生成器 - 代码预览组件
 *
 * 显示生成的代码文件列表,支持文件切换和代码高亮
 */

import React, { useState } from 'react';
import { Tree, Typography, Space, Button, Tag } from '@arco-design/web-react';
import { IconCode, IconDownload } from '@arco-design/web-react/icon';
import type { TreeDataType } from '@arco-design/web-react/es/Tree/interface';
import { useTranslation } from 'react-i18next';
import { AppDrawer } from '../../../components';
import './CodePreview.css';

import type { GeneratedFile } from '../generator/exporter';

interface CodePreviewProps {
  visible: boolean;
  files: GeneratedFile[];
  onClose: () => void;
  onDownload?: () => void;
}

interface CodeTreeNode extends TreeDataType {
  key: string;
  title: string;
  isLeaf?: boolean;
  file?: GeneratedFile;
  children?: CodeTreeNode[];
}

export const CodePreview: React.FC<CodePreviewProps> = ({
  visible,
  files,
  onClose,
  onDownload,
}) => {
  const { t } = useTranslation();
  const [selectedFile, setSelectedFile] = useState<GeneratedFile | null>(null);

  // 构建文件树
  const buildFileTree = () => {
    const tree: CodeTreeNode[] = [];
    const pathMap = new Map<string, CodeTreeNode>();

    files.forEach((file) => {
      const parts = file.path.split('/');
      let currentPath = '';
      let parentNode: CodeTreeNode[] = tree;

      parts.forEach((part, index) => {
        currentPath = currentPath ? `${currentPath}/${part}` : part;

        if (index === parts.length - 1) {
          // 文件节点
          parentNode.push({
            key: file.path,
            title: part,
            isLeaf: true,
            file,
          });
        } else {
          // 目录节点
          let dir = pathMap.get(currentPath);
          if (!dir) {
            dir = {
              key: currentPath,
              title: part,
              children: [],
            };
            pathMap.set(currentPath, dir);
            parentNode.push(dir);
          }
          parentNode = dir.children ?? [];
        }
      });
    });

    return tree;
  };

  // 选择文件
  const handleSelect = (selectedKeys: React.Key[]) => {
    const key = selectedKeys[0] as string;
    const file = files.find((f) => f.path === key);
    if (file) {
      setSelectedFile(file);
    }
  };

  // 获取语言标签
  const getLanguageTag = (language: string) => {
    const colorMap: Record<string, string> = {
      go: 'blue',
      typescript: 'green',
      tsx: 'purple',
    };
    return colorMap[language] || 'gray';
  };

  return (
    <AppDrawer
      title={t('generator.codePreview.title')}
      visible={visible}
      onCancel={onClose}
      size="xl"
      footer={
        selectedFile &&
        onDownload && (
          <Space>
            <Button type="primary" onClick={onDownload}>
              <IconDownload /> {t('generator.codePreview.download')}
            </Button>
          </Space>
        )
      }
    >
      <div className="generator-code-preview">
        {/* 左侧: 文件树 */}
        <div className="generator-code-preview__tree">
          <Typography.Title heading={6} className="generator-code-preview__section-title">
            {t('generator.codePreview.files')}
          </Typography.Title>
          <Tree treeData={buildFileTree()} onSelect={handleSelect} showLine />
        </div>

        {/* 右侧: 代码显示 */}
        <div className="generator-code-preview__content">
          {selectedFile ? (
            <div>
              <Space className="generator-code-preview__file-header">
                <Typography.Title heading={6} className="generator-code-preview__file-title">
                  {selectedFile.path.split('/').pop()}
                </Typography.Title>
                <Tag color={getLanguageTag(selectedFile.language)}>
                  {selectedFile.language.toUpperCase()}
                </Tag>
              </Space>

              <Typography.Text type="secondary" className="generator-code-preview__path">
                {selectedFile.path}
              </Typography.Text>

              <pre className="generator-code-preview__pre">
                <code>{selectedFile.content}</code>
              </pre>
            </div>
          ) : (
            <div className="generator-code-preview__empty">
              <IconCode className="generator-code-preview__empty-icon" />
              <Typography.Text>{t('generator.codePreview.selectFile')}</Typography.Text>
            </div>
          )}
        </div>
      </div>
    </AppDrawer>
  );
};
