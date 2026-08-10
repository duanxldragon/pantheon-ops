function normalizeTopics(topics: string | string[]) {
  return Array.isArray(topics) ? topics : [topics];
}

export function canonicalizeRefreshTopics(topics: string | string[]) {
  return Array.from(new Set(normalizeTopics(topics))).sort((a, b) => a.localeCompare(b));
}

export function buildRefreshTopicKey(topics: string | string[]) {
  return canonicalizeRefreshTopics(topics).join(',');
}
