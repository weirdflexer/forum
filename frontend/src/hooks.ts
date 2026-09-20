import { useEffect, useState } from "react";
import { api, message } from "./api";
export function useLoad<T>(path: string, version = 0) {
  const [data, setData] = useState<T | null>(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [retry, setRetry] = useState(0);
  useEffect(() => {
    const c = new AbortController();
    setLoading(true);
    setError("");
    api<T>(path, { signal: c.signal })
      .then((d) => {
        if (!c.signal.aborted) setData(d);
      })
      .catch((e) => {
        if (!c.signal.aborted) setError(message(e));
      })
      .finally(() => {
        if (!c.signal.aborted) setLoading(false);
      });
    return () => c.abort();
  }, [path, version, retry]);
  return { data, error, loading, reload: () => setRetry((x) => x + 1) };
}
export function useDraft<T>(key: string, initial: T) {
  const [value, setValue] = useState<T>(() => {
    try {
      return JSON.parse(sessionStorage.getItem(key) ?? "null") ?? initial;
    } catch {
      return initial;
    }
  });
  // Persist synchronously, including when navigation immediately unmounts the form.
  const write = (next: T) => {
    try {
      sessionStorage.setItem(key, JSON.stringify(next));
    } catch {
      /* In-memory draft remains available. */
    }
    setValue(next);
  };
  return [value, write] as const;
}
export function submissionKey(name: string, payload: unknown) {
  const body = JSON.stringify(payload),
    storage = "forum:submission:" + name;
  try {
    const saved = JSON.parse(sessionStorage.getItem(storage) ?? "null");
    if (saved?.body === body && saved?.key) return saved.key as string;
  } catch {
    /* Create a new key. */
  }
  const key = crypto.randomUUID();
  try {
    sessionStorage.setItem(storage, JSON.stringify({ body, key }));
  } catch {
    /* Still idempotent during this request. */
  }
  return key;
}
export function clearSubmission(name: string) {
  try {
    sessionStorage.removeItem("forum:submission:" + name);
  } catch {
    /* Storage may be disabled. */
  }
}
