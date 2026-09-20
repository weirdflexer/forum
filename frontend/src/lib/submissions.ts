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
