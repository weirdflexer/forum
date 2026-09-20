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
