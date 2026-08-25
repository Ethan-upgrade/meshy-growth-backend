import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { fetchJSON } from '../api';

interface MockResponse {
  ok: boolean;
  status?: number;
  json?: () => Promise<Record<string, unknown>>;
}

describe('fetchJSON', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('should fetch and return JSON data', async () => {
    const mockData = { id: 1, name: 'Test Campaign' };
    const mockResponse: MockResponse = {
      ok: true,
      json: async () => mockData,
    };
    vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as Response);

    const result = await fetchJSON('/v1/campaigns');

    expect(result).toEqual(mockData);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/v1/campaigns'),
      expect.objectContaining({ cache: 'no-store' })
    );
  });

  it('should throw error when response is not ok', async () => {
    const mockResponse: MockResponse = {
      ok: false,
      status: 404,
    };
    vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as Response);

    await expect(fetchJSON('/v1/missing')).rejects.toThrow(
      'admin-api /v1/missing failed: 404'
    );
  });

  it('should use default API URL when no environment variable is set', async () => {
    const mockResponse: MockResponse = {
      ok: true,
      json: async () => ({}),
    };
    vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as Response);

    await fetchJSON('/v1/test');

    expect(global.fetch).toHaveBeenCalledWith(
      'http://admin-api:8080/v1/test',
      expect.any(Object)
    );
  });

  it('should pass through RequestInit options', async () => {
    const mockResponse: MockResponse = {
      ok: true,
      json: async () => ({}),
    };
    vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as Response);

    const headers = { 'X-Custom': 'value' };
    await fetchJSON('/v1/test', { method: 'POST', headers });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        method: 'POST',
        headers,
        cache: 'no-store',
      })
    );
  });

  it('should include cache: no-store in all requests', async () => {
    const mockResponse: MockResponse = {
      ok: true,
      json: async () => ({}),
    };
    vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as Response);

    await fetchJSON('/v1/events');

    expect(global.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        cache: 'no-store',
      })
    );
  });
});
