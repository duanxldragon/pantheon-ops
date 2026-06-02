import fs from 'node:fs';
import path from 'node:path';
import ts from 'typescript';

function resolveImportPath(fromPath, specifier) {
  const basePath = path.resolve(path.dirname(fromPath), specifier);
  const candidates = [
    basePath,
    `${basePath}.ts`,
    `${basePath}.tsx`,
    `${basePath}.mjs`,
    `${basePath}.js`,
    `${basePath}.json`,
  ];

  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  throw new Error(`Unsupported resource import ${specifier} from ${fromPath}`);
}

function propertyNameToString(node, filePath) {
  if (ts.isIdentifier(node) || ts.isStringLiteral(node) || ts.isNumericLiteral(node)) {
    return node.text;
  }
  if (ts.isComputedPropertyName(node) && ts.isStringLiteralLike(node.expression)) {
    return node.expression.text;
  }
  throw new Error(`Unsupported property name in ${filePath}`);
}

function evaluateExpression(node, state) {
  if (ts.isParenthesizedExpression(node) || ts.isAsExpression(node) || ts.isSatisfiesExpression(node) || ts.isNonNullExpression(node)) {
    return evaluateExpression(node.expression, state);
  }
  if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
    return node.text;
  }
  if (ts.isNumericLiteral(node)) {
    return Number(node.text);
  }
  if (node.kind === ts.SyntaxKind.TrueKeyword) {
    return true;
  }
  if (node.kind === ts.SyntaxKind.FalseKeyword) {
    return false;
  }
  if (node.kind === ts.SyntaxKind.NullKeyword) {
    return null;
  }
  if (ts.isObjectLiteralExpression(node)) {
    const value = {};
    for (const property of node.properties) {
      if (ts.isPropertyAssignment(property)) {
        value[propertyNameToString(property.name, state.filePath)] = evaluateExpression(property.initializer, state);
        continue;
      }
      if (ts.isShorthandPropertyAssignment(property)) {
        value[property.name.text] = evaluateExpression(property.name, state);
        continue;
      }
      if (ts.isSpreadAssignment(property)) {
        const spreadValue = evaluateExpression(property.expression, state);
        if (!spreadValue || typeof spreadValue !== 'object' || Array.isArray(spreadValue)) {
          throw new Error(`Spread value must be an object in ${state.filePath}`);
        }
        Object.assign(value, spreadValue);
        continue;
      }
      throw new Error(`Unsupported object property in ${state.filePath}`);
    }
    return value;
  }
  if (ts.isIdentifier(node)) {
    const importedValue = state.imports.get(node.text);
    if (importedValue !== undefined) {
      return importedValue;
    }
    const cachedValue = state.values.get(node.text);
    if (cachedValue !== undefined) {
      return cachedValue;
    }
    const declaration = state.declarations.get(node.text);
    if (!declaration) {
      throw new Error(`Unknown identifier ${node.text} in ${state.filePath}`);
    }
    const resolvedValue = evaluateExpression(declaration, state);
    state.values.set(node.text, resolvedValue);
    return resolvedValue;
  }

  throw new Error(`Unsupported expression ${ts.SyntaxKind[node.kind]} in ${state.filePath}`);
}

export function loadResourceModule(modulePath, cache = new Map(), loading = new Set()) {
  const resolvedPath = path.resolve(modulePath);
  if (cache.has(resolvedPath)) {
    return cache.get(resolvedPath);
  }
  if (loading.has(resolvedPath)) {
    throw new Error(`Cyclic resource import detected for ${resolvedPath}`);
  }

  loading.add(resolvedPath);
  try {
    const source = fs.readFileSync(resolvedPath, 'utf8');
    const sourceFile = ts.createSourceFile(resolvedPath, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
    const state = {
      declarations: new Map(),
      filePath: resolvedPath,
      imports: new Map(),
      values: new Map(),
    };
    let exportExpression = null;

    for (const statement of sourceFile.statements) {
      if (ts.isImportDeclaration(statement)) {
        const clause = statement.importClause;
        const specifier = statement.moduleSpecifier;
        if (!clause?.name || clause.namedBindings || !ts.isStringLiteral(specifier)) {
          throw new Error(`Unsupported import shape in ${resolvedPath}`);
        }
        state.imports.set(
          clause.name.text,
          loadResourceModule(resolveImportPath(resolvedPath, specifier.text), cache, loading),
        );
        continue;
      }
      if (ts.isVariableStatement(statement)) {
        for (const declaration of statement.declarationList.declarations) {
          if (!ts.isIdentifier(declaration.name) || !declaration.initializer) {
            continue;
          }
          state.declarations.set(declaration.name.text, declaration.initializer);
        }
        continue;
      }
      if (ts.isExportAssignment(statement)) {
        exportExpression = statement.expression;
      }
    }

    if (!exportExpression) {
      throw new Error(`Missing export default in ${resolvedPath}`);
    }

    const exportedValue = evaluateExpression(exportExpression, state);
    if (!exportedValue || typeof exportedValue !== 'object' || Array.isArray(exportedValue)) {
      throw new Error(`Resource module must export an object in ${resolvedPath}`);
    }

    cache.set(resolvedPath, exportedValue);
    return exportedValue;
  } finally {
    loading.delete(resolvedPath);
  }
}
