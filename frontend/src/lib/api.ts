export interface ShortenResponse {
  code: string;
  short_url: string;
  original_url: string;
}

export interface URLMetadata {
  code: string;
  original_url: string;
  created_at: string;
}

export interface ApiError {
  error: string;
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

export async function shortenUrl(url: string): Promise<ShortenResponse> {
  const endpoint = `${API_BASE}/api/shorten`;
  const res = await fetch(endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ url: url.trim() }),
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

