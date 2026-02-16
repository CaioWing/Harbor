import { useEffect, useRef } from "react";

export function useAutoRefresh(callback, intervalMs, enabled = true) {
  const callbackRef = useRef(callback);

  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  useEffect(() => {
    if (!enabled || !intervalMs) {
      return undefined;
    }

    const timer = setInterval(() => {
      callbackRef.current();
    }, intervalMs);

    return () => clearInterval(timer);
  }, [enabled, intervalMs]);
}
