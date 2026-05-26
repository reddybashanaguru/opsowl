"use client";

import { useEffect, useRef, useState, useCallback } from "react";

const IDLE_TIMEOUT_MS = (Number(process.env.NEXT_PUBLIC_IDLE_TIMEOUT_MINUTES) || 15) * 60_000;
const WARNING_LEAD_MS = 2 * 60_000; // show warning 2 min before logout

const ACTIVITY_EVENTS = ["mousemove", "mousedown", "keydown", "touchstart", "click"] as const;

interface IdleTimeoutOptions {
  onTimeout: () => void;
  enabled?: boolean;
}

export function useIdleTimeout({ onTimeout, enabled = true }: IdleTimeoutOptions) {
  const [warningVisible, setWarningVisible] = useState(false);
  const [secondsLeft, setSecondsLeft] = useState(120);

  const logoutTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const warningTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const countdownInterval = useRef<ReturnType<typeof setInterval> | null>(null);
  const onTimeoutRef = useRef(onTimeout);
  onTimeoutRef.current = onTimeout;

  const clearAll = useCallback(() => {
    if (logoutTimer.current) clearTimeout(logoutTimer.current);
    if (warningTimer.current) clearTimeout(warningTimer.current);
    if (countdownInterval.current) clearInterval(countdownInterval.current);
    logoutTimer.current = null;
    warningTimer.current = null;
    countdownInterval.current = null;
  }, []);

  const resetTimers = useCallback(() => {
    clearAll();
    setWarningVisible(false);
    setSecondsLeft(Math.floor(WARNING_LEAD_MS / 1000));

    warningTimer.current = setTimeout(() => {
      setWarningVisible(true);
      setSecondsLeft(Math.floor(WARNING_LEAD_MS / 1000));
      countdownInterval.current = setInterval(() => {
        setSecondsLeft((s) => Math.max(0, s - 1));
      }, 1_000);
    }, IDLE_TIMEOUT_MS - WARNING_LEAD_MS);

    logoutTimer.current = setTimeout(() => {
      clearAll();
      setWarningVisible(false);
      onTimeoutRef.current();
    }, IDLE_TIMEOUT_MS);
  }, [clearAll]);

  // Call this when the user explicitly dismisses the warning (stays logged in).
  const extendSession = useCallback(() => {
    resetTimers();
  }, [resetTimers]);

  useEffect(() => {
    if (!enabled) {
      clearAll();
      return;
    }

    resetTimers();

    const handleActivity = () => resetTimers();
    ACTIVITY_EVENTS.forEach((e) => window.addEventListener(e, handleActivity, { passive: true }));

    return () => {
      clearAll();
      ACTIVITY_EVENTS.forEach((e) => window.removeEventListener(e, handleActivity));
    };
  }, [enabled, resetTimers, clearAll]);

  return { warningVisible, secondsLeft, extendSession };
}
