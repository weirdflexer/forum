import { Plus, Shield } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { api, mutation } from "../../api/client";
import type { Section, Staff } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { Empty, ErrorBox } from "../../components/Feedback";
import { Modal } from "../../components/Modal";
import { SectionIcon } from "../../components/SectionIcon";
import { useLoad } from "../../hooks/useLoad";
import { message } from "../../lib/errors";
import { StaffHeader } from "../staff/StaffHeader";
import { SectionModal } from "./SectionModal";
import { StaffModal } from "./StaffModal";

export function AdminPage() {
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
