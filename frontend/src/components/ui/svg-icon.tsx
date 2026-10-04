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
    // Guard against a superseded src and against unmount.
    let cancelled = false;

    fetch(src)
      .then((r) => {
        // fetch only rejects on network failure, so a 404 used to resolve with
        // the server's HTML error page — which was then injected as markup
        // below. Anything non-OK is simply "no icon".
        return r.ok ? r.text() : "";
      })
      .then((text) => {
        if (!cancelled) setSvg(text);
      })
      .catch(() => {
        if (!cancelled) setSvg("");
      });

    return () => {
      cancelled = true;
    };
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
