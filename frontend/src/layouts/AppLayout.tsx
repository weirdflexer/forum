import {
  ArrowRight,
  BookOpen,
  Check,
  ChevronRight,
  Compass,
  LogOut,
  Menu,
  Shield,
  ShieldCheck,
  X,
} from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";
import { api, mutation } from "../api/client";
import { useApp } from "../app/AppProvider";
import { ErrorBox } from "../components/Feedback";
import { Modal } from "../components/Modal";
import { SectionIcon } from "../components/SectionIcon";
import { message } from "../lib/errors";
import { Brand } from "./Brand";

export function AppLayout({ children }: { children: ReactNode }) {
  const {
    session,
    setSession,
    sections,
    sectionsError,
    reloadSections,
    refresh,
    notify,
    toast,
    dismissToast,
  } = useApp();
  const [menu, setMenu] = useState(false);
  const [sessionModal, setSessionModal] = useState(false);
  const [sessionError, setSessionError] = useState("");
  const location = useLocation();
  useEffect(() => {
    setMenu(false);
    window.scrollTo(0, 0);
  }, [location.pathname, location.search]);
  const endSession = async () => {
    try {
      await api("/session", mutation("DELETE", undefined, session.csrf_token));
      setSession({ ...session, active: false, csrf_token: "" });
      setSessionModal(false);
      refresh();
      notify("Анонимная сессия завершена.");
    } catch (e) {
      setSessionError(message(e));
    }
  };

  return (
    <>
      <a className="skip" href="#main">
        К содержимому
      </a>
      <div className="app-shell">
        <aside className={"sidebar " + (menu ? "is-open" : "")}>
          <div className="sidebar-top">
            <Brand />
            <button
              className="icon-button mobile-only"
              onClick={() => setMenu(false)}
              aria-label="Закрыть меню"
            >
              <X />
            </button>
          </div>
          <p className="nav-label">ПРОСТРАНСТВО</p>
          <NavLink
            className={
              "nav-item " +
              (location.search.includes("section_id=") ? "not-selected" : "")
            }
            end
            to="/"
          >
            <Compass size={20} />
            Все обсуждения
          </NavLink>
          <p className="nav-label sections-label">
            РАЗДЕЛЫ <span>{sections.length}</span>
          </p>
          <nav aria-label="Разделы">
            {sections.map((s) => (
              <NavLink
                key={s.id}
                className={
                  "nav-item " +
                  (location.search.includes(s.id) ? "selected" : "")
                }
                to={"/?section_id=" + s.id}
              >
                <SectionIcon slug={s.slug} />
                <span>
                  {s.title}
                  {s.is_archived && <small> · архив</small>}
                </span>
                <span className="nav-count">{s.topic_count}</span>
              </NavLink>
            ))}
          </nav>
          <ErrorBox text={sectionsError} retry={reloadSections} />
          <div className="sidebar-bottom">
            <div className="quiet-card">
              <ShieldCheck size={22} />
              <strong>Слова важнее имени</strong>
              <p>Для разговора не нужны профиль, почта или телефон.</p>
              <Link to="/about">
                Как это работает <ArrowRight size={14} />
              </Link>
            </div>
            <NavLink className="nav-item muted" to="/rules">
              <BookOpen size={18} />
              Правила общения
            </NavLink>
            {session.staff && (
              <NavLink className="nav-item muted" to="/moderation">
                <Shield size={18} />
                Модерация
              </NavLink>
            )}
            <div className="sidebar-footer">
              Без имени · 2026 <span>Будь собой.</span>
            </div>
          </div>
        </aside>
        {menu && (
          <button
            className="menu-scrim"
            onClick={() => setMenu(false)}
            aria-label="Закрыть меню"
          />
        )}
        <div className="workspace">
          <header className="topbar">
            <div className="topbar-left">
              <button
                className="icon-button mobile-only"
                aria-label="Открыть меню"
                onClick={() => setMenu(true)}
              >
                <Menu />
              </button>
              <span className="status-dot" />
              Пространство открытого разговора
            </div>
            <button
              className="session-pill"
              onClick={() => {
                setSessionError("");
                setSessionModal(true);
              }}
            >
              <span className="anon-avatar tiny">?</span>
              <span>Вы — Аноним</span>
              <ChevronRight size={15} />
            </button>
          </header>
          <main id="main">{children}</main>
          <footer className="page-footer">
            <span>Открытый разговор. Простые правила.</span>
            <Link to="/about">
              Об анонимности <ArrowRight size={13} />
            </Link>
          </footer>
        </div>
      </div>
      {toast && (
        <div className="toast" role="status">
          <Check size={18} />
          {toast}
          <button aria-label="Закрыть уведомление" onClick={dismissToast}>
            <X size={16} />
          </button>
        </div>
      )}
      {sessionModal && (
        <Modal
          title="Ваша анонимная сессия"
          onClose={() => setSessionModal(false)}
        >
          <p className="muted-text">
            Сессия создаётся при первой публикации и действует 30 дней. Только в
            этом браузере вы можете удалять свои сообщения.
          </p>
          <p className="notice">
            После завершения сессии вы потеряете возможность удалять прежние
            публикации. Сами сообщения останутся в обсуждениях.
          </p>
          <ErrorBox text={sessionError} />
          <div className="dialog-actions">
            <button
              className="button ghost"
              onClick={() => setSessionModal(false)}
            >
              Остаться
            </button>
            <button
              className="button danger"
              disabled={!session.active}
              onClick={endSession}
            >
              <LogOut size={16} />
              Завершить сессию
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}
