import { useRef, useState, useCallback, type ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface HoverEffectItem {
  title: string;
  description: string;
  icon?: ReactNode;
}

export const HoverEffect = ({
  items,
  className,
}: {
  items: HoverEffectItem[];
  className?: string;
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const cardRefs = useRef<(HTMLDivElement | null)[]>([]);
  const [highlight, setHighlight] = useState<{ x: number; y: number; w: number; h: number } | null>(null);

  const onMouseEnter = useCallback((idx: number) => {
    const container = containerRef.current;
    const card = cardRefs.current[idx];
    if (!container || !card) return;
    const cRect = container.getBoundingClientRect();
    const r = card.getBoundingClientRect();
    setHighlight({
      x: r.left - cRect.left,
      y: r.top - cRect.top,
      w: r.width,
      h: r.height,
    });
  }, []);

  const onMouseLeave = useCallback(() => setHighlight(null), []);

  return (
    <div
      ref={containerRef}
      className={cn(
        "relative grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 py-10",
        className,
      )}
    >
      {/* single sliding highlight */}
      <div
        className={cn(
          "absolute rounded-3xl bg-primary/10 transition-all duration-500 ease-[cubic-bezier(0.25,0.1,0.25,1)]",
          highlight ? "opacity-100" : "opacity-0 pointer-events-none",
        )}
        style={{
          transform: highlight ? `translate(${highlight.x}px, ${highlight.y}px)` : undefined,
          width: highlight?.w,
          height: highlight?.h,
        }}
      />

      {items.map((item, idx) => (
        <div
          key={item.title}
          ref={(el) => { cardRefs.current[idx] = el; }}
          className="relative group block p-2 h-full w-full"
          onMouseEnter={() => onMouseEnter(idx)}
          onMouseLeave={onMouseLeave}
        >
          <Card>
            {item.icon && (
              <div className="mb-4">
                {item.icon}
              </div>
            )}
            <CardTitle>{item.title}</CardTitle>
            <CardDescription>{item.description}</CardDescription>
          </Card>
        </div>
      ))}
    </div>
  );
};

export const Card = ({
  className,
  children,
}: {
  className?: string;
  children?: ReactNode;
}) => {
  return (
    <div
      className={cn(
        "rounded-2xl h-full w-full p-4 overflow-hidden bg-card border border-border group-hover:border-primary/30 relative z-20",
        className,
      )}
    >
      <div className="relative z-50">
        <div className="p-4">{children}</div>
      </div>
    </div>
  );
};

export const CardTitle = ({
  className,
  children,
}: {
  className?: string;
  children?: ReactNode;
}) => {
  return (
    <h3 className={cn("text-primary font-bold tracking-wide mt-4", className)}>
      {children}
    </h3>
  );
};

export const CardDescription = ({
  className,
  children,
}: {
  className?: string;
  children?: ReactNode;
}) => {
  return (
    <p
      className={cn(
        "mt-8 text-muted-foreground tracking-wide leading-relaxed text-sm",
        className,
      )}
    >
      {children}
    </p>
  );
};
