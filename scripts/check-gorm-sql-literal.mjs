#!/usr/bin/env node

// Guards the invariant that Sonar's Go taint engine cannot see: in the business modules,
// every GORM statement is built from SQL text and column names that are literals or package
// constants, and request data reaches the statement only through `?` bind parameters.
//
// That is what makes the 33 gosecurity:S3649 findings on backend/modules/business/* false
// positives. This checker turns the manual review into a maintained rule: if a later change
// concatenates a request value into SQL text, or builds a column name dynamically, the gate
// goes red instead of the finding being re-triaged by hand.
//
// Scope: backend/modules/business/** (ops-owned code). base-owned modules carry their own
// review and are not asserted here.

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const rootArgumentIndex = process.argv.indexOf('--root');
const root = rootArgumentIndex >= 0
  ? path.resolve(process.cwd(), process.argv[rootArgumentIndex + 1])
  : repoRoot;

// GORM sinks whose first argument is SQL text or a column name.
const TEXT_SINKS = ['Where', 'Or', 'Having', 'Order', 'Select', 'Group', 'Distinct', 'Joins', 'Pluck', 'Update', 'Raw', 'Exec', 'Table'];
// GORM sinks whose argument maps column names to values.
const MAP_SINKS = ['Updates', 'Assign'];

function goFiles(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) return goFiles(file);
    return entry.name.endsWith('.go') ? [file] : [];
  });
}

// Line-scoped waivers: `// gorm-sql-literal:allow <justification>` on the line above a sink
// call. Every waiver is counted and printed, so an exception stays visible in the gate output
// instead of silently reducing coverage.
const WAIVER = /gorm-sql-literal:allow\s*(.*)$/;

function collectWaivers(rawSource) {
  const waived = new Map();
  const lines = rawSource.split('\n');
  for (let i = 0; i < lines.length; i += 1) {
    const match = WAIVER.exec(lines[i]);
    // A waiver without a written justification is not a waiver.
    if (!match || match[1].trim() === '') continue;
    let target = i + 1;
    while (target < lines.length && (lines[target].trim() === '' || lines[target].trim().startsWith('//'))) target += 1;
    if (target >= lines.length) continue;
    waived.set(target + 1, { reason: match[1].trim(), marker: lines[i].trim() });
  }
  return waived;
}

// Strip comments so a SQL-shaped example inside a comment is not scanned, while line numbers
// stay aligned (newlines are preserved).
function stripGoComments(source) {
  let out = '';
  let i = 0;
  let state = 'code';
  while (i < source.length) {
    const ch = source[i];
    const next = source[i + 1];
    if (state === 'code') {
      if (ch === '/' && next === '/') { state = 'line'; out += '  '; i += 2; continue; }
      if (ch === '/' && next === '*') { state = 'block'; out += '  '; i += 2; continue; }
      if (ch === '"' || ch === '`' || ch === "'") { state = ch === '"' ? 'string' : ch === '`' ? 'raw' : 'rune'; out += ch; i += 1; continue; }
      out += ch; i += 1; continue;
    }
    if (state === 'line') {
      out += ch === '\n' ? '\n' : ' ';
      if (ch === '\n') state = 'code';
      i += 1; continue;
    }
    if (state === 'block') {
      if (ch === '*' && next === '/') { state = 'code'; out += '  '; i += 2; continue; }
      out += ch === '\n' ? '\n' : ' ';
      i += 1; continue;
    }
    // Inside a Go literal: copy verbatim, honouring escapes.
    if (state === 'string' || state === 'rune') {
      if (ch === '\\') { out += ch + (next ?? ''); i += 2; continue; }
      if (state === 'string' && ch === '"') state = 'code';
      if (state === 'rune' && ch === "'") state = 'code';
      out += ch; i += 1; continue;
    }
    if (ch === '`') state = 'code';
    out += ch;
    i += 1;
  }
  return out;
}

// Read one Go literal (quoted or raw) starting at `start`; return the value and the index after it.
function readLiteral(source, start) {
  const quote = source[start];
  let i = start + 1;
  let value = '';
  while (i < source.length) {
    const ch = source[i];
    if (quote === '`') {
      if (ch === '`') return { value, end: i + 1 };
      value += ch; i += 1; continue;
    }
    if (ch === '\\') { value += source[i + 1] ?? ''; i += 2; continue; }
    if (ch === quote) return { value, end: i + 1 };
    if (ch === '\n') return null;
    value += ch; i += 1;
  }
  return null;
}

// Collect the top-level arguments of the call whose opening paren is at `openParen`.
function readArguments(source, openParen) {
  const args = [];
  let depth = 1;
  let current = '';
  let i = openParen + 1;
  while (i < source.length) {
    const ch = source[i];
    if (ch === '"' || ch === '`' || ch === "'") {
      const literal = readLiteral(source, i);
      if (!literal) return null;
      current += source.slice(i, literal.end);
      i = literal.end;
      continue;
    }
    if (ch === '(' || ch === '[' || ch === '{') depth += 1;
    if (ch === ')' || ch === ']' || ch === '}') {
      depth -= 1;
      if (depth === 0) {
        if (current.trim() !== '') args.push(current.trim());
        return { args, end: i + 1 };
      }
    }
    if (ch === ',' && depth === 1) { args.push(current.trim()); current = ''; i += 1; continue; }
    current += ch;
    i += 1;
  }
  return null;
}

// `Where`/`Update` are also ordinary service method names, so a match only counts as a GORM
// sink when the receiver is a database handle or a chained GORM call. The name set is
// deliberately generous: a false match costs one extra literal check, an narrow set hides sinks.
const DB_HANDLE_TAIL = /^(?:q|\w*(?:db|tx|query|session|gorm|conn))$/i;

function isGormReceiver(source, matchIndex) {
  const trimmed = source.slice(0, matchIndex).trimEnd();
  if (!trimmed) return false;
  // A chained builder (.Model(&X{}).Where(...), s.hostQuery(scope).Order(...)) always ends in ")".
  if (trimmed.endsWith(')')) return true;
  const identifier = /([A-Za-z_]\w*)\s*$/.exec(trimmed);
  return !!identifier && DB_HANDLE_TAIL.test(identifier[1]);
}

// Package-level `const NAME = "..."` declarations that name a SQL fragment or column.
//
// A name only qualifies when it is declared as a const at package level and the package never
// uses that name in a mutable position: a `var`, an assignment, a short declaration or a function
// parameter. Without the exclusion, `var col = "id"; col = req.Sort; db.Order(col)` or a parameter
// that shadows a constant name would be accepted as provably literal, which is exactly the bypass
// this checker exists to close. Analysis is package-wide because a package-level constant is
// visible from every file in the directory.
function collectLiteralConstants(dir) {
  const sources = goFiles(dir).map((file) => stripGoComments(fs.readFileSync(file, 'utf8')));
  const constNames = new Set();
  const mutableNames = new Set();
  for (const source of sources) {
    scanDeclarationTargets(source, constNames, mutableNames);
    for (const parameter of parameterLists(source)) {
      for (const argument of parameter.split(',')) {
        const name = /^\s*([A-Za-z_]\w*)/.exec(argument);
        if (name) mutableNames.add(name[1]);
      }
    }
  }
  for (const name of mutableNames) constNames.delete(name);
  return constNames;
}

// Walks the file once, treating const blocks as declarations and every other assignment target as
// mutable. Const-block lines are consumed as a block so their entries are not misread as in-function
// assignments.
function scanDeclarationTargets(source, constNames, mutableNames) {
  const lines = source.split('\n');
  const assignmentTarget = /([A-Za-z_]\w*)\s*(?::=|\+=|-=|\*=|\/=|=(?!=))/g;
  for (let index = 0; index < lines.length; index += 1) {
    const trimmed = lines[index].trim();
    if (/^const\s*\($/.test(trimmed)) {
      index += 1;
      while (index < lines.length && lines[index].trim() !== ')') {
        const entry = /^([A-Za-z_]\w*)\s*(?:string\s*)?=\s*(?:"[^"]*"|`[^`]*`)$/.exec(lines[index].trim());
        if (entry) constNames.add(entry[1]);
        index += 1;
      }
      continue;
    }
    const single = /^(const|var)\s+([A-Za-z_]\w*)\s*(?:[\w.[\]]*\s*)?=\s*(?:"[^"]*"|`[^`]*`)/.exec(trimmed);
    if (single) {
      if (single[1] === 'const') constNames.add(single[2]);
      else mutableNames.add(single[2]);
      continue;
    }
    for (const match of trimmed.matchAll(assignmentTarget)) mutableNames.add(match[1]);
  }
}

function parameterLists(source) {
  const lists = [];
  const signature = /\bfunc\s*(?:\([^)]*\)\s*)?[A-Za-z_]\w*\s*\(/g;
  for (const match of source.matchAll(signature)) {
    const parsed = readArguments(source, match.index + match[0].length - 1);
    if (parsed) lists.push(parsed.args.join(','));
  }
  return lists;
}

// Split an expression on top-level `+` so a SQL fragment built by concatenation is inspected
// operand by operand instead of being accepted because it starts with a literal.
function splitTopLevelPlus(text) {
  const operands = [];
  let depth = 0;
  let token = '';
  for (let i = 0; i < text.length; i += 1) {
    const ch = text[i];
    if (ch === '"' || ch === '`' || ch === "'") {
      const literal = readLiteral(text, i);
      if (!literal) return [text];
      token += text.slice(i, literal.end);
      i = literal.end - 1;
      continue;
    }
    if (ch === '(' || ch === '[' || ch === '{') depth += 1;
    if (ch === ')' || ch === ']' || ch === '}') depth -= 1;
    if (ch === '+' && depth === 0) { operands.push(token); token = ''; continue; }
    token += ch;
  }
  operands.push(token);
  return operands;
}

function isLiteralOperand(operand, literalConstants) {
  const text = operand.trim();
  if (!text) return false;
  if (text.startsWith('"') || text.startsWith('`')) {
    const literal = readLiteral(text, 0);
    return !!literal && literal.end >= text.length;
  }
  if (/^(?:clause|gorm|datatypes)\.[A-Za-z_][\w.]*$/.test(text)) return true;
  if (/^[A-Za-z_]\w*$/.test(text)) return literalConstants.has(text);
  return false;
}

function isSqlTextArgument(arg, literalConstants) {
  if (!arg) return false;
  return splitTopLevelPlus(arg).every((operand) => isLiteralOperand(operand, literalConstants));
}

function truncate(text = '') {
  const flat = text.replaceAll(/\s+/g, ' ').trim();
  return flat.length > 48 ? `${flat.slice(0, 48)}...` : flat;
}

// `map[string]interface{}{ ... }` / `map[string]any{ ... }` -> the entry text inside the literal.
function mapLiteralBody(mapText) {
  const header = /^map\[[^\]]*\]\s*(?:interface\s*\{\}|[A-Za-z_][\w.\[\]]*)?\s*\{/.exec(mapText);
  if (!header) return null;
  const body = mapText.slice(header[0].length);
  const close = body.lastIndexOf('}');
  return close < 0 ? null : body.slice(0, close);
}

// Entry keys of a map literal that are not string literals.
function dynamicMapKeys(mapText) {
  const body = mapLiteralBody(mapText);
  if (body === null) return [];
  const parts = [];
  let depth = 0;
  let token = '';
  for (let i = 0; i < body.length; i += 1) {
    const ch = body[i];
    if (ch === '"' || ch === '`' || ch === "'") {
      const literal = readLiteral(body, i);
      if (!literal) break;
      token += body.slice(i, literal.end);
      i = literal.end - 1;
      continue;
    }
    if (ch === '(' || ch === '[' || ch === '{') depth += 1;
    if (ch === ')' || ch === ']' || ch === '}') depth -= 1;
    if (ch === ',' && depth === 0) { parts.push(token); token = ''; continue; }
    token += ch;
  }
  if (token.trim()) parts.push(token);
  const keys = [];
  for (const part of parts) {
    const key = part.split(':')[0];
    if (!key || key.trim() === '') continue;
    const trimmed = key.trim();
    if (!trimmed.startsWith('"') && !trimmed.startsWith('`')) keys.push(trimmed);
  }
  return keys;
}

// `updates["name"] = value` style assignments to a map variable, with their keys.
function mapVariableKeys(source, variable) {
  const pattern = new RegExp(`\\b${variable}\\s*\\[\\s*([^\\]]+?)\\s*\\]\\s*=[^=]`, 'g');;
  return [...source.matchAll(pattern)].map((match) => match[1].trim());
}

function scanFile(file, source, rawSource, literalConstants, waived) {
  const violations = [];
  const relative = path.relative(root, file).replaceAll('\\', '/');
  const waivers = collectWaivers(rawSource);
  const sinks = [...TEXT_SINKS, ...MAP_SINKS];
  const callPattern = new RegExp(`\\.(${sinks.join('|')})\\(`, 'g');
  const lineOf = (index) => source.slice(0, index).split('\n').length;

  const report = (line, message) => {
    const waiver = waivers.get(line);
    if (waiver) {
      waived.push({ where: `${relative}:${line}`, reason: waiver.reason });
      return;
    }
    violations.push(`${relative}:${line}: ${message}`);
  };

  for (const match of source.matchAll(callPattern)) {
    const sink = match[1];
    const openParen = match.index + match[0].length - 1;
    const parsed = readArguments(source, openParen);
    if (!parsed) continue;
    if (!isGormReceiver(source, match.index)) continue;
    // Update(column, value) is GORM's two-argument shape; the Kubernetes clientset's
    // Update(ctx, object, options) shares the name but not the signature.
    if (sink === 'Update' && parsed.args.length !== 2) continue;
    const line = lineOf(match.index);

    if (TEXT_SINKS.includes(sink)) {
      const [sqlText] = parsed.args;
      if (!isSqlTextArgument(sqlText, literalConstants)) {
        report(line, `${sink}() SQL text or column name must be a literal or package constant, got ${truncate(sqlText)}`);
      }
      continue;
    }

    for (const arg of parsed.args) {
      if (arg.startsWith('map[')) {
        for (const key of dynamicMapKeys(arg)) {
          report(line, `${sink}() column name must be a literal, got dynamic key ${truncate(key)}`);
        }
        continue;
      }
      if (!/^[A-Za-z_]\w*$/.test(arg)) continue;
      const keys = mapVariableKeys(source, arg);
      for (const key of keys) {
        if (!key.startsWith('"') && !key.startsWith('`')) {
          report(line, `${sink}() column name must be a literal, map ${arg} is keyed by ${truncate(key)}`);
        }
      }
    }
  }
  return violations;
}

function main() {
  const businessRoot = path.join(root, 'backend', 'modules', 'business');
  if (!fs.existsSync(businessRoot)) {
    console.error(`backend/modules/business not found under ${root}`);
    process.exit(2);
  }
  const violations = [];
  const waived = [];
  const constantCache = new Map();
  for (const file of goFiles(businessRoot)) {
    if (file.endsWith('_test.go')) continue;
    const rawSource = fs.readFileSync(file, 'utf8');
    const dir = path.dirname(file);
    if (!constantCache.has(dir)) constantCache.set(dir, collectLiteralConstants(dir));
    violations.push(...scanFile(file, stripGoComments(rawSource), rawSource, constantCache.get(dir), waived));
  }
  for (const entry of waived) {
    console.log(`waived ${entry.where}: ${entry.reason}`);
  }
  if (violations.length > 0) {
    console.error(violations.join('\n'));
    console.error(`\n${violations.length} GORM SQL-literal violation(s)`);
    process.exit(1);
  }
  console.log('GORM SQL text and column names are literals or package constants across business modules');
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
