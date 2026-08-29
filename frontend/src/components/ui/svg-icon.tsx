import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";

export function SvgIcon({
  src,
  className,
}: {
  src: string;
  className?: string;
}) {
  const [svg, setSvg] = useState("");

  useEffect(() => {
    fetch(src)
      .then((r) => r.text())
      .then(setSvg);
  }, [src]);

  if (!svg) return null;

  const fixed = svg
    .replace(/<svg/, '<svg width="100%" height="100%" style="color: currentColor"');

  return (
    <span
      className={cn("inline-block [&>svg]:w-full [&>svg]:h-full", className)}
      dangerouslySetInnerHTML={{ __html: fixed }}
    />
  );
}
