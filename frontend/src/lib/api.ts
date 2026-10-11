export interface ShortenResponse {
  code: string;
  short_url: string;
  original_url: string;
  expires_at?: string;
}

export interface URLMetadata {
  code: string;
  original_url: string;
  created_at: string;
  expires_at?: string;
}

export interface URLAnalytics {
  code: string;
  original_url: string;
  total_clicks: number;
  last_clicked?: string;
  browsers?: Record<string, number>;
  operating_systems?: Record<string, number>;
  devices?: Record<string, number>;
  top_referrers?: Record<string, number>;
  recent_clicks: Array<{
    ip: string;
    user_agent: string;
    referer: string;
    timestamp: string;
  }>;
}

export interface ApiError {
  error: string;
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

export async function shortenUrl(
  url: string,
  customCode?: string,
  expiresInHours?: number
): Promise<ShortenResponse> {
  const endpoint = `${API_BASE}/api/shorten`;
  const body: Record<string, unknown> = { url: url.trim() };
  if (customCode && customCode.trim()) {
    body.custom_code = customCode.trim();
  }
  if (expiresInHours && expiresInHours > 0) {
    body.expires_in_hours = expiresInHours;
  }

  const res = await fetch(endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });

  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || 'Failed to shorten URL');
  }

  return data as ShortenResponse;
}

export async function getUrlMetadata(code: string): Promise<URLMetadata> {
  const cleanCode = code.trim().replace(/^.*\/([a-zA-Z0-9_-]+)$/, '$1');
  const endpoint = `${API_BASE}/api/urls/${encodeURIComponent(cleanCode)}`;
  const res = await fetch(endpoint, {
    method: 'GET',
    headers: {
      'Accept': 'application/json',
    },
  });

  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || 'Short URL not found');
  }

  return data as URLMetadata;
}

export async function getUrlAnalytics(code: string): Promise<URLAnalytics> {
  const cleanCode = code.trim().replace(/^.*\/([a-zA-Z0-9_-]+)$/, '$1');
  const endpoint = `${API_BASE}/api/urls/${encodeURIComponent(cleanCode)}/analytics`;
  const res = await fetch(endpoint, {
    method: 'GET',
    headers: {
      'Accept': 'application/json',
    },
  });

  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || 'Failed to retrieve analytics');
  }

  return data as URLAnalytics;
}
