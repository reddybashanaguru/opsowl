/**
 * API client — wraps fetch with auth header, base URL, and {data, meta, error} envelope handling.
 * All API calls go through here — never call fetch directly in components.
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export class APIError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number,
    public readonly details?: unknown,
  ) {
    super(message);
    this.name = "APIError";
  }
}

function getAuthHeader(): Record<string, string> {
  if (typeof window === "undefined") return {};
  // Import lazily to avoid circular deps — Zustand store is a singleton
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { useAuthStore } = require("@/lib/stores/auth.store");
  const token: string | null = useAuthStore.getState().accessToken;
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const url = `${API_BASE}${path}`;

  const response = await fetch(url, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...getAuthHeader(),
      ...options.headers,
    },
  });

  const envelope = await response.json() as {
    data?: T;
    meta?: unknown;
    error?: { code: string; message: string; details?: unknown };
  };

  if (!response.ok || envelope.error) {
    throw new APIError(
      envelope.error?.code ?? "UNKNOWN",
      envelope.error?.message ?? "An unexpected error occurred",
      response.status,
      envelope.error?.details,
    );
  }

  return envelope.data as T;
}

// For endpoints that return {"data":[...],"meta":{...}} — returns the full {data, meta} shape.
async function requestList<TItem, TMeta = Record<string, unknown>>(
  path: string,
  options: RequestInit = {},
): Promise<{ data: TItem[]; meta: TMeta }> {
  const url = `${API_BASE}${path}`;

  const response = await fetch(url, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...getAuthHeader(),
      ...options.headers,
    },
  });

  const envelope = await response.json() as {
    data?: TItem[];
    meta?: TMeta;
    error?: { code: string; message: string; details?: unknown };
  };

  if (!response.ok || envelope.error) {
    throw new APIError(
      envelope.error?.code ?? "UNKNOWN",
      envelope.error?.message ?? "An unexpected error occurred",
      response.status,
      envelope.error?.details,
    );
  }

  return { data: envelope.data ?? [], meta: envelope.meta as TMeta };
}

export const api = {
  get: <T>(path: string, init?: RequestInit) =>
    request<T>(path, { method: "GET", ...init }),

  // Use for list endpoints that return {data:[...], meta:{...}}
  getList: <TItem, TMeta = Record<string, unknown>>(path: string, init?: RequestInit) =>
    requestList<TItem, TMeta>(path, { method: "GET", ...init }),

  // For multipart/form-data uploads — does NOT set Content-Type (browser sets boundary).
  upload: <T>(path: string, formData: FormData): Promise<T> => {
    const url = `${API_BASE}${path}`;
    return fetch(url, {
      method: "POST",
      body: formData,
      credentials: "include",
      headers: { ...getAuthHeader() },
    }).then(async (resp) => {
      const envelope = await resp.json() as {
        data?: T;
        error?: { code: string; message: string; details?: unknown };
      };
      if (!resp.ok || envelope.error) {
        throw new APIError(
          envelope.error?.code ?? "UNKNOWN",
          envelope.error?.message ?? "Upload failed",
          resp.status,
          envelope.error?.details,
        );
      }
      return envelope.data as T;
    });
  },

  post: <T>(path: string, body: unknown, init?: RequestInit) =>
    request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
      ...init,
    }),

  put: <T>(path: string, body: unknown, init?: RequestInit) =>
    request<T>(path, {
      method: "PUT",
      body: JSON.stringify(body),
      ...init,
    }),

  patch: <T>(path: string, body: unknown, init?: RequestInit) =>
    request<T>(path, {
      method: "PATCH",
      body: JSON.stringify(body),
      ...init,
    }),

  delete: <T>(path: string, init?: RequestInit) =>
    request<T>(path, { method: "DELETE", ...init }),
};
