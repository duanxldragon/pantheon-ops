"""
S7780: Add String.raw prefix to regex template literals that contain escaped backslashes.
       Targets .mjs files under frontend/scripts/ and scripts/.
"""
import os
import re

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
SCRIPT_DIRS = [
    os.path.join(BASE_DIR, 'frontend', 'scripts'),
    os.path.join(BASE_DIR, 'scripts'),
]


def find_template_literal_end(text, start):
    """Find the closing backtick of a template literal starting at `start` (just after opening backtick)."""
    i = start
    while i < len(text):
        ch = text[i]
        if ch == '`':
            return i
        if ch == '\\':
            # skip escaped char (e.g. \`, \n, \t, \\, etc.)
            # In the source text, \` is backslash + backtick (2 chars)
            i += 2
            continue
        if ch == '$' and i + 1 < len(text) and text[i + 1] == '{':
            # Skip interpolation: find matching }
            depth = 1
            i += 2
            while i < len(text) and depth > 0:
                if text[i] == '{':
                    depth += 1
                elif text[i] == '}':
                    depth -= 1
                elif text[i] == '"' or text[i] == "'" or text[i] == '`':
                    # Skip string literals inside interpolation
                    quote = text[i]
                    i += 1
                    while i < len(text) and text[i] != quote:
                        if text[i] == '\\':
                            i += 1
                        i += 1
                i += 1
            continue
        i += 1
    return -1


def fix_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        original = f.read()

    modified = original
    changes = 0

    # Find template literals inside new RegExp() that contain \\ sequences
    new_regexp_pattern = re.compile(r'new RegExp\(`')

    offset = 0
    while True:
        m = new_regexp_pattern.search(modified, offset)
        if not m:
            break

        backtick_pos = m.end() - 1           # position of the opening backtick
        tl_content_start = m.end()            # position after opening backtick

        tl_end = find_template_literal_end(modified, tl_content_start)
        if tl_end < 0:
            offset = m.end()
            continue

        tl_body = modified[tl_content_start:tl_end]

        # Check if this template literal contains \\ sequences (escaped backslashes)
        # In the file, \\ is two literal backslash characters
        if '\\\\' not in tl_body.replace('\\\\', ''):
            # Actually: check if tl_body contains two consecutive backslashes
            pass

        # Proper check: does tl_body contain "\\" (two backslash chars)?
        if '\\\\' not in tl_body:
            offset = tl_end + 1
            continue

        # Count and apply replacements: convert \\X sequences to \X
        # In Python string read from file, \\ appears as two backslash characters
        # Python regex r'\\' matches ONE literal backslash
        # Python regex r'\\\\' matches TWO literal backslashes (which is \\ in the source)
        # Replace \\X with \X (where X is any character including newlines)
        de_escaped_body, de_escaped_count = re.subn(
            r'\\\\(.)', r'\\\1', tl_body, flags=re.DOTALL
        )

        if de_escaped_count == 0:
            offset = tl_end + 1
            continue

        # Build the replacement: wrap with String.raw
        # Before: new RegExp(`...`)
        # After:  new RegExp(String.raw`...`)
        before_tl = modified[:backtick_pos]
        after_tl = modified[tl_end + 1:]
        modified = before_tl + 'String.raw`' + de_escaped_body + '`' + after_tl
        changes += 1

        # Advance offset past this replacement
        offset = len(before_tl) + len('String.raw`') + len(de_escaped_body) + 1

    if changes > 0:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(modified)
        print(f"  FIXED ({changes}): {os.path.relpath(filepath, BASE_DIR)}")
        return changes
    return 0


def main():
    print("=== S7780: String.raw prefix for regex template literals ===\n")
    total_files = 0
    total_fixes = 0

    for script_dir in SCRIPT_DIRS:
        if not os.path.isdir(script_dir):
            continue
        for root, dirs, files in os.walk(script_dir):
            for filename in files:
                if filename.endswith('.mjs'):
                    filepath = os.path.join(root, filename)
                    fixes = fix_file(filepath)
                    if fixes > 0:
                        total_files += 1
                        total_fixes += fixes

    print(f"\nResult: {total_files} files changed, {total_fixes} total fixes applied")


if __name__ == '__main__':
    main()
