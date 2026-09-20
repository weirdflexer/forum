import { Check, Flag, Shield, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { api, mutation } from "../../api/client";
import type { Action, Page, Report } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { Empty, ErrorBox, Loading } from "../../components/Feedback";
import { Pager } from "../../components/Pager";
import { ReasonModal } from "../../components/ReasonModal";
import { useLoad } from "../../hooks/useLoad";
import { date } from "../../lib/format";
import { StaffHeader } from "../staff/StaffHeader";
import { reasons } from "./constants";

export function ModerationPage() {
  const { session } = useApp();
  return !session.staff ? (
    <div className="narrow-page">
      <Empty title="Нужен служебный вход" icon={Shield}>
        <Link to="/" className="button primary">
          К обсуждениям
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
    tab === "actions" ? null : `/mod/reports?status=${tab}&page=${page}`,
    version,
  );
  const actions = useLoad<Page<Action>>(
    tab === "actions" ? `/mod/actions?page=${page}` : null,
    version,
  );
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
