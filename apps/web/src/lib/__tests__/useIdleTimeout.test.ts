// @vitest-environment jsdom
import { renderHook, act } from "@testing-library/react";
import { useIdleTimeout } from "../hooks/useIdleTimeout";

// ── helpers ───────────────────────────────────────────────────────────────────

function setup(overrideMs?: number) {
  // Expose timers via vitest fake timers.
  vi.useFakeTimers();

  const onTimeout = vi.fn();
  const { result, unmount } = renderHook(() =>
    useIdleTimeout({ onTimeout, enabled: true }),
  );

  return { result, unmount, onTimeout };
}

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

// ── tests ─────────────────────────────────────────────────────────────────────

describe("useIdleTimeout", () => {
  it("does not fire when disabled", () => {
    vi.useFakeTimers();
    const onTimeout = vi.fn();
    renderHook(() => useIdleTimeout({ onTimeout, enabled: false }));

    act(() => { vi.advanceTimersByTime(20 * 60_000); });
    expect(onTimeout).not.toHaveBeenCalled();
  });

  it("warningVisible is false initially", () => {
    const { result } = setup();
    expect(result.current.warningVisible).toBe(false);
  });

  it("shows warning before the idle timeout fires", () => {
    const { result } = setup();
    const IDLE_MS = (Number(process.env.NEXT_PUBLIC_IDLE_TIMEOUT_MINUTES) || 15) * 60_000;
    const WARNING_MS = 2 * 60_000;

    // Advance past the warning threshold (IDLE - 2 min).
    act(() => { vi.advanceTimersByTime(IDLE_MS - WARNING_MS + 100); });
    expect(result.current.warningVisible).toBe(true);
  });

  it("fires onTimeout after full idle period", () => {
    const { result, onTimeout } = setup();
    const IDLE_MS = (Number(process.env.NEXT_PUBLIC_IDLE_TIMEOUT_MINUTES) || 15) * 60_000;

    act(() => { vi.advanceTimersByTime(IDLE_MS + 100); });
    expect(onTimeout).toHaveBeenCalledTimes(1);
    expect(result.current.warningVisible).toBe(false);
  });

  it("resets timers on user activity (mousemove)", () => {
    const { onTimeout } = setup();
    const IDLE_MS = (Number(process.env.NEXT_PUBLIC_IDLE_TIMEOUT_MINUTES) || 15) * 60_000;

    // Advance to just before warning.
    act(() => { vi.advanceTimersByTime(IDLE_MS - 3 * 60_000); });
    // Simulate activity — resets the idle clock.
    act(() => { window.dispatchEvent(new Event("mousemove")); });
    // Advance another IDLE_MS - 1s; should NOT have timed out.
    act(() => { vi.advanceTimersByTime(IDLE_MS - 1_000); });
    expect(onTimeout).not.toHaveBeenCalled();
  });

  it("extendSession resets timers and hides warning", () => {
    const { result, onTimeout } = setup();
    const IDLE_MS = (Number(process.env.NEXT_PUBLIC_IDLE_TIMEOUT_MINUTES) || 15) * 60_000;
    const WARNING_MS = 2 * 60_000;

    // Show warning.
    act(() => { vi.advanceTimersByTime(IDLE_MS - WARNING_MS + 100); });
    expect(result.current.warningVisible).toBe(true);

    // User clicks "Stay Logged In".
    act(() => { result.current.extendSession(); });
    expect(result.current.warningVisible).toBe(false);

    // Advance another IDLE_MS - 1s; should still not have timed out.
    act(() => { vi.advanceTimersByTime(IDLE_MS - 1_000); });
    expect(onTimeout).not.toHaveBeenCalled();
  });

  it("clears timers on unmount", () => {
    const { unmount, onTimeout } = setup();
    const IDLE_MS = (Number(process.env.NEXT_PUBLIC_IDLE_TIMEOUT_MINUTES) || 15) * 60_000;

    unmount();
    act(() => { vi.advanceTimersByTime(IDLE_MS + 1_000); });
    expect(onTimeout).not.toHaveBeenCalled();
  });
});
