export function compareStrings(left, right) {
  return String(left).localeCompare(String(right));
}

export function sortStrings(values) {
  return Array.from(values).sort(compareStrings);
}
