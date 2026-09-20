import { api, mutation } from "../../api/client";
import type { Post } from "../../api/types";
import { useApp } from "../../app/AppProvider";
import { ErrorBox } from "../../components/Feedback";
import { Modal } from "../../components/Modal";
import { useAsyncAction } from "../../hooks/useAsyncAction";

export function DeletePostModal({
  post,
  onClose,
}: {
  post: Post;
  onClose: () => void;
}) {
  const { session, refresh, notify } = useApp();
  const { error, busy, run } = useAsyncAction();
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
            await run(async () => {
              await api(
                "/posts/" + post.id,
                mutation("DELETE", undefined, session.csrf_token),
              );
              refresh();
              notify("Сообщение удалено.");
              onClose();
            });
          }}
        >
          Удалить сообщение
        </button>
      </div>
    </Modal>
  );
}
