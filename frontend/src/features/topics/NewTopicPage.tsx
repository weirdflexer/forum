import { ArrowLeft, ArrowRight, ShieldCheck } from "lucide-react";
import type { FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useApp } from "../../app/AppProvider";
import { Count } from "../../components/Count";
import { ErrorBox } from "../../components/Feedback";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { useDraft } from "../../hooks/useDraft";
import { usePublication } from "./usePublication";

export function NewTopicPage() {
  const { sections, refresh, notify } = useApp(),
    navigate = useNavigate(),
    [params] = useSearchParams();
  const [draft, setDraft] = useDraft("forum:new", {
    title: "",
    body: "",
    section_id: params.get("section_id") ?? "",
  });
  const { error, busy, run } = useAsyncAction();
  const publish = usePublication("new", "/topics");
  const submit = (event: FormEvent) => {
    event.preventDefault();
    void run(async () => {
      const payload = {
        ...draft,
        title: draft.title.trim(),
        body: draft.body.trim(),
      };
      const out = await publish<{ topic_id: string }>(payload);
      setDraft({ title: "", body: "", section_id: "" });
      refresh();
      notify("Тема опубликована. Разговор начался!");
      navigate("/topics/" + out.topic_id);
    });
  };
  return (
    <div className="narrow-page">
      <Link to="/" className="back-link">
        <ArrowLeft size={16} />К обсуждениям
      </Link>
      <div className="page-heading">
        <div>
          <div className="eyebrow">ЕСТЬ ЧТО ОБСУДИТЬ?</div>
          <h1>
            Начни разговор<span className="heading-dot">.</span>
          </h1>
          <p>Хорошая тема начинается с простого вопроса.</p>
        </div>
      </div>
      <form className="form-card" onSubmit={submit}>
        <div className="form-intro">
          <span className="anon-avatar">?</span>
          <div>
            <strong>Аноним</strong>
            <p>Твоё имя останется за кадром</p>
          </div>
          <ShieldCheck size={22} />
        </div>
        <label>
          Раздел
          <select
            aria-label="Раздел"
            required
            value={draft.section_id}
            onChange={(e) => setDraft({ ...draft, section_id: e.target.value })}
          >
            <option value="">Выберите раздел</option>
            {sections
              .filter((s) => !s.is_archived)
              .map((s) => (
                <option value={s.id} key={s.id}>
                  {s.title}
                </option>
              ))}
          </select>
        </label>
        <label>
          Заголовок
          <input
            aria-label="Заголовок"
            required
            minLength={5}
            maxLength={240}
            value={draft.title}
            placeholder="О чём хочется поговорить?"
            onChange={(e) => setDraft({ ...draft, title: e.target.value })}
          />
          <Count text={draft.title} max={120} />
        </label>
        <label>
          Первое сообщение
          <textarea
            aria-label="Первое сообщение"
            required
            maxLength={10000}
            rows={9}
            value={draft.body}
            placeholder="Добавь контекст, задай вопрос или расскажи свою историю…"
            onChange={(e) => setDraft({ ...draft, body: e.target.value })}
          />
          <Count text={draft.body} max={5000} />
        </label>
        <ErrorBox text={error} />
        <div className="form-bottom">
          <p>
            Черновик хранится в этой вкладке.
            <br />
            Публикуя, соблюдай <Link to="/rules">правила общения</Link>.
          </p>
          <button className="button primary" disabled={busy}>
            {busy ? (
              "Публикуем…"
            ) : (
              <>
                Опубликовать
                <ArrowRight size={17} />
              </>
            )}
          </button>
        </div>
      </form>
    </div>
  );
}
