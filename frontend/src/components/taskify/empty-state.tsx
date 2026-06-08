import { type LucideIcon, Sparkles } from "lucide-react";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
}

export function EmptyState({ icon: Icon = Sparkles, title }: EmptyStateProps) {
  return (
    <main className="flex flex-1 items-center justify-center bg-canvas px-6">
      <div className="flex max-w-sm flex-col items-center text-center">
        <div className="mb-4 flex size-14 items-center justify-center rounded-full border border-border bg-card text-muted-foreground shadow-sm">
          <Icon className="size-6" />
        </div>
        <h2 className="text-lg font-semibold tracking-normal text-foreground">
          {title}
        </h2>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          Esta seccion estara disponible en la proxima actualizacion.
        </p>
      </div>
    </main>
  );
}
