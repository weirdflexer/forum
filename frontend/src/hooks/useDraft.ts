import { useState } from "react";

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
