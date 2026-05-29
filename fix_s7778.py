"""
S7778: Combine multiple consecutive items.push(...) calls into a single
       array spread or concatenation where possible.
       Targets .ts/.tsx files under frontend/src/.
"""
import os
import re

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
TARGET_DIR = os.path.join(BASE_DIR, 'frontend', 'src')


def skip_string(text, i, quote):
    """Skip past a string literal starting at i (where text[i] == quote)."""
    i += 1
    while i < len(text) and text[i] != quote:
        if text[i] == '\\':
            i += 1
        i += 1
    return i + 1  # return position after closing quote


def skip_template_literal(text, i):
    """Skip past a template literal starting at i (where text[i] == '`')."""
    i += 1
    while i < len(text) and text[i] != '`':
        if text[i] == '\\':
            i += 2  # skip escaped char
            continue
        if text[i] == '$' and i + 1 < len(text) and text[i + 1] == '{':
            i += 2
            depth = 1
            while i < len(text) and depth > 0:
                if text[i] == '{':
                    depth += 1
                elif text[i] == '}':
                    depth -= 1
                elif text[i] == '"' or text[i] == "'":
                    i = skip_string(text, i, text[i])
                    continue
                elif text[i] == '`':
                    i = skip_template_literal(text, i)
                    continue
                i += 1
            continue
        i += 1
    return i + 1


def find_closing_paren(text, start):
    """Find position of closing ')' matching the '(' at start-1."""
    depth = 1
    i = start
    while i < len(text) and depth > 0:
        ch = text[i]
        if ch == '(':
            depth += 1
        elif ch == ')':
            depth -= 1
        elif ch == '"' or ch == "'":
            i = skip_string(text, i, ch)
            continue
        elif ch == '`':
            i = skip_template_literal(text, i)
            continue
        i += 1
    return i - 1 if depth == 0 else -1  # position of closing ')'


def text_between_is_empty(text, start, end):
    """Check if there's only whitespace/comments between start and end."""
    between = text[start:end]
    # Remove single-line comments
    lines = between.split('\n')
    for line in lines:
        stripped = line.strip()
        if stripped and not stripped.startswith('//'):
            # Check for block comments
            no_comments = re.sub(r'/\*.*?\*/', '', stripped, flags=re.DOTALL).strip()
            if no_comments:
                return False
    return True


def get_push_groups(text):
    """Find groups of consecutive .push() calls on the same array variable."""
    push_re = re.compile(r'(\w+(?:\.\w+)*)\.push\(')
    groups = []
    current_group = None

    for m in push_re.finditer(text):
        var_name = m.group(1)
        arg_start = m.end()
        close_paren = find_closing_paren(text, arg_start)
        if close_paren < 0:
            continue

        # find end of statement (past ';' if present)
        stmt_end = close_paren + 1  # past ')'
        while stmt_end < len(text) and text[stmt_end] in ' \t':
            stmt_end += 1
        if stmt_end < len(text) and text[stmt_end] == ';':
            stmt_end += 1
        # stmt_end is now right after ';' (or after ')' if no ';')

        if current_group and current_group['var'] == var_name:
            # Check if only whitespace/comments between previous and this push
            if text_between_is_empty(text, current_group['stmt_end'], m.start()):
                current_group['pushes'].append({
                    'start': m.start(),
                    'stmt_end': stmt_end,
                    'args': text[arg_start:close_paren].strip(),
                })
                current_group['stmt_end'] = stmt_end
                continue

        # Start new group
        if current_group and len(current_group['pushes']) >= 2:
            groups.append(current_group)
        current_group = {
            'var': var_name,
            'pushes': [{
                'start': m.start(),
                'stmt_end': stmt_end,
                'args': text[arg_start:close_paren].strip(),
            }],
            'stmt_end': stmt_end,
        }

    if current_group and len(current_group['pushes']) >= 2:
        groups.append(current_group)

    return groups


def fix_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        original = f.read()

    groups = get_push_groups(original)
    if not groups:
        return 0

    # Apply fixes from end to start to preserve positions
    edits = []
    for group in groups:
        pushes = group['pushes']
        var_name = group['var']

        # Build consolidated push: var_name.push(arg1, arg2, ...);
        all_args = [p['args'] for p in pushes]

        # Determine indentation from the first push
        start_pos = pushes[0]['start']
        line_start = original.rfind('\n', 0, start_pos) + 1
        indent = original[line_start:start_pos]

        # Build the consolidated statement
        inner_indent = indent + '    '
        joined_args = (',\n' + inner_indent).join(all_args)

        # Determine trailing newlines after the last push statement
        trail_start = pushes[-1]['stmt_end']
        while trail_start < len(original) and original[trail_start] in ' \t':
            trail_start += 1

        replacement = f"{var_name}.push(\n{inner_indent}{joined_args},\n{indent});"

        # The range to replace: from first push start to last push stmt_end
        # plus any trailing whitespace before next non-whitespace content
        edit_start = pushes[0]['start']
        edit_end = pushes[-1]['stmt_end']

        # Consume trailing whitespace/newlines up to (but not including) next content
        while edit_end < len(original) and original[edit_end] in ' \t':
            edit_end += 1

        edits.append({
            'start': edit_start,
            'end': edit_end,
            'replacement': replacement,
            'count': len(pushes),
        })

    if not edits:
        return 0

    # Apply from end to start
    modified = original
    for edit in sorted(edits, key=lambda e: e['start'], reverse=True):
        modified = modified[:edit['start']] + edit['replacement'] + '\n' + modified[edit['end']:]

    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(modified)

    total_pushes = sum(e['count'] for e in edits)
    print(f"  FIXED ({len(edits)} groups, {total_pushes} pushes): {os.path.relpath(filepath, BASE_DIR)}")
    return total_pushes


def main():
    print("=== S7778: Array push consolidation ===\n")
    total_files = 0
    total_fixes = 0

    for root, dirs, files in os.walk(TARGET_DIR):
        for filename in files:
            if filename.endswith(('.ts', '.tsx')):
                filepath = os.path.join(root, filename)
                fixes = fix_file(filepath)
                if fixes > 0:
                    total_files += 1
                    total_fixes += fixes

    print(f"\nResult: {total_files} files changed, {total_fixes} total fixes applied")


if __name__ == '__main__':
    main()
