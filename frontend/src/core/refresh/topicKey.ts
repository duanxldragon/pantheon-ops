export type CanonicalRefreshTopic = string;

function normalizeTopics(topics: CanonicalRefreshTopic | CanonicalRefreshTopic[]) {
  return Array.isArray(topics) ? topics : [topics];
}

export function canonicalizeRefreshTopics(topics: CanonicalRefreshTopic | CanonicalRefreshTopic[]) {
  return Array.from(new Set(normalizeTopics(topics))).sort((a, b) => a.localeCompare(b));
}

export function buildRefreshTopicKey(topics: CanonicalRefreshTopic | CanonicalRefreshTopic[]) {
  return canonicalizeRefreshTopics(topics).join(',');
}
