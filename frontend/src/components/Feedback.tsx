import type { LucideIcon } from "lucide-react";
import { CircleHelp, MessagesSquare } from "lucide-react";
import type { ReactNode } from "react";

export function Empty({
  title,
  children,
  icon: Icon = MessagesSquare,
}: {
  title: string;
  children?: ReactNode;
  icon?: LucideIcon;
}) {
  return (
    <div className="empty">
      <span className="empty-icon">
        <Icon size={28} />
      </span>
      <h3>{title}</h3>
      <p>{children}</p>
    </div>
  );
}

export function ErrorBox({
  text,
  retry,
}: {
  text: string;
  retry?: () => void;
}) {
  return text ? (
    <div className="error" role="alert">
      <CircleHelp size={18} />
      <div>
        {text}
        {retry && <button onClick={retry}>Повторить</button>}
      </div>
    </div>
  ) : null;
}

export function Loading() {
  return (
    <div className="loading" role="status">
      <span />
      Загружаем обсуждения…
    </div>
  );
}
