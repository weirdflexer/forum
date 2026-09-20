import { ArrowLeft, ArrowRight, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../../api/client";
import type { Session } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { Empty, ErrorBox } from "../../components/Feedback";
import { useAsyncAction } from "../../hooks/useAsyncAction";

export function LoginPage() {
  const { session, setSession, notify } = useApp();
  const [login, setLogin] = useState(""),
    [password, setPassword] = useState("");
  const { error, busy, run } = useAsyncAction();
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
          await run(async () => {
            const s = await api<Session>("/staff/session", {
              method: "POST",
              body: JSON.stringify({ login, password }),
            });
            setSession(s);
            setPassword("");
            notify("Добро пожаловать в команду.");
            navigate("/moderation");
          });
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
