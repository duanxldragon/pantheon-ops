import { describe, it, expect, vi } from 'vitest';
import { fetchAllPages } from '../../../src/api/fetchAllPages';

describe('fetchAllPages', () => {
  it('returns a single page when everything fits', async () => {
    const fetchPage = vi.fn(async (page: number, pageSize: number) => ({
      items: Array.from({ length: pageSize }, (_, i) => `row-${page}-${i}`),
      total: 100,
    }));

    const result = await fetchAllPages(fetchPage, 100);

    expect(result).toHaveLength(100);
    expect(fetchPage).toHaveBeenCalledTimes(1);
    expect(fetchPage).toHaveBeenCalledWith(1, 100);
  });

  it('aggregates across multiple pages until the set is complete', async () => {
    const fetchPage = vi.fn(async (page: number, pageSize: number) => {
      if (page === 3) {
        return { items: ['row-200', 'row-201'], total: 202 };
      }
      return {
        items: Array.from({ length: pageSize }, (_, i) => `row-${(page - 1) * pageSize + i + 1}`),
        total: 202,
      };
    });

    const result = await fetchAllPages(fetchPage, 100);

    expect(result).toHaveLength(202);
    expect(fetchPage).toHaveBeenCalledTimes(3);
  });

  it('stops exactly at the total boundary', async () => {
    const fetchPage = vi.fn(async (page: number, pageSize: number) => ({
      items:
        page === 2
          ? Array.from({ length: 50 }, (_, i) => `row-${page}-${i}`)
          : Array.from({ length: pageSize }, (_, i) => `row-${page}-${i}`),
      total: 150,
    }));

    const result = await fetchAllPages(fetchPage, 100);

    expect(result).toHaveLength(150);
    expect(fetchPage).toHaveBeenCalledTimes(2);
  });

  it('returns an empty array for an empty result', async () => {
    const fetchPage = vi.fn(async () => ({ items: [], total: 0 }));

    const result = await fetchAllPages(fetchPage);

    expect(result).toEqual([]);
    expect(fetchPage).toHaveBeenCalledTimes(1);
  });

  it('throws when maxPages is reached', async () => {
    // A back end that lies about total never returns a short page, so the
    // guard must reject instead of looping forever.
    const fetchPage = vi.fn(async (page: number, pageSize: number) => ({
      items: Array.from({ length: pageSize }, (_, i) => `row-${page}-${i}`),
      total: 100_000,
    }));

    await expect(fetchAllPages(fetchPage, 100, 3)).rejects.toThrow(/reached 3 pages/);
    expect(fetchPage).toHaveBeenCalledTimes(3);
  });
});