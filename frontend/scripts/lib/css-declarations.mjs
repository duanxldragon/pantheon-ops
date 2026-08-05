export function hasCssDeclaration(block, property, expectedValue) {
  const normalizedValue = String(expectedValue).replace(/\\([()[\]])/g, '$1');
  const escapedProperty = property.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`);
  const escapedValue = normalizedValue.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`);
  const pattern = new RegExp(
    String.raw`(?:^|[;\n])\s*${escapedProperty}\s*:\s*${escapedValue}(?:\s*!important)?\s*;`,
    'i',
  );
  return pattern.test(block);
}
