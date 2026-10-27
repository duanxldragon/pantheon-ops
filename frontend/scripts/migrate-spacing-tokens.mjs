#!/usr/bin/env node
/**
 * 批量迁移硬编码间距值到 token
 * 只处理 modules/ 下的 CSS 文件
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendRoot = path.resolve(__dirname, '..');
const modulesRoot = path.join(frontendRoot, 'src', 'modules');

// 间距值映射表
const SPACING_MAP = {
  '2px': 'var(--space-2xs)',
  '3px': 'var(--space-3xs)',
  '4px': 'var(--space-xs)',
  '6px': 'var(--space-xs-plus)',
  '8px': 'var(--space-sm)',
  '9px': 'var(--space-sm-alt)',
  '10px': 'var(--space-sm-plus)',
  '11px': 'var(--space-sm-large)',
  '12px': 'var(--space-md)',
  '13px': 'var(--space-md-alt)',
  '14px': 'var(--space-md-plus)',
  '16px': 'var(--space-lg)',
  '18px': 'var(--space-lg-alt)',
  '20px': 'var(--space-lg-plus)',
  '22px': 'var(--space-lg-large)',
  '24px': 'var(--space-xl)',
  '26px': 'var(--space-xl-alt)',
  '28px': 'var(--space-xl-plus)',
  '32px': 'var(--space-2xl)',
  '34px': 'var(--space-2xl-alt)',
  '36px': 'var(--space-2xl-plus)',
  '44px': 'var(--space-3xl)', // 最接近 48px
  '48px': 'var(--space-3xl)',
  '56px': 'var(--space-3xl-alt)',
  '64px': 'var(--space-3xl-plus)',
  '84px': 'var(--space-4xl)',
  '88px': 'var(--space-4xl-plus)',
};

function walk(dir) {
  const files = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(fullPath));
    } else if (entry.name.endsWith('.css')) {
      files.push(fullPath);
    }
  }
  return files;
}

function migratePaddingMargin(line) {
  // 匹配 padding/margin 属性
  const match = line.match(/^(\s*)(padding|margin)(-(?:top|right|bottom|left))?:\s*([^;]+);/);
  if (!match) return line;

  const [, indent, prop, suffix, value] = match;

  // 如果已经使用 token，跳过
  if (value.includes('var(--')) return line;

  // 如果是 0，跳过
  if (value.trim() === '0') return line;

  // 替换所有像素值
  let newValue = value;
  for (const [px, token] of Object.entries(SPACING_MAP)) {
    // 使用 \b 确保完整匹配（避免 12px 被误匹配为 2px）
    newValue = newValue.replace(new RegExp(`\\b${px}\\b`, 'g'), token);
  }

  // 如果没有变化，返回原行
  if (newValue === value) return line;

  return `${indent}${prop}${suffix || ''}: ${newValue};`;
}

let totalFiles = 0;
let modifiedFiles = 0;
let totalReplacements = 0;

const files = walk(modulesRoot);

for (const file of files) {
  totalFiles++;
  const content = fs.readFileSync(file, 'utf8');
  const lines = content.split(/\r?\n/);

  let modified = false;
  const newLines = lines.map((line) => {
    const newLine = migratePaddingMargin(line);
    if (newLine !== line) {
      modified = true;
      totalReplacements++;
    }
    return newLine;
  });

  if (modified) {
    fs.writeFileSync(file, newLines.join('\n'), 'utf8');
    modifiedFiles++;
    console.log(`✓ ${path.relative(frontendRoot, file)}`);
  }
}

console.log(`\n完成！`);
console.log(`  扫描文件: ${totalFiles}`);
console.log(`  修改文件: ${modifiedFiles}`);
console.log(`  替换次数: ${totalReplacements}`);
