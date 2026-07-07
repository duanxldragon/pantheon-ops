/**
 * 模块生成器 - 代码导出器
 *
 * 整合后端和前端生成器,提供完整的代码预览和下载功能
 */

import type { ModuleSchema } from './schema';
import { getLeafModuleName, inferModelName } from './schema';
import { BackendGenerator } from './backendGenerator';
import { FrontendGenerator } from './frontendGenerator';

/**
 * 生成的文件
 */
export interface GeneratedFile {
  path: string; // 文件路径 (如: backend/modules/business/order/order_model.go)
  content: string; // 文件内容
  language: string; // 语言 (go, typescript, tsx)
}

/**
 * 模块导出器
 */
export class ModuleExporter {
  private schema: ModuleSchema;
  private backendGen: BackendGenerator;
  private frontendGen: FrontendGenerator;

  constructor(schema: ModuleSchema) {
    this.schema = schema;
    this.backendGen = new BackendGenerator(schema);
    this.frontendGen = new FrontendGenerator(schema);
  }

  /**
   * 生成所有代码文件
   */
  generateAll(): GeneratedFile[] {
    const files: GeneratedFile[] = [];
    const { scope, name } = this.schema;
    const leafName = getLeafModuleName(name);

    // 后端文件
    files.push({
      path: `backend/modules/${scope}/${name}/${leafName}_model.go`,
      content: this.backendGen.generateModel(),
      language: 'go',
    });

    files.push({
      path: `backend/modules/${scope}/${name}/${leafName}_dto.go`,
      content: this.backendGen.generateDTO(),
      language: 'go',
    });

    files.push({
      path: `backend/modules/${scope}/${name}/${leafName}_service.go`,
      content: this.backendGen.generateService(),
      language: 'go',
    });

    files.push({
      path: `backend/modules/${scope}/${name}/${leafName}_handler.go`,
      content: this.backendGen.generateHandler(),
      language: 'go',
    });

    files.push({
      path: `backend/modules/${scope}/${name}/module.go`,
      content: this.backendGen.generateModule(),
      language: 'go',
    });

    // 前端文件
    files.push({
      path: `frontend/src/modules/${scope}/${name}/index.ts`,
      content: this.frontendGen.generateModuleIndex(),
      language: 'typescript',
    });

    files.push({
      path: `frontend/src/modules/${scope}/${name}/api.ts`,
      content: this.frontendGen.generateAPI(),
      language: 'typescript',
    });

    files.push({
      path: `frontend/src/modules/${scope}/${name}/${this.inferModelName()}List.tsx`,
      content: this.frontendGen.generateListPage(),
      language: 'tsx',
    });

    files.push({
      path: `frontend/src/modules/${scope}/${name}/${this.inferModelName()}Form.tsx`,
      content: this.frontendGen.generateFormComponent(),
      language: 'tsx',
    });

    files.push({
      path: `frontend/src/modules/${scope}/${name}/${this.inferModelName()}Detail.tsx`,
      content: this.frontendGen.generateDetailPage(),
      language: 'tsx',
    });

    return files;
  }

  /**
   * 生成代码摘要(用于预览)
   */
  generateSummary(): {
    backendFiles: string[];
    frontendFiles: string[];
    totalLines: number;
  } {
    const files = this.generateAll();
    const backendFiles = files.filter((f) => f.language === 'go').map((f) => f.path);
    const frontendFiles = files.filter((f) => f.language !== 'go').map((f) => f.path);
    const totalLines = files.reduce((sum, f) => sum + f.content.split('\n').length, 0);

    return {
      backendFiles,
      frontendFiles,
      totalLines,
    };
  }

  /**
   * 导出为 ZIP (需要引入 JSZip 库)
   *
   * 注意: 需要先安装 jszip: npm install jszip
   */
  async exportAsZip(): Promise<Blob> {
    // 动态导入 JSZip (如果未安装会报错)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const JSZipModule = await import('jszip' as any);
    const JSZip = JSZipModule.default;
    const zip = new JSZip();

    const files = this.generateAll();
    files.forEach((file) => {
      zip.file(file.path, file.content);
    });

    return zip.generateAsync({ type: 'blob' });
  }

  /**
   * 推断 Model 名称
   */
  private inferModelName(): string {
    return inferModelName(this.schema);
  }
}
