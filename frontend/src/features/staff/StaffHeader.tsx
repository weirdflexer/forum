import { Flag, LogOut, Settings2 } from "lucide-react";
import { useState } from "react";
import { NavLink } from "react-router-dom";
import { api, mutation } from "../../api/client";
import { useApp } from "../../app/AppProvider";
import { ErrorBox } from "../../components/Feedback";
import { message } from "../../lib/errors";

export function StaffHeader({ admin = false }: { admin?: boolean }) {
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
