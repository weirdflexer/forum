import {
  ArrowDownWideNarrow,
  ArrowRight,
  ChevronRight,
  Compass,
  LockKeyhole,
  Plus,
  Search,
  Sparkles,
  X,
} from "lucide-react";
import type { FormEvent } from "react";
import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import type { Page, Topic } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { Empty, ErrorBox, Loading } from "../../components/Feedback";
import { Pager } from "../../components/Pager";
import { SectionIcon } from "../../components/SectionIcon";
import { useLoad } from "../../hooks/useLoad";
import { countLabel } from "../../lib/format";
import { TopicRow } from "../topics/TopicRow";

export function HomePage() {
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
