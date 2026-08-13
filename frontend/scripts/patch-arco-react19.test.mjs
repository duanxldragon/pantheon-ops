import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const patchScript = path.join(frontendRoot, 'scripts', 'patch-arco-react19.mjs');
const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-arco-react19-'));

after(() => {
  fs.rmSync(tmpRoot, { recursive: true, force: true });
});

function writeFixture(relativePath, content) {
  const filePath = path.join(tmpRoot, relativePath);
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content);
}

function runPatchScript() {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [patchScript, '--root', tmpRoot], {
      cwd: frontendRoot,
      env: process.env,
      shell: false,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    let stdout = '';
    let stderr = '';
    child.stdout.setEncoding('utf8');
    child.stderr.setEncoding('utf8');
    child.stdout.on('data', (chunk) => {
      stdout += chunk;
    });
    child.stderr.on('data', (chunk) => {
      stderr += chunk;
    });
    child.once('error', reject);
    child.once('exit', (code, signal) => {
      resolve({ code: code ?? 0, signal: signal ?? null, stdout, stderr });
    });
  });
}

test('patch-arco-react19 rewrites direct ref reads in the installed Arco package', async () => {
  writeFixture(
    'node_modules/@arco-design/web-react/es/_util/react-dom.js',
    [
      "import { isObject, isFunction, isReact18 } from './is';",
      'export var callbackOriginRef = function (children, node) {',
      '    if (children && children.ref) {',
      '        if (isFunction(children.ref)) {',
      '            children === null || children === void 0 ? void 0 : children.ref(node);',
      '        }',
      "        if ('current' in children.ref) {",
      '            children.ref.current = node;',
      '        }',
      '    }',
      '};',
    ].join('\n'),
  );
  writeFixture(
    'node_modules/@arco-design/web-react/lib/_util/react-dom.js',
    [
      '"use strict";',
      'var is_1 = require("./is");',
      'var callbackOriginRef = function (children, node) {',
      '    if (children && children.ref) {',
      '        if ((0, is_1.isFunction)(children.ref)) {',
      '            children === null || children === void 0 ? void 0 : children.ref(node);',
      '        }',
      "        if ('current' in children.ref) {",
      '            children.ref.current = node;',
      '        }',
      '    }',
      '};',
    ].join('\n'),
  );
  writeFixture(
    'node_modules/@arco-design/web-react/es/Trigger/index.js',
    [
      "import { isFunction, isObject, isArray, supportRef } from '../_util/is';",
      'React.createElement(popupChildren.type, __assign({ ref: popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
    ].join('\n'),
  );
  writeFixture(
    'node_modules/@arco-design/web-react/lib/Trigger/index.js',
    [
      'var is_1 = require("../_util/is");',
      'react_1.default.createElement(popupChildren.type, __assign({ ref: popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
    ].join('\n'),
  );
  writeFixture(
    'node_modules/@arco-design/web-react/es/AutoComplete/index.js',
    [
      "import useMergeProps from '../_util/hooks/useMergeProps';",
      "var IMPOSSIBLE_VALUE = \"Autocomplete_\" + Math.random();",
      '    var originRef = usedTriggerElement.ref;',
    ].join('\n'),
  );
  writeFixture(
    'node_modules/@arco-design/web-react/lib/AutoComplete/index.js',
    [
      'var useMergeProps_1 = __importDefault(require("../_util/hooks/useMergeProps"));',
      'var IMPOSSIBLE_VALUE = "Autocomplete_" + Math.random();',
      '    var originRef = usedTriggerElement.ref;',
    ].join('\n'),
  );
  fs.mkdirSync(path.join(tmpRoot, 'node_modules', '.vite', 'deps'), { recursive: true });
  fs.writeFileSync(path.join(tmpRoot, 'node_modules', '.vite', 'deps', 'placeholder.txt'), 'cache');
  fs.mkdirSync(path.join(tmpRoot, 'node_modules', '.vite-temp'), { recursive: true });
  fs.writeFileSync(path.join(tmpRoot, 'node_modules', '.vite-temp', 'placeholder.txt'), 'cache');

  const first = await runPatchScript();
  assert.equal(first.code, 0, `${first.stderr}\n${first.stdout}`);

  const expectedFiles = [
    [
      'node_modules/@arco-design/web-react/es/_util/react-dom.js',
      [
        "import { isObject, isFunction, isReact18, isReact19 } from './is';",
        '    var originRef = isReact19 ? children === null || children === void 0 ? void 0 : children.props.ref : children && children.ref;',
      ],
    ],
    [
      'node_modules/@arco-design/web-react/lib/_util/react-dom.js',
      [
        '    var originRef = is_1.isReact19 ? children === null || children === void 0 ? void 0 : children.props.ref : children && children.ref;',
      ],
    ],
    [
      'node_modules/@arco-design/web-react/es/Trigger/index.js',
      [
        "import { isFunction, isObject, isArray, supportRef, isReact19 } from '../_util/is';",
        'React.createElement(popupChildren.type, __assign({ ref: isReact19 ? popupChildren === null || popupChildren === void 0 ? void 0 : popupChildren.props.ref : popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
      ],
    ],
    [
      'node_modules/@arco-design/web-react/lib/Trigger/index.js',
      [
        'react_1.default.createElement(popupChildren.type, __assign({ ref: is_1.isReact19 ? popupChildren === null || popupChildren === void 0 ? void 0 : popupChildren.props.ref : popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
      ],
    ],
    [
      'node_modules/@arco-design/web-react/es/AutoComplete/index.js',
      [
        "import useMergeProps from '../_util/hooks/useMergeProps';\nimport { isReact19 } from '../_util/is';",
        '    var originRef = isReact19 ? usedTriggerElement === null || usedTriggerElement === void 0 ? void 0 : usedTriggerElement.props.ref : usedTriggerElement.ref;',
      ],
    ],
    [
      'node_modules/@arco-design/web-react/lib/AutoComplete/index.js',
      [
        'var useMergeProps_1 = __importDefault(require("../_util/hooks/useMergeProps"));\nvar is_1 = require("../_util/is");',
        '    var originRef = is_1.isReact19 ? usedTriggerElement === null || usedTriggerElement === void 0 ? void 0 : usedTriggerElement.props.ref : usedTriggerElement.ref;',
      ],
    ],
  ];

  for (const [relativePath, snippets] of expectedFiles) {
    const contents = fs.readFileSync(path.join(tmpRoot, relativePath), 'utf8');
    for (const snippet of snippets) {
      assert.match(contents, new RegExp(snippet.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
    }
  }

  assert.equal(fs.existsSync(path.join(tmpRoot, 'node_modules', '.vite')), false);
  assert.equal(fs.existsSync(path.join(tmpRoot, 'node_modules', '.vite-temp')), false);

  const second = await runPatchScript();
  assert.equal(second.code, 0, `${second.stderr}\n${second.stdout}`);
  for (const [relativePath] of expectedFiles) {
    const contents = fs.readFileSync(path.join(tmpRoot, relativePath), 'utf8');
    assert.ok(contents.length > 0);
  }
});
