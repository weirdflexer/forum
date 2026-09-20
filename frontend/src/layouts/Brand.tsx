import { MessagesSquare } from "lucide-react";
import { Link } from "react-router-dom";

export function Brand() {
  return (
    <Link className="brand" to="/" aria-label="Без имени — главная">
      <span className="brand-mark">
        <MessagesSquare size={26} />
      </span>
      <span>
        без имени<span className="brand-sub">анонимный форум</span>
      </span>
    </Link>
  );
}
