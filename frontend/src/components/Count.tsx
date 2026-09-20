export function Count({ text, max }: { text: string; max: number }) {
  return (
    <span
      className={"counter " + (Array.from(text).length > max ? "over" : "")}
    >
      {Array.from(text).length} / {max}
    </span>
  );
}
