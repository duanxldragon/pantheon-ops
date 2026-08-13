import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

import { prepareTranspiledWorkspace } from './transpile-typescript-files.mjs';

const files = [
  'src/api/requestErrorUtils.ts',
  'tests/api/request-error-utils.test.ts',
];

const { tempDir } = prepareTranspiledWorkspace('request-error-utils-test', files);
await import(pathToFileURL(join(tempDir, 'tests', 'api', 'request-error-utils.test.js')));
