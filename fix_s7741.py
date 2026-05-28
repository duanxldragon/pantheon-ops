"""
S7741: Replace `typeof x === 'undefined'` with `x === undefined`
       and `typeof x !== 'undefined'` with `x !== undefined`
       in all .ts/.tsx files under frontend/src/
"""
import os
import re
import glob

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
TARGET_DIR = os.path.join(BASE_DIR, 'frontend', 'src')

# Match typeof <expr> === 'undefined' or typeof <expr> !== 'undefined'
# The expression after typeof can be a dotted identifier (e.g. globalThis.document)
# or a simple identifier (e.g. BroadcastChannel, document)
RE_TYPEOF_EQ = re.compile(r"typeof\s+(\S+?)\s*===\s*['\"]undefined['\"]")
RE_TYPEOF_NEQ = re.compile(r"typeof\s+(\S+?)\s*!==\s*['\"]undefined['\"]")


def fix_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        original = f.read()

    modified = original
    changes = 0

    # Replace typeof x === 'undefined' with x === undefined
    new_text, n = RE_TYPEOF_EQ.subn(r'\1 === undefined', modified)
    changes += n
    modified = new_text

    # Replace typeof x !== 'undefined' with x !== undefined
    new_text, n2 = RE_TYPEOF_NEQ.subn(r'\1 !== undefined', modified)
    changes += n2
    modified = new_text

    if modified != original:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(modified)
        print(f"  FIXED ({changes}): {os.path.relpath(filepath, BASE_DIR)}")
        return changes
    return 0


def main():
    print("=== S7741: typeof undefined -> direct compare ===\n")
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
