export class ApiError extends Error {
  status: number;
  code: string;
  requestId?: string;
  retryAfter?: string;
  constructor(
    message: string,
    status: number,
    code: string,
    requestId?: string,
    retryAfter?: string,
  ) {
    super(message);
    this.status = status;
    this.code = code;
    this.requestId = requestId;
    this.retryAfter = retryAfter;
  }
}
export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  let res: Response;
  try {
    res = await fetch("/api/v1" + path, {
      ...options,
      credentials: "same-origin",
      headers: {
        ...(options.body ? { "Content-Type": "application/json" } : {}),
        ...options.headers,
      },
    });
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") throw e;
    throw new ApiError(
      "Нет связи с сервером. Ваш текст сохранён в этой вкладке. Попробуйте ещё раз.",
      0,
      "network",
    );
  }
  if (!res.ok) {
    let e;
    try {
      e = await res.json();
    } catch {
      e = {
        message: "Сервер временно недоступен. Попробуйте позже.",
        code: "unavailable",
      };
    }
    throw new ApiError(
      e.message,
      res.status,
      e.code,
      e.request_id,
      res.headers.get("Retry-After") ?? undefined,
    );
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}
export function mutation(
  method: string,
  data: unknown,
  csrf: string,
  key?: string,
): RequestInit {
  return {
    method,
    headers: {
      "X-CSRF-Token": csrf,
      ...(key ? { "Idempotency-Key": key } : {}),
    },
    ...(data !== undefined ? { body: JSON.stringify(data) } : {}),
  };
}
