/**
 * API client — wraps fetch with base URL and {data, meta, error} envelope handling.
 * Auth is carried by the httpOnly access_token cookie (sent automatically via
 * credentials: "include"). No JS-readable token is stored or sent as a header.
 */

/** Generates a v4 UUID idempotency key for use with idempotentPost(). */
export function newIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  // Fallback for environments without crypto.randomUUID (Node < 19)
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === "x" ? r : (r & 0x3) | 0x8).toString(16);
  });
}

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
      ...options.headers,
    },
  });

  if (response.status === 204) {
    return undefined as T;
  }

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

  /**
   * Idempotent POST: attaches an Idempotency-Key header so the server replays
   * the cached response on duplicate submissions (network retries, button
   * double-clicks that trigger the same mutation key). Generate the key once
   * per user action using newIdempotencyKey() and pass the same value on retries.
   */
  idempotentPost: <T>(path: string, body: unknown, idempotencyKey: string, init?: RequestInit) =>
    request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
      ...init,
      headers: {
        "Idempotency-Key": idempotencyKey,
        ...(init?.headers ?? {}),
      },
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

  /** Fetches a file from the API and triggers a browser download. */
  download: async (path: string, filename: string): Promise<void> => {
    const url = `${API_BASE}${path}`;
    const resp = await fetch(url, { method: "GET", credentials: "include" });
    if (!resp.ok) {
      throw new APIError("DOWNLOAD_FAILED", "Export failed", resp.status);
    }
    const blob = await resp.blob();
    const objectUrl = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = objectUrl;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(objectUrl);
  },
};

/**
 * Issues a short-lived (90s) SSE ticket for use as ?token= on EventSource URLs.
 * EventSource cannot set custom headers, so the full access token must not appear
 * in URLs. This endpoint requires an authenticated session (cookie-based).
 */
export async function getSseTicket(): Promise<string> {
  const data = await api.post<{ ticket: string; expires_in: number }>("/api/v1/auth/sse-ticket", {});
  return data.ticket;
}
