import { useState } from "react";
import { api, mutation } from "../../api/client";
import type { Post } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { ErrorBox } from "../../components/Feedback";
import { Modal } from "../../components/Modal";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { reasons } from "../moderation/constants";

export function ReportModal({
  post,
  onClose,
}: {
  post: Post;
  onClose: () => void;
}) {
  const { ensureSession, notify } = useApp();
  const [reason, setReason] = useState("spam"),
    [comment, setComment] = useState("");
  const { error, busy, run } = useAsyncAction();
  return (
    <Modal title="Сообщить о нарушении" onClose={onClose}>
      <p className="muted-text">
        Жалоба на сообщение #{post.number}. Модератор рассмотрит её и примет
        решение.
      </p>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          await run(async () => {
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
          });
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
