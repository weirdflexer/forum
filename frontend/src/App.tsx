import {
ArrowDownWideNarrow,
ArrowLeft,
ArrowRight,
Check,
ChevronRight,
Compass,
Flag,
LockKeyhole,
LogOut,
MessageCircle,
Plus,
Search,
Send,
Settings2,
Shield,
ShieldCheck,
Sparkles,
Trash2,
X,
} from "lucide-react";
import type { FormEvent } from "react";
import { useEffect,useRef,useState } from "react";
import {
Link,
NavLink,
Route,
Routes,
useLocation,
useNavigate,
useParams,
useSearchParams,
} from "react-router-dom";
import { api,mutation } from "./api/client";
import type {
Action,
Discussion,
Page,
Post,
Report,
Section,
Session,
Staff,
Topic,
} from "./api/types";
import { AppProvider,useApp } from "./app/AppProvider";
import { Count } from "./components/Count";
import { Empty,ErrorBox,Loading } from "./components/Feedback";
import { Modal } from "./components/Modal";
import { Pager } from "./components/Pager";
import { SectionIcon } from "./components/SectionIcon";
import { reasons } from "./features/moderation/constants";
import { useDraft } from "./hooks/useDraft";
import { useLoad } from "./hooks/useLoad";
import { AppLayout } from "./layouts/AppLayout";
import { message } from "./lib/errors";
import { countLabel,date,relative } from "./lib/format";
import { clearSubmission,submissionKey } from "./lib/submissions";

export default function App() {
  const location = useLocation();
  return (
    <AppProvider>
      <AppLayout>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/new" element={<NewTopic />} />
          <Route
            path="/topics/:id"
            element={<TopicPage key={location.pathname} />}
          />
          <Route path="/login" element={<Login />} />
          <Route path="/moderation" element={<Moderation />} />
          <Route path="/admin" element={<Admin />} />
          <Route path="/rules" element={<InfoPage />} />
          <Route path="/about" element={<InfoPage about />} />
          <Route
            path="*"
            element={
              <Empty title="Такой страницы нет">
                <Link to="/">Вернуться к обсуждениям</Link>
              </Empty>
            }
          />
        </Routes>
      </AppLayout>
    </AppProvider>
  );
}

function Home() {
  const { sections, version } = useApp();
  const [params, setParams] = useSearchParams();
  const selected = params.get("section_id") ?? "",
    q = params.get("q") ?? "",
    sort = params.get("sort") ?? "active",
    page = Number(params.get("page") ?? 1);
  const [search, setSearch] = useState(q),
    [searchError, setSearchError] = useState("");
  const section = sections.find((s) => s.id === selected);
  const feed = useLoad<Page<Topic>>("/topics?" + params.toString(), version);
  useEffect(() => setSearch(q), [q]);
  const change = (updates: Record<string, string>) => {
    const p = new URLSearchParams(params);
    p.delete("page");
    for (const [k, v] of Object.entries(updates)) {
      if (v) p.set(k, v);
      else p.delete(k);
    }
    setParams(p);
  };
  const searchSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (search.trim() && Array.from(search.trim()).length < 2) {
      setSearchError("Введите хотя бы 2 символа.");
      return;
    }
    setSearchError("");
    change({ q: search.trim() });
  };
  return (
    <div className="home">
      <div className="page-heading">
        <div>
          <div className="eyebrow">МЕНЬШЕ МАСОК. БОЛЬШЕ СМЫСЛА.</div>
          <h1>
            {section ? section.title : "Все обсуждения"}
            <span className="heading-dot">.</span>
          </h1>
          <p>
            {section
              ? section.description
              : "Мысли, вопросы и истории. Здесь каждому есть место."}
          </p>
        </div>
        <Link
          className="button primary"
          to={"/new" + (selected ? "?section_id=" + selected : "")}
        >
          <Plus size={19} />
          Создать тему
        </Link>
      </div>
      {!selected && !q && page === 1 && (
        <section className="hero">
          <div className="hero-copy">
            <span className="hero-label">
              <span />
              БЕЗ РЕГИСТРАЦИИ. БЕЗ ЛИШНЕГО.
            </span>
            <h2>
              Разговор
              <br />
              начинается с тебя.
            </h2>
            <p>
              Задай вопрос. Поделись мыслью.
              <br />
              Или просто побудь среди своих.
            </p>
            <Link to="/new" className="hero-link">
              Начать обсуждение <ArrowRight size={18} />
            </Link>
          </div>
          <div className="hero-art" aria-hidden="true">
            <div className="orbit orbit-one" />
            <div className="orbit orbit-two" />
            <span className="art-spark">✳</span>
            <div className="chat-card chat-peach">
              <span />
              <span />
              <span />
            </div>
            <div className="chat-card chat-cream">
              <i />
              <i />
            </div>
            <div className="floating-note">
              <Sparkles size={14} />
              Твоё мнение имеет значение
            </div>
          </div>
        </section>
      )}
      <div className="content-grid">
        <section className="feed">
          <form className="search-box" onSubmit={searchSubmit}>
            <Search size={19} />
            <input
              aria-label="Поиск по обсуждениям"
              placeholder="Найти мысль, вопрос или обсуждение…"
              maxLength={100}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            {search && (
              <button
                type="button"
                className="icon-button"
                aria-label="Очистить поиск"
                onClick={() => {
                  setSearch("");
                  change({ q: "" });
                  setSearchError("");
                }}
              >
                <X size={15} />
              </button>
            )}
            <button className="search-submit" type="submit">
              Найти
            </button>
          </form>
          <ErrorBox text={searchError} />
          <div className="feed-tools">
            <div className="feed-title">
              {q ? `Результаты поиска «${q}»` : "Обсуждения"}
              <span>{feed.data?.total ?? "—"}</span>
            </div>
            <label className="sort">
              <ArrowDownWideNarrow size={15} />
              <select
                aria-label="Порядок тем"
                value={sort}
                onChange={(e) => change({ sort: e.target.value })}
              >
                <option value="active">По активности</option>
                <option value="new">Сначала новые</option>
              </select>
            </label>
          </div>
          {section?.is_archived && (
            <div className="notice">
              <LockKeyhole size={17} />
              Раздел в архиве. Обсуждения доступны для чтения.
            </div>
          )}
          <ErrorBox text={feed.error} retry={feed.reload} />
          {feed.loading ? (
            <Loading />
          ) : (
            !feed.error &&
            feed.data &&
            (feed.data.items.length ? (
              <div className="topic-list">
                {feed.data.items.map((t) => (
                  <TopicRow key={t.id} topic={t} />
                ))}
              </div>
            ) : (
              <Empty title={q ? "Ничего не нашлось" : "Здесь пока тихо"}>
                {q ? (
                  "Попробуйте другие слова или выберите другой раздел."
                ) : (
                  <>
                    Станьте первым, кто начнёт разговор.{" "}
                    <Link to={"/new?section_id=" + selected}>Создать тему</Link>
                  </>
                )}
              </Empty>
            ))
          )}
          <Pager
            page={page}
            total={feed.data?.total ?? 0}
            onPage={(n) => {
              const p = new URLSearchParams(params);
              p.set("page", String(n));
              setParams(p);
            }}
          />
        </section>
        <aside className="right-rail">
          <div className="rail-card welcome">
            <span className="mini-label">РАДЫ, ЧТО ТЫ ЗДЕСЬ</span>
            <h3>
              У каждого есть
              <br />
              что сказать.
            </h3>
            <p>
              За каждой мыслью — человек. Отвечай так, как хотел бы услышать в
              ответ.
            </p>
            <Link to="/rules">
              Наши простые правила <ArrowRight size={15} />
            </Link>
            <div className="friendly-faces" aria-hidden="true">
              <span>☺</span>
              <span>✳</span>
              <span>?</span>
              <small>
                Разные люди.
                <br />
                Общий разговор.
              </small>
            </div>
          </div>
          <div className="rail-card">
            <h3 className="rail-title">
              <Compass size={17} />
              Найди свой раздел
            </h3>
            {sections
              .filter((s) => !s.is_archived)
              .slice(0, 5)
              .map((s) => (
                <Link
                  key={s.id}
                  className="section-mini"
                  to={"/?section_id=" + s.id}
                >
                  <span className={"section-icon " + s.slug}>
                    <SectionIcon slug={s.slug} />
                  </span>
                  <span>
                    <strong>{s.title}</strong>
                    <small>
                      {countLabel(s.topic_count, [
                        "обсуждение",
                        "обсуждения",
                        "обсуждений",
                      ])}
                    </small>
                  </span>
                  <ChevronRight size={14} />
                </Link>
              ))}
          </div>
          <div className="privacy-note">
            <LockKeyhole size={15} />
            <p>
              Никаких публичных профилей.
              <br />
              Только ты и твои слова.
            </p>
          </div>
        </aside>
      </div>
    </div>
  );
}

function TopicRow({ topic: t }: { topic: Topic }) {
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

function NewTopic() {
  const { sections, ensureSession, refresh, notify } = useApp(),
    navigate = useNavigate(),
    [params] = useSearchParams();
  const [draft, setDraft] = useDraft("forum:new", {
    title: "",
    body: "",
    section_id: params.get("section_id") ?? "",
  });
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const request = useRef<{ body: string; key: string } | null>(null);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const session = await ensureSession(),
        payload = {
          ...draft,
          title: draft.title.trim(),
          body: draft.body.trim(),
        },
        serialized = JSON.stringify(payload);
      if (request.current?.body !== serialized)
        request.current = {
          body: serialized,
          key: submissionKey("new", payload),
        };
      const out = await api<{ topic_id: string }>(
        "/topics",
        mutation("POST", payload, session.csrf_token, request.current.key),
      );
      setDraft({ title: "", body: "", section_id: "" });
      clearSubmission("new");
      refresh();
      notify("Тема опубликована. Разговор начался!");
      navigate("/topics/" + out.topic_id);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="narrow-page">
      <Link to="/" className="back-link">
        <ArrowLeft size={16} />К обсуждениям
      </Link>
      <div className="page-heading">
        <div>
          <div className="eyebrow">ЕСТЬ ЧТО ОБСУДИТЬ?</div>
          <h1>
            Начни разговор<span className="heading-dot">.</span>
          </h1>
          <p>Хорошая тема начинается с простого вопроса.</p>
        </div>
      </div>
      <form className="form-card" onSubmit={submit}>
        <div className="form-intro">
          <span className="anon-avatar">?</span>
          <div>
            <strong>Аноним</strong>
            <p>Твоё имя останется за кадром</p>
          </div>
          <ShieldCheck size={22} />
        </div>
        <label>
          Раздел
          <select
            aria-label="Раздел"
            required
            value={draft.section_id}
            onChange={(e) => setDraft({ ...draft, section_id: e.target.value })}
          >
            <option value="">Выберите раздел</option>
            {sections
              .filter((s) => !s.is_archived)
              .map((s) => (
                <option value={s.id} key={s.id}>
                  {s.title}
                </option>
              ))}
          </select>
        </label>
        <label>
          Заголовок
          <input
            aria-label="Заголовок"
            required
            minLength={5}
            maxLength={240}
            value={draft.title}
            placeholder="О чём хочется поговорить?"
            onChange={(e) => setDraft({ ...draft, title: e.target.value })}
          />
          <Count text={draft.title} max={120} />
        </label>
        <label>
          Первое сообщение
          <textarea
            aria-label="Первое сообщение"
            required
            maxLength={10000}
            rows={9}
            value={draft.body}
            placeholder="Добавь контекст, задай вопрос или расскажи свою историю…"
            onChange={(e) => setDraft({ ...draft, body: e.target.value })}
          />
          <Count text={draft.body} max={5000} />
        </label>
        <ErrorBox text={error} />
        <div className="form-bottom">
          <p>
            Черновик хранится в этой вкладке.
            <br />
            Публикуя, соблюдай <Link to="/rules">правила общения</Link>.
          </p>
          <button className="button primary" disabled={busy}>
            {busy ? (
              "Публикуем…"
            ) : (
              <>
                Опубликовать
                <ArrowRight size={17} />
              </>
            )}
          </button>
        </div>
      </form>
    </div>
  );
}

function TopicPage() {
  const { id } = useParams();
  const { version, session, ensureSession, refresh, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const page = Number(params.get("page") ?? 1);
  const load = useLoad<Discussion>(`/topics/${id}/posts?page=${page}`, version);
  const [body, setBody] = useDraft("forum:reply:" + id, "");
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [reportPost, setReportPost] = useState<Post | null>(null),
    [deletePost, setDeletePost] = useState<Post | null>(null),
    [topicMod, setTopicMod] = useState(false);
  const request = useRef<{ body: string; key: string } | null>(null);
  const send = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const s = await ensureSession();
      const content = body.trim();
      if (request.current?.body !== content)
        request.current = {
          body: content,
          key: submissionKey("reply:" + id, { body: content }),
        };
      const out = await api<{ page: number }>(
        `/topics/${id}/posts`,
        mutation("POST", { body: content }, s.csrf_token, request.current.key),
      );
      setBody("");
      clearSubmission("reply:" + id);
      request.current = null;
      setParams(out.page > 1 ? { page: String(out.page) } : {});
      refresh();
      notify("Ответ опубликован.");
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };
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
          <article
            key={p.id}
            id={"post-" + p.number}
            className={"post " + (p.number === 1 ? "first-post" : "")}
          >
            <header>
              <span className={"anon-avatar " + (p.is_mine ? "mine" : "")}>
                ?
              </span>
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
            <div
              className={
                "post-body " + (p.status !== "visible" ? "removed" : "")
              }
            >
              {p.status === "visible"
                ? p.body
                : p.status === "deleted"
                  ? "Сообщение удалено автором."
                  : "Сообщение скрыто модератором."}
            </div>
            {p.status === "visible" && (
              <footer>
                <button
                  className="text-button"
                  onClick={() => setReportPost(p)}
                >
                  <Flag size={14} />
                  Пожаловаться
                </button>
                {p.is_mine && (
                  <button
                    className="text-button"
                    onClick={() => setDeletePost(p)}
                  >
                    <Trash2 size={14} />
                    Удалить
                  </button>
                )}
              </footer>
            )}
          </article>
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
        <form className="reply-form form-card" onSubmit={send}>
          <h2>
            <MessageCircle size={20} />
            Продолжить разговор
          </h2>
          <label className="sr-only" htmlFor="reply">
            Ваш ответ
          </label>
          <textarea
            id="reply"
            required
            maxLength={10000}
            rows={5}
            placeholder="Что ты думаешь об этом?"
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
          <Count text={body} max={5000} />
          <ErrorBox text={error} />
          <div className="form-bottom">
            <p>
              <ShieldCheck size={15} />
              Ответ будет опубликован анонимно
            </p>
            <button className="button primary" disabled={busy}>
              {busy ? (
                "Отправляем…"
              ) : (
                <>
                  Отправить ответ
                  <Send size={16} />
                </>
              )}
            </button>
          </div>
        </form>
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
        <ConfirmDelete post={deletePost} onClose={() => setDeletePost(null)} />
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

function ReportModal({ post, onClose }: { post: Post; onClose: () => void }) {
  const { ensureSession, notify } = useApp();
  const [reason, setReason] = useState("spam"),
    [comment, setComment] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  return (
    <Modal title="Сообщить о нарушении" onClose={onClose}>
      <p className="muted-text">
        Жалоба на сообщение #{post.number}. Модератор рассмотрит её и примет
        решение.
      </p>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError("");
          try {
            const s = await ensureSession();
            await api(
              "/reports",
              mutation(
                "POST",
                { post_id: post.id, reason, comment },
                s.csrf_token,
              ),
            );
            notify("Жалоба отправлена модератору.");
            onClose();
          } catch (e) {
            setError(message(e));
          } finally {
            setBusy(false);
          }
        }}
      >
        <label>
          Причина
          <select
            aria-label="Причина"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          >
            {Object.entries(reasons).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </select>
        </label>
        <label>
          Комментарий <span className="optional">необязательно</span>
          <textarea
            rows={3}
            maxLength={1000}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
        </label>
        <ErrorBox text={error} />
        <div className="dialog-actions">
          <button type="button" className="button ghost" onClick={onClose}>
            Отмена
          </button>
          <button className="button primary" disabled={busy}>
            Отправить жалобу
          </button>
        </div>
      </form>
    </Modal>
  );
}

function ConfirmDelete({ post, onClose }: { post: Post; onClose: () => void }) {
  const { session, refresh, notify } = useApp();
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  return (
    <Modal title="Удалить сообщение?" onClose={onClose}>
      <p className="muted-text">
        Текст сообщения #{post.number} будет удалён. На его месте останется
        отметка. Восстановить текст не получится.
      </p>
      <ErrorBox text={error} />
      <div className="dialog-actions">
        <button className="button ghost" onClick={onClose}>
          Оставить
        </button>
        <button
          className="button danger"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            try {
              await api(
                "/posts/" + post.id,
                mutation("DELETE", undefined, session.csrf_token),
              );
              refresh();
              notify("Сообщение удалено.");
              onClose();
            } catch (e) {
              setError(message(e));
            } finally {
              setBusy(false);
            }
          }}
        >
          Удалить сообщение
        </button>
      </div>
    </Modal>
  );
}

function ReasonModal({
  title,
  label,
  button,
  onClose,
  action,
}: {
  title: string;
  label: string;
  button: string;
  onClose: () => void;
  action: (reason: string) => Promise<void>;
}) {
  const [reason, setReason] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  return (
    <Modal title={title} onClose={onClose}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await action(reason.trim());
            onClose();
          } catch (e) {
            setError(message(e));
          } finally {
            setBusy(false);
          }
        }}
      >
        <label>
          {label}
          <textarea
            required
            maxLength={1000}
            rows={4}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        </label>
        <ErrorBox text={error} />
        <div className="dialog-actions">
          <button type="button" className="button ghost" onClick={onClose}>
            Отмена
          </button>
          <button className="button primary" disabled={busy}>
            {button}
          </button>
        </div>
      </form>
    </Modal>
  );
}

function Login() {
  const { session, setSession, notify } = useApp();
  const [login, setLogin] = useState(""),
    [password, setPassword] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const navigate = useNavigate();
  if (session.staff)
    return (
      <div className="narrow-page">
        <Empty title={"Вы вошли как " + session.staff.login}>
          <Link className="button primary" to="/moderation">
            Перейти к модерации
          </Link>
        </Empty>
      </div>
    );
  return (
    <div className="login-page">
      <span className="login-icon">
        <ShieldCheck size={34} />
      </span>
      <div className="eyebrow">ЗАБОТА О ПРОСТРАНСТВЕ</div>
      <h1>
        Вход для команды<span className="heading-dot">.</span>
      </h1>
      <p className="muted-text">
        Служебный доступ для модераторов и администраторов.
      </p>
      <form
        className="form-card"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError("");
          try {
            const s = await api<Session>("/staff/session", {
              method: "POST",
              body: JSON.stringify({ login, password }),
            });
            setSession(s);
            setPassword("");
            notify("Добро пожаловать в команду.");
            navigate("/moderation");
          } catch (e) {
            setError(message(e));
          } finally {
            setBusy(false);
          }
        }}
      >
        <label>
          Логин
          <input
            required
            autoComplete="username"
            maxLength={40}
            value={login}
            onChange={(e) => setLogin(e.target.value)}
          />
        </label>
        <label>
          Пароль
          <input
            required
            autoComplete="current-password"
            type="password"
            maxLength={128}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        <ErrorBox text={error} />
        <button className="button primary full" disabled={busy}>
          {busy ? "Входим…" : "Войти"}
          <ArrowRight size={17} />
        </button>
      </form>
      <Link className="back-link" to="/">
        <ArrowLeft size={15} />
        Вернуться к обсуждениям
      </Link>
    </div>
  );
}

function StaffHeader({ admin = false }: { admin?: boolean }) {
  const { session, setSession, notify } = useApp();
  const [error, setError] = useState("");
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow">
            {session.staff?.role === "admin" ? "АДМИНИСТРАТОР" : "МОДЕРАТОР"} ·{" "}
            {session.staff?.login}
          </div>
          <h1>
            {admin ? "Управление" : "Модерация"}
            <span className="heading-dot">.</span>
          </h1>
          <p>
            {admin
              ? "Разделы форума и доступ команды."
              : "Помогаем разговорам оставаться уважительными."}
          </p>
        </div>
        <button
          className="button ghost small"
          onClick={async () => {
            try {
              await api(
                "/staff/session",
                mutation("DELETE", undefined, session.staff!.csrf_token),
              );
              setSession({ ...session, staff: null });
              notify("Служебная сессия завершена.");
            } catch (e) {
              setError(message(e));
            }
          }}
        >
          <LogOut size={16} />
          Выйти
        </button>
      </div>
      <ErrorBox text={error} />
      <nav className="tabs">
        <NavLink end to="/moderation">
          <Flag size={16} />
          Жалобы и журнал
        </NavLink>
        {session.staff?.role === "admin" && (
          <NavLink to="/admin">
            <Settings2 size={16} />
            Управление
          </NavLink>
        )}
      </nav>
    </>
  );
}

function Moderation() {
  const { session } = useApp();
  return !session.staff ? (
    <div className="narrow-page">
      <Empty title="Нужен служебный вход" icon={Shield}>
        <Link to="/login" className="button primary">
          Войти
        </Link>
      </Empty>
    </div>
  ) : (
    <ModerationContent />
  );
}

function ModerationContent() {
  const { session, version, refresh, notify } = useApp();
  const [tab, setTab] = useState("open"),
    [page, setPage] = useState(1),
    [decision, setDecision] = useState<{
      report: Report;
      hide: boolean;
    } | null>(null);
  const reports = useLoad<Page<Report>>(
    "/mod/reports?status=" +
      (tab === "actions" ? "open" : tab) +
      "&page=" +
      page,
    version,
  );
  const actions = useLoad<Page<Action>>("/mod/actions?page=" + page, version);
  return (
    <div className="management-page">
      <StaffHeader />
      <div className="segmented" aria-label="Статус жалоб">
        {[
          ["open", "Ожидают решения"],
          ["resolved", "Скрытые"],
          ["dismissed", "Отклонённые"],
          ["actions", "Журнал действий"],
        ].map(([k, v]) => (
          <button
            key={k}
            className={tab === k ? "active" : ""}
            onClick={() => {
              setTab(k);
              setPage(1);
            }}
          >
            {v}
          </button>
        ))}
      </div>
      {tab === "actions" ? (
        <>
          <ErrorBox text={actions.error} retry={actions.reload} />
          {actions.loading ? (
            <Loading />
          ) : actions.data?.items.length ? (
            <div className="audit-list">
              {actions.data.items.map((a) => (
                <article className="audit-item" key={a.id}>
                  <ShieldCheck size={20} />
                  <div>
                    <strong>
                      {a.login} ·{" "}
                      {{
                        hide: "скрыл сообщение",
                        dismiss: "отклонил жалобу",
                        open: "открыл тему",
                        closed: "закрыл тему",
                      }[a.action] ?? a.action}
                    </strong>
                    <p>{a.reason}</p>
                    <small>
                      {date(a.created_at)} ·{" "}
                      <Link to={"/topics/" + a.topic_id}>К обсуждению</Link>
                    </small>
                  </div>
                </article>
              ))}
            </div>
          ) : (
            !actions.error && (
              <Empty title="Журнал пока пуст">
                Здесь появятся решения команды.
              </Empty>
            )
          )}
          <Pager
            page={page}
            total={actions.data?.total ?? 0}
            onPage={setPage}
          />
        </>
      ) : (
        <>
          <ErrorBox text={reports.error} retry={reports.reload} />
          {reports.loading ? (
            <Loading />
          ) : reports.data?.items.length ? (
            <div className="report-list">
              {reports.data.items.map((r) => (
                <article className="report-card" key={r.id}>
                  <div className="report-top">
                    <span className="tag warning">
                      <Flag size={12} />
                      {reasons[r.reason]}
                    </span>
                    <time>{date(r.created_at)}</time>
                  </div>
                  <Link
                    className="report-topic"
                    to={
                      "/topics/" +
                      r.topic_id +
                      "?page=" +
                      Math.ceil(r.number / 20) +
                      "#post-" +
                      r.number
                    }
                  >
                    {r.topic_title} · #{r.number}
                  </Link>
                  <blockquote>
                    {r.body || "Текст сообщения удалён автором."}
                  </blockquote>
                  {r.comment && (
                    <p className="muted-text">Комментарий: {r.comment}</p>
                  )}
                  {r.status === "open" ? (
                    <div className="report-actions">
                      <button
                        className="button ghost small"
                        onClick={() => setDecision({ report: r, hide: false })}
                      >
                        Отклонить жалобу
                      </button>
                      <button
                        className="button danger small"
                        onClick={() => setDecision({ report: r, hide: true })}
                      >
                        Скрыть сообщение
                      </button>
                    </div>
                  ) : (
                    <p className="decision-reason">
                      <Check size={15} />
                      {r.decision_reason}
                    </p>
                  )}
                </article>
              ))}
            </div>
          ) : (
            !reports.error && (
              <Empty title="Здесь всё спокойно" icon={ShieldCheck}>
                Жалоб в этой категории нет.
              </Empty>
            )
          )}
          <Pager
            page={page}
            total={reports.data?.total ?? 0}
            onPage={setPage}
          />
        </>
      )}
      {decision && (
        <ReasonModal
          title={decision.hide ? "Скрыть сообщение" : "Отклонить жалобу"}
          label="Причина решения — попадёт в журнал"
          button="Сохранить решение"
          onClose={() => setDecision(null)}
          action={async (reason) => {
            await api(
              `/mod/reports/${decision.report.id}/decision`,
              mutation(
                "POST",
                { decision: decision.hide ? "hide" : "dismiss", reason },
                session.staff!.csrf_token,
              ),
            );
            refresh();
            notify("Решение сохранено в журнале.");
          }}
        />
      )}
    </div>
  );
}

function Admin() {
  const { session } = useApp();
  return session.staff?.role !== "admin" ? (
    <div className="narrow-page">
      <Empty title="Доступ для администратора" icon={Shield}>
        <Link to="/login">Служебный вход</Link>
      </Empty>
    </div>
  ) : (
    <AdminContent />
  );
}

function AdminContent() {
  const { sections, session, refresh, version, notify, setSession } = useApp();
  const staff = useLoad<{ items: Staff[] }>("/admin/staff", version);
  const [editing, setEditing] = useState<Section | "new" | null>(null),
    [staffModal, setStaffModal] = useState(false),
    [error, setError] = useState(""),
    [pendingRole, setPendingRole] = useState<Staff | null>(null);
  return (
    <div className="management-page">
      <StaffHeader admin />
      <section className="admin-section">
        <div className="section-heading">
          <h2>Разделы</h2>
          <button
            className="button primary small"
            onClick={() => setEditing("new")}
          >
            <Plus size={16} />
            Добавить раздел
          </button>
        </div>
        <div className="admin-list">
          {sections.map((s) => (
            <article className="admin-row" key={s.id}>
              <span className={"section-icon " + s.slug}>
                <SectionIcon slug={s.slug} />
              </span>
              <div>
                <strong>
                  {s.title}{" "}
                  {s.is_archived && <span className="tag">Архив</span>}
                </strong>
                <p>{s.description}</p>
              </div>
              <button
                className="button ghost small"
                onClick={() => setEditing(s)}
              >
                Изменить
              </button>
            </article>
          ))}
        </div>
      </section>
      <section className="admin-section">
        <div className="section-heading">
          <h2>Команда</h2>
          <button
            className="button primary small"
            onClick={() => setStaffModal(true)}
          >
            <Plus size={16} />
            Добавить участника
          </button>
        </div>
        <ErrorBox text={staff.error || error} retry={staff.reload} />
        <div className="admin-list">
          {staff.data?.items.map((s) => (
            <article className="admin-row" key={s.id}>
              <span className="section-icon">
                <Shield size={20} />
              </span>
              <div>
                <strong>{s.login}</strong>
                <p>{s.role === "admin" ? "Администратор" : "Модератор"}</p>
              </div>
              <button
                className="button ghost small"
                onClick={() => setPendingRole(s)}
              >
                Изменить роль
              </button>
            </article>
          ))}
        </div>
      </section>
      {editing && (
        <SectionModal
          section={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}{" "}
      {staffModal && <StaffModal onClose={() => setStaffModal(false)} />}{" "}
      {pendingRole && (
        <Modal
          title="Изменить роль участника?"
          onClose={() => setPendingRole(null)}
        >
          <p className="muted-text">
            {pendingRole.login} получит роль «
            {pendingRole.role === "admin" ? "Модератор" : "Администратор"}». Его
            текущие служебные сессии будут завершены.
          </p>
          <div className="dialog-actions">
            <button
              className="button ghost"
              onClick={() => setPendingRole(null)}
            >
              Отмена
            </button>
            <button
              className="button primary"
              onClick={async () => {
                setError("");
                try {
                  await api(
                    `/admin/staff/${pendingRole.id}/role`,
                    mutation(
                      "PATCH",
                      {
                        role:
                          pendingRole.role === "admin" ? "moderator" : "admin",
                      },
                      session.staff!.csrf_token,
                    ),
                  );
                  if (pendingRole.login === session.staff!.login)
                    setSession({ ...session, staff: null });
                  refresh();
                  notify("Роль обновлена. Участнику нужно войти снова.");
                } catch (e) {
                  setError(message(e));
                } finally {
                  setPendingRole(null);
                }
              }}
            >
              Изменить роль
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

function SectionModal({
  section,
  onClose,
}: {
  section: Section | null;
  onClose: () => void;
}) {
  const { session, refresh, notify } = useApp();
  const [form, setForm] = useState({
      slug: section?.slug ?? "",
      title: section?.title ?? "",
      description: section?.description ?? "",
      is_archived: section?.is_archived ?? false,
    }),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  return (
    <Modal
      title={section ? "Изменить раздел" : "Новый раздел"}
      onClose={onClose}
    >
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await api(
              "/admin/sections" + (section ? "/" + section.id : ""),
              mutation(
                section ? "PATCH" : "POST",
                form,
                session.staff!.csrf_token,
              ),
            );
            refresh();
            notify("Раздел сохранён.");
            onClose();
          } catch (e) {
            setError(message(e));
          } finally {
            setBusy(false);
          }
        }}
      >
        <label>
          Название
          <input
            required
            minLength={2}
            maxLength={80}
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
          />
        </label>
        <label>
          Короткий адрес
          <input
            required
            pattern="[a-z0-9-]{2,40}"
            title="От 2 до 40 латинских букв, цифр или дефисов"
            maxLength={40}
            value={form.slug}
            placeholder="new-section"
            onChange={(e) => setForm({ ...form, slug: e.target.value })}
          />
        </label>
        <label>
          Описание
          <textarea
            maxLength={240}
            rows={3}
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
        </label>
        <label className="checkbox">
          <input
            type="checkbox"
            checked={form.is_archived}
            onChange={(e) =>
              setForm({ ...form, is_archived: e.target.checked })
            }
          />
          В архиве — нельзя создавать новые темы
        </label>
        <ErrorBox text={error} />
        <div className="dialog-actions">
          <button type="button" className="button ghost" onClick={onClose}>
            Отмена
          </button>
          <button className="button primary" disabled={busy}>
            Сохранить
          </button>
        </div>
      </form>
    </Modal>
  );
}

function StaffModal({ onClose }: { onClose: () => void }) {
  const { session, refresh, notify } = useApp();
  const [form, setForm] = useState({
      login: "",
      password: "",
      role: "moderator",
    }),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  return (
    <Modal title="Добавить участника команды" onClose={onClose}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await api(
              "/admin/staff",
              mutation("POST", form, session.staff!.csrf_token),
            );
            refresh();
            notify("Учётная запись создана.");
            onClose();
          } catch (e) {
            setError(message(e));
          } finally {
            setBusy(false);
          }
        }}
      >
        <label>
          Логин
          <input
            required
            pattern="[a-zA-Z0-9_-]{3,40}"
            autoComplete="off"
            value={form.login}
            onChange={(e) => setForm({ ...form, login: e.target.value })}
          />
        </label>
        <label>
          Пароль
          <input
            aria-label="Пароль"
            type="password"
            autoComplete="new-password"
            required
            minLength={12}
            maxLength={128}
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
          />
          <span className="field-help">Не менее 12 символов.</span>
        </label>
        <label>
          Роль
          <select
            aria-label="Роль"
            value={form.role}
            onChange={(e) => setForm({ ...form, role: e.target.value })}
          >
            <option value="moderator">Модератор</option>
            <option value="admin">Администратор</option>
          </select>
        </label>
        <ErrorBox text={error} />
        <div className="dialog-actions">
          <button type="button" className="button ghost" onClick={onClose}>
            Отмена
          </button>
          <button className="button primary" disabled={busy}>
            Создать аккаунт
          </button>
        </div>
      </form>
    </Modal>
  );
}

function InfoPage({ about = false }: { about?: boolean }) {
  const rules = [
    [
      "Уважай собеседника",
      "Обсуждай мысли, а не личность. Оскорбления, травля и угрозы недопустимы.",
    ],
    [
      "Береги чужую приватность",
      "Не публикуй имена, адреса, телефоны, переписки и другие личные сведения без согласия человека.",
    ],
    [
      "Давай разговору пространство",
      "Выбирай подходящий раздел, пиши понятный заголовок и не дублируй темы. Не размещай спам.",
    ],
    [
      "Помогай модерации",
      "Если видишь нарушение, отправь жалобу. Не поддерживай конфликт новыми оскорблениями.",
    ],
  ];
  return (
    <div className="narrow-page info-page">
      <Link className="back-link" to="/">
        <ArrowLeft size={16} />К обсуждениям
      </Link>
      <div className="eyebrow">
        {about ? "БЕЗ ИМЕНИ, НО С УВАЖЕНИЕМ" : "ПРОСТЫЕ ДОГОВОРЁННОСТИ"}
      </div>
      <h1>
        {about ? "Как устроена анонимность" : "Правила общения"}
        <span className="heading-dot">.</span>
      </h1>
      {about ? (
        <div className="prose form-card">
          <h2>Здесь не нужен публичный профиль</h2>
          <p>
            Чтобы читать или писать, не нужно указывать имя, телефон или почту.
            Все публикации подписываются «Аноним».
          </p>
          <h2>Сессия вместо аккаунта</h2>
          <p>
            При первом действии записи браузер получает сессию на 30 дней. Она
            позволяет удалять свои сообщения. После её завершения, истечения
            срока или очистки cookie доступ к удалению прежних сообщений
            теряется.
          </p>
          <h2>Что видит система</h2>
          <p>
            Сервис временно связывает сообщения с сессией, чтобы проверять
            авторство и ограничивать спам. Эти связи не публикуются. Сессии
            модераторов и администраторов отделены от анонимного участия.
          </p>
          <p>
            Отсутствие публичного имени не означает полной неотслеживаемости в
            сети. Не публикуйте сведения, по которым можно узнать вас или
            другого человека.
          </p>
          <h2>Черновики</h2>
          <p>
            Незавершённые сообщения сохраняются в текущей вкладке браузера,
            чтобы не потеряться при сбое сети. Они удаляются после успешной
            публикации или закрытия вкладки.
          </p>
          <Link className="button ghost" to="/rules">
            Посмотреть правила
            <ArrowRight size={16} />
          </Link>
        </div>
      ) : (
        <>
          <p className="info-lead">
            Свобода разговора начинается с уважения к тем, кто рядом.
          </p>
          <div className="rules-list">
            {rules.map(([title, body], i) => (
              <article key={title}>
                <span>0{i + 1}</span>
                <div>
                  <h2>{title}</h2>
                  <p>{body}</p>
                </div>
              </article>
            ))}
          </div>
          <div className="notice">
            <ShieldCheck size={20} />
            Модераторы могут скрывать нарушающие правила сообщения и закрывать
            темы. Причина каждого решения сохраняется в журнале.
          </div>
        </>
      )}
    </div>
  );
}
