import { ArrowLeft, LockKeyhole, Settings2 } from "lucide-react";
import { useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { api, mutation } from "../../api/client";
import type { Discussion, Post } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { ErrorBox, Loading } from "../../components/Feedback";
import { Pager } from "../../components/Pager";
import { ReasonModal } from "../../components/ReasonModal";
import { useLoad } from "../../hooks/useLoad";
import { countLabel, date } from "../../lib/format";
import { DeletePostModal } from "./DeletePostModal";
import { PostCard } from "./PostCard";
import { ReplyForm } from "./ReplyForm";
import { ReportModal } from "./ReportModal";

export function TopicPage() {
  const { id } = useParams();
  const { version, session, refresh, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const page = Number(params.get("page") ?? 1);
  const load = useLoad<Discussion>(`/topics/${id}/posts?page=${page}`, version);
  const [reportPost, setReportPost] = useState<Post | null>(null);
  const [deletePost, setDeletePost] = useState<Post | null>(null);
  const [topicMod, setTopicMod] = useState(false);
  if (load.loading && !load.data) return <Loading />;
  if (load.error)
    return (
      <div className="narrow-page">
        <Link className="back-link" to="/">
          ← К обсуждениям
        </Link>
        <ErrorBox text={load.error} retry={load.reload} />
      </div>
    );
  if (!load.data) return null;
  const { topic, items, total } = load.data;
  return (
    <div className="discussion-page">
      <Link className="back-link" to={"/?section_id=" + topic.section_id}>
        <ArrowLeft size={16} />
        {topic.section_title}
      </Link>
      <div className="discussion-heading">
        <span className={"tag " + topic.section_slug}>
          {topic.section_title}
        </span>
        <h1>{topic.title}</h1>
        <div className="topic-meta">
          <span>Аноним</span>
          <span>·</span>
          <time dateTime={topic.created_at}>{date(topic.created_at)}</time>
          <span>·</span>
          <span>
            {countLabel(Math.max(0, total - 1), ["ответ", "ответа", "ответов"])}
          </span>
          {topic.status === "closed" && (
            <span className="closed-label">
              <LockKeyhole size={14} />
              Тема закрыта
            </span>
          )}
        </div>
        {session.staff && (
          <button
            className="button ghost small mod-topic"
            onClick={() => setTopicMod(true)}
          >
            <Settings2 size={15} />
            {topic.status === "open" ? "Закрыть тему" : "Открыть тему"}
          </button>
        )}
      </div>
      <div className="posts-list">
        {items.map((p) => (
          <PostCard
            key={p.id}
            post={p}
            onReport={setReportPost}
            onDelete={setDeletePost}
          />
        ))}
      </div>
      <Pager
        page={page}
        total={total}
        onPage={(n) => {
          setParams(n > 1 ? { page: String(n) } : {});
          window.scrollTo(0, 0);
        }}
      />
      {topic.status === "open" ? (
        <ReplyForm
          topicId={id!}
          onPublished={(page) =>
            setParams(page > 1 ? { page: String(page) } : {})
          }
        />
      ) : (
        <div className="notice">
          <LockKeyhole size={18} />
          Модератор закрыл тему. Можно продолжить читать обсуждение.
        </div>
      )}
      {reportPost && (
        <ReportModal post={reportPost} onClose={() => setReportPost(null)} />
      )}
      {deletePost && (
        <DeletePostModal
          post={deletePost}
          onClose={() => setDeletePost(null)}
        />
      )}
      {topicMod && (
        <ReasonModal
          title={topic.status === "open" ? "Закрыть тему" : "Открыть тему"}
          label="Причина решения"
          button="Подтвердить"
          onClose={() => setTopicMod(false)}
          action={async (reason) => {
            await api(
              `/mod/topics/${id}`,
              mutation(
                "PATCH",
                { status: topic.status === "open" ? "closed" : "open", reason },
                session.staff!.csrf_token,
              ),
            );
            refresh();
            notify("Статус темы обновлён.");
          }}
        />
      )}
    </div>
  );
}
