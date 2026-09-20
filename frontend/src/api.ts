export type Section = {
  id: string;
  slug: string;
  title: string;
  description: string;
  is_archived: boolean;
  topic_count: number;
};
export type Topic = {
  id: string;
  section_id: string;
  section_title: string;
  section_slug: string;
  section_archived: boolean;
  title: string;
  status: "open" | "closed";
  created_at: string;
  last_activity_at: string;
  post_count: number;
  preview: string;
};
export type Post = {
  id: string;
  number: number;
  body: string;
  status: "visible" | "deleted" | "hidden";
  created_at: string;
  is_mine: boolean;
};
export type Session = {
  active: boolean;
  csrf_token: string;
  staff: null | {
    login: string;
    role: "admin" | "moderator";
    csrf_token: string;
  };
};
export type Page<T> = {
  items: T[];
  page: number;
  page_size: number;
  total: number;
};
export type Discussion = Page<Post> & { topic: Topic };
export type Report = {
  id: string;
  post_id: string;
  reason: string;
  comment: string;
  status: string;
  decision_reason: string;
  created_at: string;
  body: string;
  post_status: string;
  topic_id: string;
  number: number;
  topic_title: string;
};
export type Action = {
  id: string;
  login: string;
  action: string;
  reason: string;
  created_at: string;
  topic_id: string;
};
export type Staff = { id: string; login: string; role: "admin" | "moderator" };
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
export function date(value: string) {
  return new Intl.DateTimeFormat("ru", {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}
export function relative(value: string) {
  const diff = Math.max(
    0,
    Math.floor((Date.now() - new Date(value).getTime()) / 1000),
  );
  if (diff < 60) return "только что";
  if (diff < 3600) return `${Math.floor(diff / 60)} мин. назад`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} ч. назад`;
  return new Intl.DateTimeFormat("ru", {
    day: "numeric",
    month: "short",
  }).format(new Date(value));
}
export function message(e: unknown) {
  return e instanceof Error ? e.message : "Не удалось выполнить действие.";
}
export const reasons: Record<string, string> = {
  spam: "Спам",
  abuse: "Оскорбление",
  personal_data: "Личные данные",
  other: "Другое",
};

export function countLabel(n: number, forms: [string, string, string]) {
  const mod100 = n % 100;
  const mod10 = n % 10;
  const word =
    mod100 >= 11 && mod100 <= 14
      ? forms[2]
      : mod10 === 1
        ? forms[0]
        : mod10 >= 2 && mod10 <= 4
          ? forms[1]
          : forms[2];
  return `${n} ${word}`;
}
