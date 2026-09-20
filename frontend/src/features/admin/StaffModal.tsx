import { useState } from "react";
import { api, mutation } from "../../api/client";
import { useApp } from "../../app/AppProvider";
import { ErrorBox } from "../../components/Feedback";
import { Modal } from "../../components/Modal";
import { useAsyncAction } from "../../hooks/useAsyncAction";

export function StaffModal({ onClose }: { onClose: () => void }) {
  const { session, refresh, notify } = useApp();
  const [form, setForm] = useState({
    login: "",
    password: "",
    role: "moderator",
  });
  const { error, busy, run } = useAsyncAction();
  return (
    <Modal title="Добавить участника команды" onClose={onClose}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          await run(async () => {
            await api(
              "/admin/staff",
              mutation("POST", form, session.staff!.csrf_token),
            );
            refresh();
            notify("Учётная запись создана.");
            onClose();
          });
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
