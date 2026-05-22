// @vitest-environment node
// Running in the node environment so that `typeof window === "undefined"` is
// true inside api.ts's getAuthHeader(), which causes it to return {} immediately
// and skip the lazy require('@/lib/stores/auth.store') call that would otherwise
// fail (CJS require can't resolve Vite '@/' aliases in the test runner).

import { newIdempotencyKey, api, APIError } from "../api";

// ── newIdempotencyKey ──────────────────────────────────────────────────────────

describe("newIdempotencyKey", () => {
  const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

  it("returns a valid v4 UUID", () => {
    expect(newIdempotencyKey()).toMatch(UUID_RE);
  });

  it("returns a unique value on each call", () => {
    const keys = new Set(Array.from({ length: 100 }, () => newIdempotencyKey()));
    expect(keys.size).toBe(100);
  });

  it("falls back to Math.random UUID when crypto.randomUUID is unavailable", () => {
    const original = (crypto as unknown as { randomUUID?: () => string }).randomUUID;
    (crypto as unknown as { randomUUID?: () => string }).randomUUID = undefined;
    try {
      expect(newIdempotencyKey()).toMatch(UUID_RE);
    } finally {
      (crypto as unknown as { randomUUID?: () => string }).randomUUID = original;
    }
  });
});

// ── api.idempotentPost ─────────────────────────────────────────────────────────

describe("api.idempotentPost", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ data: { id: "abc-123" } }),
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends an Idempotency-Key header with the provided key", async () => {
    const key = "test-key-123";
    await api.idempotentPost("/api/v1/cases", { title: "Test" }, key);

    const fetchMock = vi.mocked(fetch);
    const [, init] = fetchMock.mock.calls[0];
    const headers = init?.headers as Record<string, string>;
    expect(headers["Idempotency-Key"]).toBe(key);
  });

  it("uses POST method", async () => {
    await api.idempotentPost("/api/v1/cases", { title: "Test" }, "k1");
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(init?.method).toBe("POST");
  });

  it("serializes the body as JSON", async () => {
    const body = { title: "Test Case", priority: "high" };
    await api.idempotentPost("/api/v1/cases", body, "k2");
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(init?.body).toBe(JSON.stringify(body));
  });

  it("returns the data field from the response envelope", async () => {
    const result = await api.idempotentPost<{ id: string }>("/api/v1/cases", {}, "k3");
    expect(result).toEqual({ id: "abc-123" });
  });

  it("throws APIError when the server returns an error envelope", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        json: async () => ({
          error: { code: "VALIDATION_ERROR", message: "Title too short" },
        }),
      }),
    );

    await expect(api.idempotentPost("/api/v1/cases", {}, "k4")).rejects.toBeInstanceOf(APIError);
  });

  it("includes the error code and message from the envelope in the thrown APIError", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({
          error: { code: "CONFLICT", message: "Case already exists" },
        }),
      }),
    );

    const err = await api
      .idempotentPost("/api/v1/cases", {}, "k5")
      .catch((e: unknown) => e);

    expect(err).toBeInstanceOf(APIError);
    const apiErr = err as APIError;
    expect(apiErr.code).toBe("CONFLICT");
    expect(apiErr.message).toBe("Case already exists");
    expect(apiErr.status).toBe(409);
  });
});
