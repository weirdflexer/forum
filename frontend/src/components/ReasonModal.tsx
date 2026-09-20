import { useState } from "react";
import { useAsyncAction } from "../hooks/useAsyncAction";
import { ErrorBox } from "./Feedback";
import { Modal } from "./Modal";

export function ReasonModal({
  title,
  label,
  button,
  onClose,
  action,
}: {
  title: string;
  label: string;
  button: string;
  onClose: () => void;
  action: (reason: string) => Promise<void>;
}) {
  const [reason, setReason] = useState("");
  const { error, busy, run } = useAsyncAction();
  return (
    <Modal title={title} onClose={onClose}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          await run(async () => {
            await action(reason.trim());
            onClose();
          });
        }}
      >
        <label>
          {label}
          <textarea
            required
            maxLength={1000}
            rows={4}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        </label>
        <ErrorBox text={error} />
        <div className="dialog-actions">
          <button type="button" className="button ghost" onClick={onClose}>
            Отмена
          </button>
          <button className="button primary" disabled={busy}>
            {button}
          </button>
        </div>
      </form>
    </Modal>
  );
}
