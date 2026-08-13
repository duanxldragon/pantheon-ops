import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

function readArg(flag, fallback = '') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return process.argv[index + 1];
}

function resolveFrontendRoot() {
  const cliRoot = readArg('--root', '');
  const envRoot = process.env.PANTHEON_FRONTEND_ROOT ?? '';
  const candidate = cliRoot || envRoot;
  if (candidate) {
    return path.resolve(candidate);
  }
  return path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
}

function patchFile(filePath, replacements) {
  const original = fs.readFileSync(filePath, 'utf8');
  let next = original;
  let changed = false;
  for (const [from, to, label] of replacements) {
    if (next.includes(from)) {
      next = next.replace(from, to);
      changed = true;
      continue;
    }
    if (next.includes(to)) {
      continue;
    }
    throw new Error(`Expected to find ${label} in ${filePath}`);
  }
  if (changed) {
    fs.writeFileSync(filePath, next);
  }
  return changed;
}

function buildTargets(packageRoot) {
  return [
    {
      file: path.join(packageRoot, 'es', '_util', 'react-dom.js'),
      replacements: [
        [
          "import { isObject, isFunction, isReact18 } from './is';",
          "import { isObject, isFunction, isReact18, isReact19 } from './is';",
          'ES react-dom import',
        ],
        [
          [
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
          [
            'export var callbackOriginRef = function (children, node) {',
            '    var originRef = isReact19 ? children === null || children === void 0 ? void 0 : children.props.ref : children && children.ref;',
            '    if (originRef) {',
            '        if (isFunction(originRef)) {',
            '            originRef(node);',
            '        }',
            "        if ('current' in originRef) {",
            '            originRef.current = node;',
            '        }',
            '    }',
            '};',
          ].join('\n'),
          'ES react-dom ref callback',
        ],
      ],
    },
    {
      file: path.join(packageRoot, 'lib', '_util', 'react-dom.js'),
      replacements: [
        [
          [
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
          [
            'var callbackOriginRef = function (children, node) {',
            '    var originRef = is_1.isReact19 ? children === null || children === void 0 ? void 0 : children.props.ref : children && children.ref;',
            '    if (originRef) {',
            '        if ((0, is_1.isFunction)(originRef)) {',
            '            originRef(node);',
            '        }',
            "        if ('current' in originRef) {",
            '            originRef.current = node;',
            '        }',
            '    }',
            '};',
          ].join('\n'),
          'CJS react-dom ref callback',
        ],
      ],
    },
    {
      file: path.join(packageRoot, 'es', 'Trigger', 'index.js'),
      replacements: [
        [
          "import { isFunction, isObject, isArray, supportRef } from '../_util/is';",
          "import { isFunction, isObject, isArray, supportRef, isReact19 } from '../_util/is';",
          'ES Trigger import',
        ],
        [
          'React.createElement(popupChildren.type, __assign({ ref: popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
          'React.createElement(popupChildren.type, __assign({ ref: isReact19 ? popupChildren === null || popupChildren === void 0 ? void 0 : popupChildren.props.ref : popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
          'ES Trigger popup ref',
        ],
      ],
    },
    {
      file: path.join(packageRoot, 'lib', 'Trigger', 'index.js'),
      replacements: [
        [
          'react_1.default.createElement(popupChildren.type, __assign({ ref: popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
          'react_1.default.createElement(popupChildren.type, __assign({ ref: is_1.isReact19 ? popupChildren === null || popupChildren === void 0 ? void 0 : popupChildren.props.ref : popupChildren.ref }, popupChildren.props, { style: __assign(__assign({}, popupChildren.props.style), dropdownPopupStyle) })),',
          'CJS Trigger popup ref',
        ],
      ],
    },
    {
      file: path.join(packageRoot, 'es', 'AutoComplete', 'index.js'),
      replacements: [
        [
          "import useMergeProps from '../_util/hooks/useMergeProps';\nvar IMPOSSIBLE_VALUE = \"Autocomplete_\" + Math.random();",
          "import useMergeProps from '../_util/hooks/useMergeProps';\nimport { isReact19 } from '../_util/is';\nvar IMPOSSIBLE_VALUE = \"Autocomplete_\" + Math.random();",
          'ES AutoComplete import',
        ],
        [
          '    var originRef = usedTriggerElement.ref;',
          '    var originRef = isReact19 ? usedTriggerElement === null || usedTriggerElement === void 0 ? void 0 : usedTriggerElement.props.ref : usedTriggerElement.ref;',
          'ES AutoComplete origin ref',
        ],
      ],
    },
    {
      file: path.join(packageRoot, 'lib', 'AutoComplete', 'index.js'),
      replacements: [
        [
          'var useMergeProps_1 = __importDefault(require("../_util/hooks/useMergeProps"));\nvar IMPOSSIBLE_VALUE = "Autocomplete_" + Math.random();',
          'var useMergeProps_1 = __importDefault(require("../_util/hooks/useMergeProps"));\nvar is_1 = require("../_util/is");\nvar IMPOSSIBLE_VALUE = "Autocomplete_" + Math.random();',
          'CJS AutoComplete import',
        ],
        [
          '    var originRef = usedTriggerElement.ref;',
          '    var originRef = is_1.isReact19 ? usedTriggerElement === null || usedTriggerElement === void 0 ? void 0 : usedTriggerElement.props.ref : usedTriggerElement.ref;',
          'CJS AutoComplete origin ref',
        ],
      ],
    },
  ];
}

function clearViteCache(frontendRoot) {
  const cacheDirs = [
    path.join(frontendRoot, 'node_modules', '.vite'),
    path.join(frontendRoot, 'node_modules', '.vite-temp'),
  ];
  const cleared = [];
  for (const cacheDir of cacheDirs) {
    if (fs.existsSync(cacheDir)) {
      fs.rmSync(cacheDir, { recursive: true, force: true });
      cleared.push(path.relative(frontendRoot, cacheDir).replaceAll('\\', '/'));
    }
  }
  return cleared;
}

function main() {
  const frontendRoot = resolveFrontendRoot();
  const packageRoot = path.join(frontendRoot, 'node_modules', '@arco-design', 'web-react');

  if (!fs.existsSync(packageRoot)) {
    throw new Error(`Arco package not found at ${packageRoot}`);
  }

  const targets = buildTargets(packageRoot);
  const changedFiles = [];

  for (const target of targets) {
    if (!fs.existsSync(target.file)) {
      throw new Error(`Missing expected file: ${target.file}`);
    }
    const changed = patchFile(target.file, target.replacements);
    if (changed) {
      changedFiles.push(path.relative(frontendRoot, target.file).replaceAll('\\', '/'));
    }
  }

  const clearedCaches = clearViteCache(frontendRoot);

  if (changedFiles.length > 0) {
    console.log(`[arco-react19] patched ${changedFiles.length} file(s)`);
    for (const file of changedFiles) {
      console.log(`[arco-react19] ${file}`);
    }
    if (clearedCaches.length > 0) {
      console.log(`[arco-react19] cleared Vite cache: ${clearedCaches.join(', ')}`);
    }
    return;
  }

  if (clearedCaches.length > 0) {
    console.log(`[arco-react19] cleared Vite cache: ${clearedCaches.join(', ')}`);
  }
  console.log('[arco-react19] patch already applied');
}

main();
