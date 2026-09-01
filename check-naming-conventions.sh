#!/bin/bash

# Check file naming conventions for pantheon-ops project
# Go files should use snake_case
# TypeScript/TSX component files should use PascalCase
# TypeScript utility files should use camelCase

echo "=== Checking File Naming Conventions ==="
echo ""

# Check Go files (should be snake_case)
echo "=== Go Backend Files (should be snake_case) ==="
go_violations=()

while IFS= read -r file; do
    filename=$(basename "$file")
    # Skip _test.go files and valid snake_case files
    if [[ "$filename" =~ ^[a-z][a-z0-9]*(_[a-z0-9]+)*\.go$ ]] || \
       [[ "$filename" =~ ^[a-z][a-z0-9]*(_[a-z0-9]+)*_test\.go$ ]]; then
        continue
    fi
    # Check if it contains camelCase or PascalCase
    if [[ "$filename" =~ [A-Z] ]]; then
        go_violations+=("$file")
    fi
done < <(find backend -name "*.go" 2>/dev/null)

if [ ${#go_violations[@]} -gt 0 ]; then
    echo "Found ${#go_violations[@]} Go files with naming violations:"
    printf '%s\n' "${go_violations[@]}"
else
    echo "✓ All Go files follow snake_case convention"
fi
echo ""

# Check TypeScript/TSX files
echo "=== Frontend TypeScript/TSX Files ==="
ts_violations=()

while IFS= read -r file; do
    filename=$(basename "$file")
    dir=$(dirname "$file")

    # Skip test files, config files, and specific patterns
    if [[ "$filename" =~ \.(test|spec)\.(ts|tsx)$ ]] || \
       [[ "$filename" =~ ^(vite\.config|tsconfig|vitest\.config) ]] || \
       [[ "$filename" == "index.ts" ]] || \
       [[ "$filename" == "index.tsx" ]]; then
        continue
    fi

    # Check if it's a component file (in components directory or .tsx)
    if [[ "$dir" == *"/components"* ]] || [[ "$filename" == *.tsx ]]; then
        # Should be PascalCase
        if ! [[ "$filename" =~ ^[A-Z][a-zA-Z0-9]*\.(tsx|ts)$ ]]; then
            ts_violations+=("$file (should be PascalCase)")
        fi
    else
        # Utility/service files should be camelCase or kebab-case
        if [[ "$filename" =~ ^[A-Z] ]]; then
            # Check if it's an intentional PascalCase in non-component location
            if ! [[ "$filename" =~ \.(d\.ts|types\.ts)$ ]]; then
                ts_violations+=("$file (should be camelCase or kebab-case)")
            fi
        fi
    fi
done < <(find frontend/src -name "*.ts" -o -name "*.tsx" 2>/dev/null)

if [ ${#ts_violations[@]} -gt 0 ]; then
    echo "Found ${#ts_violations[@]} TypeScript/TSX files with naming violations:"
    printf '%s\n' "${ts_violations[@]}"
else
    echo "✓ All TypeScript/TSX files follow naming conventions"
fi
echo ""

# Summary
total_violations=$((${#go_violations[@]} + ${#ts_violations[@]}))
echo "=== Summary ==="
echo "Total violations found: $total_violations"
