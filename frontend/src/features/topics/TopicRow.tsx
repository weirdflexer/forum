import { ArrowRight, LockKeyhole, MessageCircle } from "lucide-react";
import { Link } from "react-router-dom";
import type { Topic } from "../../api/types";
import { SectionIcon } from "../../components/SectionIcon";
import { countLabel, relative } from "../../lib/format";

export function TopicRow({ topic: t }: { topic: Topic }) {
  return (
    <article className="topic-row">
      <div className={"topic-icon " + t.section_slug}>
        <SectionIcon slug={t.section_slug} size={21} />
      </div>
      <div className="topic-summary">
        <div className="topic-meta">
          <Link
            className={"tag " + t.section_slug}
            to={"/?section_id=" + t.section_id}
          >
            {t.section_title}
          </Link>
          <span>{relative(t.created_at)}</span>
          {t.status === "closed" && (
            <span className="closed-label">
              <LockKeyhole size={12} />
              Закрыта
            </span>
          )}
        </div>
        <h2>
          <Link to={"/topics/" + t.id}>{t.title}</Link>
        </h2>
        <p>{t.preview}</p>
        <div className="topic-bottom">
          <span className="author">
            <span className="anon-dot">?</span>Аноним
          </span>
          <Link className="reply-count" to={"/topics/" + t.id}>
            <MessageCircle size={15} />
            {countLabel(Math.max(0, t.post_count - 1), [
              "ответ",
              "ответа",
              "ответов",
            ])}
          </Link>
          <span className="last-activity">
            активность {relative(t.last_activity_at)}
          </span>
        </div>
      </div>
      <Link
        to={"/topics/" + t.id}
        className="row-arrow"
        aria-label={"Открыть: " + t.title}
      >
        <ArrowRight size={18} />
      </Link>
    </article>
  );
}
