#!/usr/bin/env node

/**
 * Checker wiring gate (GOV-03, the FR-026 recurrence guard).
 *
 * A governance checker that nothing invokes is not coverage: it looks like
 * assurance while the class it was written for accumulates silently. This
 * script asks the only question that matters for every checker file — does
 * anything actually run this file?
 *
 * A checker counts as wired when a `package.json` or workflow run command
 * invokes its Node entrypoint (including npm-script indirection). Anything else must be
 * listed in EXCEPTIONS with a reason, mirroring the wiring table in
 * scripts/harness/README.md.
 *
 * Usage:
 *   node scripts/harness/check-checker-wiring.mjs [--root <path>] [--json]
 */

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const CHECKER_DIRECTORIES = ["scripts", path.join("scripts", "harness")];
const CHECKER_PATTERN = /^check-.*\.mjs$/;
const WORKFLOW_DIRECTORY = path.join(".github", "workflows");
const WORKFLOW_PATTERN = /\.ya?ml$/;

// Explicit, reviewable exceptions. Each entry must state why the checker is
// deliberately unreferenced; "not wired yet" is never a valid reason.
const EXCEPTIONS = new Map([
  [
    "check-doc-frontmatter.mjs",
    "mirror-only: scripts/frontmatter-check.mjs is the single implementation run as check:docs-frontmatter (2026-10-31 W5 decision)",
  ],
]);

function parseArgs(argv) {
  const options = { root: process.cwd(), json: false };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      options.json = true;
    } else if (arg === "--root") {
      const value = argv[index + 1];
      if (!value) {
        throw new Error("--root requires a path");
      }
      options.root = path.resolve(value);
      index += 1;
    } else {
      throw new Error(`unknown argument: ${arg}`);
    }
  }

  return options;
}

function listFiles(directory) {
  if (!fs.existsSync(directory)) {
    return [];
  }
  return fs
    .readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isFile())
    .map((entry) => entry.name)
    .sort((left, right) => left.localeCompare(right));
}

function collectCheckers(root) {
  const checkers = [];

  for (const relativeDirectory of CHECKER_DIRECTORIES) {
    const absoluteDirectory = path.join(root, relativeDirectory);
    for (const name of listFiles(absoluteDirectory)) {
      if (!CHECKER_PATTERN.test(name)) {
        continue;
      }
      checkers.push({
        name,
        file: path.posix.join(
          relativeDirectory.split(path.sep).join("/"),
          name,
        ),
      });
    }
  }

  return checkers;
}

// A bounded shell lexer: preserve quoted arguments, omit comments and split
// command operators. This is a configuration sensor, not a shell interpreter.
function commandTokens(text) {
  const commands = [];
  let tokens = [],
    token = "",
    quote = "";
  const flush = () => {
    if (token) tokens.push(token);
    token = "";
  };
  const end = () => {
    flush();
    if (tokens.length) commands.push(tokens);
    tokens = [];
  };
  for (let i = 0; i < text.length; i++) {
    const char = text[i];
    if (char === "\\" && quote !== "'" && /[\s"'\\#]/.test(text[i + 1] ?? "")) {
      token += text[++i];
    } else if (quote) {
      if (char === quote) quote = "";
      else token += char;
    } else if (char === '"' || char === "'") quote = char;
    else if (char === "#" && !token) {
      while (i < text.length && text[i] !== "\n") i++;
      end();
    } else if (/[\n;&|()]/.test(char)) end();
    else if (/\s/.test(char)) flush();
    else token += char;
  }
  end();
  return commands;
}

function invokedFiles(text, scripts, seen = new Set()) {
  const files = new Set();
  for (let tokens of commandTokens(text)) {
    while (/^(?:[A-Za-z_][\w]*=|then$|do$)/.test(tokens[0] ?? ""))
      tokens = tokens.slice(1);
    if (/^node(?:\.exe)?$/.test(tokens[0] ?? "")) {
      if (
        tokens.some((token) => /^(?:-e|-p|--eval|--print)(?:=|$)/.test(token))
      )
        continue;
      let index = 1;
      while (tokens[index]?.startsWith("-") && tokens[index] !== "-") {
        index += /^(?:-r|--require|--import|--loader)$/.test(tokens[index])
          ? 2
          : 1;
      }
      if (tokens[index])
        files.add(tokens[index].replaceAll("\\", "/").replace(/^\.\//, ""));
    } else if (tokens[0] === "npm" && tokens[1] === "run") {
      const alias = tokens[2];
      if (seen.has(alias) || typeof scripts[alias] !== "string") continue;
      seen.add(alias);
      for (const file of invokedFiles(scripts[alias], scripts, seen))
        files.add(file);
    }
  }
  return files;
}

/**
 * Read direct Node invocations and declared npm aliases. CI references describe
 * configuration, not observed execution or branch protection. Dynamic routing
 * and shell wrappers still need separate evidence.
 */
function readWiring(root) {
  const sources = [];
  let scripts = {};

  const packageJsonPath = path.join(root, "package.json");
  if (fs.existsSync(packageJsonPath)) {
    scripts =
      JSON.parse(fs.readFileSync(packageJsonPath, "utf8")).scripts ?? {};
    sources.push({
      file: "package.json",
      files: invokedFiles(
        Object.values(scripts)
          .filter((value) => typeof value === "string")
          .join("\n"),
        scripts,
      ),
      kind: "manual",
    });
  }

  const workflowDirectory = path.join(root, WORKFLOW_DIRECTORY);
  for (const name of listFiles(workflowDirectory)) {
    if (!WORKFLOW_PATTERN.test(name)) {
      continue;
    }
    const lines = fs
      .readFileSync(path.join(workflowDirectory, name), "utf8")
      .split(/\r?\n/);
    const commands = [];
    for (let index = 0; index < lines.length; index++) {
      const match = lines[index].match(/^(\s*)(?:-\s+)?run:\s*(.*)$/);
      if (!match) continue;
      if (/^[|>][-+]?\s*(?:#.*)?$/.test(match[2])) {
        const indent =
          match[1].length + (/^\s*-\s+run:/.test(lines[index]) ? 2 : 0);
        const block = [];
        while (
          index + 1 < lines.length &&
          (!lines[index + 1].trim() ||
            lines[index + 1].match(/^\s*/)[0].length > indent)
        ) {
          block.push(lines[++index]);
        }
        if (match[2].startsWith(">")) {
          while (block.length && !block.at(-1).trim()) block.pop();
          const contentIndent = block[0]?.match(/^\s*/)[0].length;
          if (
            block.some(
              (line) =>
                !line.trim() || line.match(/^\s*/)[0].length !== contentIndent,
            )
          ) {
            throw new Error(
              `Complex folded run in ${name}; use a literal | block for wiring inspection`,
            );
          }
          commands.push(block.map((line) => line.trim()).join(" "));
        } else commands.push(block.join("\n"));
      } else {
        const scalar = match[2].trim();
        // Common quoted YAML scalars; shell quotes inside block runs stay intact.
        if (scalar.startsWith('"') && scalar.endsWith('"'))
          commands.push(JSON.parse(scalar));
        else if (scalar.startsWith("'") && scalar.endsWith("'"))
          commands.push(scalar.slice(1, -1).replaceAll("''", "'"));
        else commands.push(scalar);
      }
    }
    sources.push({
      file: path.posix.join(".github/workflows", name),
      files: invokedFiles(commands.join("\n"), scripts),
      kind: "ci-reference",
    });
  }

  return sources;
}

function evaluate(root) {
  const sources = readWiring(root);
  const checkers = collectCheckers(root);

  const results = checkers.map((checker) => {
    const wiredIn = sources
      .filter((source) => source.files.has(checker.file))
      .map((source) => source.file);

    if (wiredIn.length > 0) {
      const invocation = sources.some(
        (source) =>
          source.kind === "ci-reference" && source.files.has(checker.file),
      )
        ? "ci-reference"
        : "manual-only";
      return { ...checker, status: "wired", wiredIn, invocation };
    }
    if (EXCEPTIONS.has(checker.name)) {
      return {
        ...checker,
        status: "exception",
        reason: EXCEPTIONS.get(checker.name),
      };
    }
    return { ...checker, status: "unwired" };
  });

  return results;
}

function main(argv) {
  const options = parseArgs(argv);
  const results = evaluate(options.root);
  const unwired = results.filter((result) => result.status === "unwired");
  const wired = results.filter((result) => result.status === "wired");
  const exceptions = results.filter((result) => result.status === "exception");
  const ciReferenced = wired.filter(
    (result) => result.invocation === "ci-reference",
  ).length;
  const manualOnly = wired.length - ciReferenced;

  if (options.json) {
    process.stdout.write(
      `${JSON.stringify({ total: results.length, wired: wired.length, ciReferenced, manualOnly, exceptions: exceptions.length, unwired, results }, null, 2)}\n`,
    );
  } else {
    for (const result of results) {
      if (result.status === "wired") {
        console.log(
          `wired      ${result.file} <- ${result.wiredIn.join(", ")} [${result.invocation}]`,
        );
      } else if (result.status === "exception") {
        console.log(`exception  ${result.file} (${result.reason})`);
      } else {
        console.error(
          `NOT WIRED  ${result.file} — no package.json script and no .github/workflows reference`,
        );
      }
    }
    console.log(
      `checker-wiring: ${results.length} checkers / ${wired.length} wired / ${exceptions.length} exception / ${unwired.length} unwired`,
    );
    console.log(
      `invocation: ${ciReferenced} CI command references / ${manualOnly} manual-only; required/observed CI status must be checked separately`,
    );
  }

  if (unwired.length > 0) {
    console.error(
      "\nA checker must land in the same diff as its wiring: an npm script plus a workflow step, " +
        "or an explicit exception in check-checker-wiring.mjs and the wiring table in scripts/harness/README.md.",
    );
    return 1;
  }
  return 0;
}

if (
  process.argv[1] &&
  path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))
) {
  try {
    process.exitCode = main(process.argv.slice(2));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 2;
  }
}

export { evaluate };
