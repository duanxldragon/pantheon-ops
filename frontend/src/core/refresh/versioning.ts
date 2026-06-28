export function shouldPublishRefreshForVersionChange(
  previousVersion: number | undefined,
  nextVersion: number,
) {
  return previousVersion !== undefined && nextVersion > previousVersion;
}
