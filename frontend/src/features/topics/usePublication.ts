import { useRef } from "react";
import { api, mutation } from "../../api/client";
import { useApp } from "../../app/AppProvider";
import { clearSubmission, submissionKey } from "../../lib/submissions";

// Keep the same key for the same payload until the server confirms success,
// including a page reload after the response was lost on the network.
export function usePublication(name: string, path: string) {
  const { ensureSession } = useApp();
  const request = useRef<{ name: string; body: string; key: string } | null>(
    null,
  );
  return async function publish<T>(payload: unknown): Promise<T> {
    const session = await ensureSession();
    const body = JSON.stringify(payload);
    if (request.current?.name !== name || request.current.body !== body) {
      request.current = { name, body, key: submissionKey(name, payload) };
    }
    const result = await api<T>(
      path,
      mutation("POST", payload, session.csrf_token, request.current.key),
    );
    clearSubmission(name);
    request.current = null;
    return result;
  };
}
