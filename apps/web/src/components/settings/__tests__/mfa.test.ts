// @vitest-environment node
/**
 * Unit tests for MFA-related frontend logic — API calls, token handling, state
 * transitions. We test the API layer (fetch calls, response shapes) rather than
 * rendering the settings page, because AIConfigSection + MFASection are defined
 * as inline functions in the page file and not exported.
 *
 * These tests validate that:
 *  - The MFA enrollment API call returns the expected shape
 *  - The MFA challenge API call produces a token pair
 *  - The MFA status endpoint returns { active: boolean }
 *  - The AI mode endpoint returns { provider, mode, model? }
 *  - Error responses are surfaced as APIError with the correct status
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api, APIError } from "@/lib/api";

const API_URL = "http://localhost:8080";

// ── Mock fetch globally ────────────────────────────────────────────────────────

function mockFetch(status: number, body: unknown) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => "application/json" },
    json: async () => body,
    text: async () => JSON.stringify(body),
  });
}

// ── MFA Status endpoint ────────────────────────────────────────────────────────

describe("MFA status", () => {
  afterEach(() => vi.restoreAllMocks());

  it("returns { active: false } when MFA is not enrolled", async () => {
    vi.stubGlobal("fetch", mockFetch(200, { data: { active: false } }));
    const result = await api.get<{ active: boolean }>(`${API_URL}/api/v1/auth/mfa/status`);
    expect(result.active).toBe(false);
  });

  it("returns { active: true } when MFA is enrolled and active", async () => {
    vi.stubGlobal("fetch", mockFetch(200, { data: { active: true } }));
    const result = await api.get<{ active: boolean }>(`${API_URL}/api/v1/auth/mfa/status`);
    expect(result.active).toBe(true);
  });

  it("throws APIError 401 when unauthenticated", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(401, { error: { code: "UNAUTHORIZED", message: "not authenticated" } })
    );
    await expect(
      api.get(`${API_URL}/api/v1/auth/mfa/status`)
    ).rejects.toMatchObject({ status: 401 });
  });
});

// ── MFA Enroll endpoint ────────────────────────────────────────────────────────

describe("MFA enroll", () => {
  afterEach(() => vi.restoreAllMocks());

  it("returns secret, otpauth_url, and 8 backup codes on success", async () => {
    const payload = {
      data: {
        secret: "JBSWY3DPEHPK3PXP",
        otpauth_url: "otpauth://totp/OpsOwl:user@test.com?secret=JBSWY3DPEHPK3PXP&issuer=OpsOwl",
        backup_codes: ["ABCD1234", "EFGH5678", "IJKL9012", "MNOP3456", "QRST7890", "UVWX1234", "YZAB5678", "CDEF9012"],
      },
    };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.post<typeof payload["data"]>(`${API_URL}/api/v1/auth/mfa/enroll`, {});
    expect(result.secret).toBe("JBSWY3DPEHPK3PXP");
    expect(result.otpauth_url).toContain("otpauth://totp/");
    expect(result.backup_codes).toHaveLength(8);
    result.backup_codes.forEach((code, i) => {
      expect(code, `backup code at index ${i}`).toHaveLength(8);
    });
  });

  it("throws APIError 409 when MFA is already active", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(409, { error: { code: "MFA_ALREADY_ACTIVE", message: "MFA is already enabled" } })
    );
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/enroll`, {})
    ).rejects.toMatchObject({ status: 409 });
  });

  it("throws APIError 401 when unauthenticated", async () => {
    vi.stubGlobal("fetch", mockFetch(401, { error: { code: "UNAUTHORIZED", message: "not authenticated" } }));
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/enroll`, {})
    ).rejects.toMatchObject({ status: 401 });
  });
});

// ── MFA Verify Enroll endpoint ─────────────────────────────────────────────────

describe("MFA verify enrollment", () => {
  afterEach(() => vi.restoreAllMocks());

  it("returns success message on valid code", async () => {
    vi.stubGlobal("fetch", mockFetch(200, { data: { message: "MFA enabled successfully" } }));
    const result = await api.post<{ message: string }>(`${API_URL}/api/v1/auth/mfa/verify-enroll`, { code: "123456" });
    expect(result.message).toBe("MFA enabled successfully");
  });

  it("throws APIError 400 for wrong TOTP code", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(400, { error: { code: "INVALID_MFA_CODE", message: "code is incorrect" } })
    );
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/verify-enroll`, { code: "000000" })
    ).rejects.toMatchObject({ status: 400 });
  });

  it("throws APIError 409 when no pending enrollment", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(409, { error: { code: "NO_PENDING_ENROLLMENT", message: "start enrollment first" } })
    );
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/verify-enroll`, { code: "654321" })
    ).rejects.toMatchObject({ status: 409 });
  });
});

// ── MFA Challenge endpoint ─────────────────────────────────────────────────────

describe("MFA challenge", () => {
  afterEach(() => vi.restoreAllMocks());

  it("returns full token pair on valid TOTP code", async () => {
    const payload = {
      data: {
        access_token: "eyJhbGciOiJSUzI1NiJ9.test.sig",
        refresh_token: "rt_abcdef123456",
        token_type: "Bearer",
        expires_in: 3600,
        user: { id: "uuid-1", email: "user@test.com", full_name: "Test User", role: "analyst" },
      },
    };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.post<typeof payload["data"]>(`${API_URL}/api/v1/auth/mfa/challenge`, {
      mfa_token: "mfa.partial.token",
      code: "123456",
    });
    expect(result.access_token).toBeTruthy();
    expect(result.refresh_token).toMatch(/^rt_/);
    expect(result.user.email).toBe("user@test.com");
  });

  it("returns full token pair when using a backup code", async () => {
    const payload = {
      data: {
        access_token: "eyJhbGciOiJSUzI1NiJ9.backup.sig",
        refresh_token: "rt_backup789",
        token_type: "Bearer",
        expires_in: 3600,
        user: { id: "uuid-1", email: "user@test.com", full_name: "Test User", role: "analyst" },
      },
    };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.post<typeof payload["data"]>(`${API_URL}/api/v1/auth/mfa/challenge`, {
      mfa_token: "mfa.partial.token",
      code: "ABCD1234", // backup code format
    });
    expect(result.access_token).toBeTruthy();
  });

  it("throws APIError 401 for invalid code", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(401, { error: { code: "INVALID_MFA_CODE", message: "invalid or expired code" } })
    );
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/challenge`, { mfa_token: "tok", code: "000000" })
    ).rejects.toMatchObject({ status: 401 });
  });

  it("throws APIError 400 when mfa_token or code is missing", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(400, { error: { code: "INVALID_REQUEST", message: "mfa_token and code are required" } })
    );
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/challenge`, { mfa_token: "", code: "" })
    ).rejects.toMatchObject({ status: 400 });
  });
});

// ── Login → MFA gate ───────────────────────────────────────────────────────────

describe("Login response when MFA is required", () => {
  afterEach(() => vi.restoreAllMocks());

  it("returns mfa_required:true + mfa_token (no access_token) when user has MFA enabled", async () => {
    const payload = {
      data: {
        mfa_required: true,
        mfa_token: "eyJhbGciOiJSUzI1NiJ9.mfapending.sig",
      },
    };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.post<typeof payload["data"]>(`${API_URL}/api/v1/auth/login`, {
      email: "mfa@test.com",
      password: "TestP@ss1",
    });
    expect(result.mfa_required).toBe(true);
    expect(result.mfa_token).toBeTruthy();
    expect((result as Record<string, unknown>).access_token).toBeUndefined();
  });

  it("returns access_token directly when user has no MFA", async () => {
    const payload = {
      data: {
        access_token: "eyJhbGciOiJSUzI1NiJ9.full.sig",
        refresh_token: "rt_nomfa",
        token_type: "Bearer",
        expires_in: 3600,
        user: { id: "uuid-2", email: "nomfa@test.com", full_name: "No MFA", role: "analyst" },
      },
    };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.post<typeof payload["data"]>(`${API_URL}/api/v1/auth/login`, {
      email: "nomfa@test.com",
      password: "TestP@ss1",
    });
    expect(result.access_token).toBeTruthy();
    expect((result as Record<string, unknown>).mfa_required).toBeUndefined();
  });
});

// ── MFA Disable endpoint ───────────────────────────────────────────────────────

describe("MFA disable", () => {
  afterEach(() => vi.restoreAllMocks());

  it("returns 200 with success message on correct password", async () => {
    vi.stubGlobal("fetch", mockFetch(200, { data: { message: "MFA disabled" } }));
    const result = await api.post<{ message: string }>(`${API_URL}/api/v1/auth/mfa/disable`, {
      password: "TestP@ss1",
    });
    expect(result.message).toBeTruthy();
  });

  it("throws APIError 400 for wrong password", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(400, { error: { code: "INCORRECT_PASSWORD", message: "current password is incorrect" } })
    );
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/disable`, { password: "Wrong" })
    ).rejects.toMatchObject({ status: 400 });
  });

  it("throws APIError 401 when unauthenticated", async () => {
    vi.stubGlobal("fetch", mockFetch(401, { error: { code: "UNAUTHORIZED", message: "not authenticated" } }));
    await expect(
      api.post(`${API_URL}/api/v1/auth/mfa/disable`, { password: "any" })
    ).rejects.toMatchObject({ status: 401 });
  });
});

// ── AI Configuration mode endpoint ────────────────────────────────────────────

describe("AI configuration mode", () => {
  afterEach(() => vi.restoreAllMocks());

  it("returns air-gapped mode with Ollama provider", async () => {
    const payload = { data: { provider: "ollama", mode: "air-gapped", model: "llama3.1:8b" } };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.get<{ provider: string; mode: string; model?: string }>(
      `${API_URL}/api/v1/system/ai-mode`
    );
    expect(result.provider).toBe("ollama");
    expect(result.mode).toBe("air-gapped");
    expect(result.model).toBe("llama3.1:8b");
  });

  it("returns cloud mode with Anthropic provider", async () => {
    const payload = { data: { provider: "anthropic", mode: "cloud", model: null } };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.get<{ provider: string; mode: string; model: string | null }>(
      `${API_URL}/api/v1/system/ai-mode`
    );
    expect(result.provider).toBe("anthropic");
    expect(result.mode).toBe("cloud");
    expect(result.model).toBeNull();
  });

  it("returns cloud mode with OpenAI provider", async () => {
    const payload = { data: { provider: "openai", mode: "cloud", model: null } };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.get<{ provider: string; mode: string; model: string | null }>(
      `${API_URL}/api/v1/system/ai-mode`
    );
    expect(result.provider).toBe("openai");
    expect(result.mode).toBe("cloud");
  });

  it("air-gapped mode has model field set (not null)", async () => {
    const payload = { data: { provider: "ollama", mode: "air-gapped", model: "llama3.1:8b" } };
    vi.stubGlobal("fetch", mockFetch(200, payload));
    const result = await api.get<{ provider: string; mode: string; model?: string }>(
      `${API_URL}/api/v1/system/ai-mode`
    );
    expect(result.model).toBeTruthy();
  });

  it("throws when AI service is unreachable (500)", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(500, { error: { code: "INTERNAL_ERROR", message: "AI service unreachable" } })
    );
    await expect(
      api.get(`${API_URL}/api/v1/system/ai-mode`)
    ).rejects.toBeInstanceOf(APIError);
  });

  it("throws APIError 401 when not authenticated", async () => {
    vi.stubGlobal("fetch", mockFetch(401, { error: { code: "UNAUTHORIZED", message: "not authenticated" } }));
    await expect(
      api.get(`${API_URL}/api/v1/system/ai-mode`)
    ).rejects.toMatchObject({ status: 401 });
  });
});
