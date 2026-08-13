import { describe, it, expect } from 'vitest'
import { canonicalizeRefreshTopics, buildRefreshTopicKey } from '../../../../src/core/refresh/topicKey'

describe('canonicalizeRefreshTopics', () => {
  it('wraps a single string in an array', () => {
    expect(canonicalizeRefreshTopics('users')).toEqual(['users'])
  })

  it('deduplicates topics', () => {
    expect(canonicalizeRefreshTopics(['a', 'a', 'b'])).toEqual(['a', 'b'])
  })

  it('sorts topics alphabetically', () => {
    expect(canonicalizeRefreshTopics(['z', 'a', 'm'])).toEqual(['a', 'm', 'z'])
  })

  it('handles empty array', () => {
    expect(canonicalizeRefreshTopics([])).toEqual([])
  })
})

describe('buildRefreshTopicKey', () => {
  it('joins single topic', () => {
    expect(buildRefreshTopicKey('users')).toBe('users')
  })

  it('joins sorted deduplicated topics with comma', () => {
    expect(buildRefreshTopicKey(['roles', 'users', 'roles'])).toBe('roles,users')
  })

  it('returns empty string for empty array', () => {
    expect(buildRefreshTopicKey([])).toBe('')
  })
})
