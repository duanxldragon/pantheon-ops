function normalizeIdentityLabel(value: string) {
  return value.trim().toLocaleLowerCase();
}

export function getUserInitial(userDisplayName: string) {
  return userDisplayName.trim().slice(0, 1).toUpperCase();
}

export function shouldShowIdentityLabel(userDisplayName: string, label: string) {
  const normalizedLabel = normalizeIdentityLabel(label);
  return normalizedLabel.length > 0 && normalizedLabel !== normalizeIdentityLabel(userDisplayName);
}

export function filterIdentityLabels(userDisplayName: string, labels?: string[]) {
  return labels?.filter((label) => shouldShowIdentityLabel(userDisplayName, label)) ?? [];
}
