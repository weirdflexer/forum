import type { LucideIcon } from "lucide-react";
import {
BookOpen,
Code2,
Hash,
Heart,
Lightbulb,
MessagesSquare,
} from "lucide-react";

const icons: Record<string, LucideIcon> = {
  general: MessagesSquare,
  tech: Code2,
  study: BookOpen,
  life: Heart,
  ideas: Lightbulb,
};

export function SectionIcon({
  slug,
  size = 19,
}: {
  slug: string;
  size?: number;
}) {
  const Icon = icons[slug] ?? Hash;
  return <Icon size={size} aria-hidden="true" />;
}
