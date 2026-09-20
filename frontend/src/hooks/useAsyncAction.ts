import { useCallback, useEffect, useRef, useState } from "react";
import { message } from "../lib/errors";

// One action at a time, with shared error and pending state for forms/dialogs.
export function useAsyncAction() {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const inFlight = useRef(false);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const run = useCallback(async (action: () => Promise<void>) => {
    if (inFlight.current) return;
    inFlight.current = true;
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (error) {
      if (mounted.current) setError(message(error));
    } finally {
      inFlight.current = false;
      if (mounted.current) setBusy(false);
    }
  }, []);
  return { error, busy, run };
}
