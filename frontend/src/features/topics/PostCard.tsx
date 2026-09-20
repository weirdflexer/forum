import { Flag, Trash2 } from "lucide-react";
import type { Post } from "../../api/types";
import { date } from "../../lib/format";

export function PostCard({
  post: p,
  onReport,
  onDelete,
}: {
  post: Post;
  onReport: (post: Post) => void;
  onDelete: (post: Post) => void;
}) {
  return (
    <article
      id={"post-" + p.number}
      className={"post " + (p.number === 1 ? "first-post" : "")}
    >
      <header>
        <span className={"anon-avatar " + (p.is_mine ? "mine" : "")}>?</span>
        <div>
          <strong>
            Аноним {p.is_mine && <span className="mine-tag">это вы</span>}
          </strong>
          <time dateTime={p.created_at}>{date(p.created_at)}</time>
        </div>
        <a href={"#post-" + p.number} className="post-number">
          #{p.number}
        </a>
      </header>
      <div className={"post-body " + (p.status !== "visible" ? "removed" : "")}>
        {p.status === "visible"
          ? p.body
          : p.status === "deleted"
            ? "Сообщение удалено автором."
            : "Сообщение скрыто модератором."}
      </div>
      {p.status === "visible" && (
        <footer>
          <button className="text-button" onClick={() => onReport(p)}>
            <Flag size={14} />
            Пожаловаться
          </button>
          {p.is_mine && (
            <button className="text-button" onClick={() => onDelete(p)}>
              <Trash2 size={14} />
              Удалить
            </button>
          )}
        </footer>
      )}
    </article>
  );
}
