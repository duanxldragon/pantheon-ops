import { createRequire } from 'node:module';
import path from 'node:path';

const frontend = process.cwd();
const require = createRequire(path.join(frontend, 'package.json'));
const ts = require('typescript');
const configPath = path.join(frontend, 'tsconfig.app.json');
const config = ts.readConfigFile(configPath, ts.sys.readFile);
if (config.error) throw new Error(ts.flattenDiagnosticMessageText(config.error.messageText, '\n'));
const parsed = ts.parseJsonConfigFileContent(config.config, ts.sys, frontend);
const owned = [
  'src/modules/business/deploy/task/useDeployTaskSnapshot.ts',
  'tests/unit/business/operational/deploy-read-revocation.test.tsx',
  'tests/unit/business/operational/deploy-detail-read.test.tsx',
].map((file) => path.join(frontend, file));
const program = ts.createProgram({ rootNames: [...new Set([...parsed.fileNames, ...owned])], options: parsed.options });
const diagnostics = [...parsed.errors, ...ts.getPreEmitDiagnostics(program)];
const selected = diagnostics.filter((diagnostic) => !diagnostic.file || owned.includes(path.normalize(diagnostic.file.fileName)));
const format = (diagnostic) => ({
  file: diagnostic.file?.fileName,
  line: diagnostic.file && diagnostic.start !== undefined ? diagnostic.file.getLineAndCharacterOfPosition(diagnostic.start).line + 1 : undefined,
  category: ts.DiagnosticCategory[diagnostic.category],
  code: diagnostic.code,
  message: ts.flattenDiagnosticMessageText(diagnostic.messageText, '\n'),
});
console.log(JSON.stringify({ compiler: ts.version, files: owned, ownedDiagnostics: selected.map(format), totalErrors: diagnostics.filter((diagnostic) => diagnostic.category === ts.DiagnosticCategory.Error).length, outsideOwnedDiagnostics: diagnostics.filter((diagnostic) => !selected.includes(diagnostic)).map(format) }, null, 2));
process.exitCode = diagnostics.some((diagnostic) => diagnostic.category === ts.DiagnosticCategory.Error) ? 1 : 0;
