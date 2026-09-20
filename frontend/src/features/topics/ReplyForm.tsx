import { MessageCircle, Send, ShieldCheck } from "lucide-react";
import type { FormEvent } from "react";
import { useApp } from "../../app/AppProvider";
import { Count } from "../../components/Count";
import { ErrorBox } from "../../components/Feedback";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { useDraft } from "../../hooks/useDraft";
import { usePublication } from "./usePublication";

export function ReplyForm({
  topicId,
  onPublished,
}: {
  topicId: string;
  onPublished: (page: number) => void;
}) {
  const { refresh, notify } = useApp();
  const [body, setBody] = useDraft("forum:reply:" + topicId, "");
  const { error, busy, run } = useAsyncAction();
  const publish = usePublication(
    "reply:" + topicId,
    `/topics/${topicId}/posts`,
  );
  const send = (event: FormEvent) => {
    event.preventDefault();
    void run(async () => {
      const out = await publish<{ page: number }>({ body: body.trim() });
      setBody("");
      onPublished(out.page);
      refresh();
      notify("Ответ опубликован.");
    });
  };
  return (
    <form className="reply-form form-card" onSubmit={send}>
      <h2>
        <MessageCircle size={20} />
        Продолжить разговор
      </h2>
      <label className="sr-only" htmlFor="reply">
        Ваш ответ
      </label>
      <textarea
        id="reply"
        required
        maxLength={10000}
        rows={5}
        placeholder="Что ты думаешь об этом?"
        value={body}
        onChange={(e) => setBody(e.target.value)}
      />
      <Count text={body} max={5000} />
      <ErrorBox text={error} />
      <div className="form-bottom">
        <p>
          <ShieldCheck size={15} />
          Ответ будет опубликован анонимно
        </p>
        <button className="button primary" disabled={busy}>
          {busy ? (
            "Отправляем…"
          ) : (
            <>
              Отправить ответ
              <Send size={16} />
            </>
          )}
        </button>
      </div>
    </form>
  );
}
