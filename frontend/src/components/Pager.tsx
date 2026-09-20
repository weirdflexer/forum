import { ChevronLeft, ChevronRight } from "lucide-react";

export function Pager({
  page,
  total,
  onPage,
}: {
  page: number;
  total: number;
  onPage: (n: number) => void;
}) {
  const last = Math.ceil(total / 20);
  return last > 1 ? (
    <nav className="pagination" aria-label="Страницы">
      <button
        className="button ghost small"
        disabled={page <= 1}
        onClick={() => onPage(page - 1)}
      >
        <ChevronLeft size={16} />
        Назад
      </button>
      <span>
        {page} / {last}
      </span>
      <button
        className="button ghost small"
        disabled={page >= last}
        onClick={() => onPage(page + 1)}
      >
        Далее
        <ChevronRight size={16} />
      </button>
    </nav>
  ) : null;
}
