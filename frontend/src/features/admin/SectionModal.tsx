import { useState } from "react";
import { api, mutation } from "../../api/client";
import type { Section } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { ErrorBox } from "../../components/Feedback";
import { Modal } from "../../components/Modal";
import { useAsyncAction } from "../../hooks/useAsyncAction";

export function SectionModal({
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
  });
  const { error, busy, run } = useAsyncAction();
  return (
    <Modal
      title={section ? "Изменить раздел" : "Новый раздел"}
      onClose={onClose}
    >
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          await run(async () => {
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
          });
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
