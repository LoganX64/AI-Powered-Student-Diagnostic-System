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
    const controller = new AbortController();

    fetch(src, { signal: controller.signal })
      .then((r) => {
        // fetch only rejects on network failure, so a 404 used to resolve with
        // the server's HTML error page — which was then injected as markup
        // below. Anything non-OK is simply "no icon".
        return r.ok ? r.text() : "";
      })
      .then((text) => {
        setSvg(text);
      })
      .catch(() => {
        // Aborted (src changed / unmounted) or network failure — no icon.
      });

    return () => {
      controller.abort();
    };
  }, [src]);

  if (!svg) return null;

  const sanitized = sanitizeSvg(svg);
  if (!sanitized) return null;

  const fixed = sanitized.replace(
    /<svg/,
    '<svg width="100%" height="100%" style="color: currentColor"',
  );

  return (
    <span
      className={cn("inline-block [&>svg]:w-full [&>svg]:h-full", className)}
      dangerouslySetInnerHTML={{ __html: fixed }}
    />
  );
}

// Parse the fetched markup and strip anything that can execute script before it
// reaches the DOM: <script>, event-handler attributes (on*), and javascript: URLs.
function sanitizeSvg(markup: string): string {
  const doc = new DOMParser().parseFromString(markup, "image/svg+xml");
  const root = doc.documentElement;
  if (root.tagName.toLowerCase() !== "svg") return "";

  const walk = (node: Element) => {
    if (node.tagName.toLowerCase() === "script") {
      node.remove();
      return;
    }
    for (const attr of Array.from(node.attributes)) {
      const name = attr.name.toLowerCase();
      const value = attr.value.trim().toLowerCase();
      if (name.startsWith("on") || value.startsWith("javascript:")) {
        node.removeAttribute(attr.name);
      }
    }
    Array.from(node.children).forEach((child) => walk(child as Element));
  };
  walk(root);
  return new XMLSerializer().serializeToString(root);
}
