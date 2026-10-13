import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import router from "../routes/routes";
import { TooltipProvider } from "./components/ui/tooltip";
import { Toaster } from "./components/ui/sonner";
import "./index.css";

const OG_DESCRIPTION =
  "Multi-dimensional student diagnostics for coaching institutes: SQI scoring, live proctoring, exam autosave, and topic-level learning gap analysis.";

function setMeta(attribute: "property" | "name", key: string, content: string) {
  const selector = `meta[${attribute}="${key}"]`;
  let tag = document.head.querySelector<HTMLMetaElement>(selector);
  if (!tag) {
    tag = document.createElement("meta");
    tag.setAttribute(attribute, key);
    document.head.appendChild(tag);
  }
  tag.setAttribute("content", content);
}

// Crawlers that do not execute JS read the hardcoded URL in index.html. This
// rewrites it to the actual serving origin so preview and staging deploys
// resolve correctly in browser share previews.
function applyRuntimeMetadata() {
  const origin = window.location.origin;
  const image = `${origin}/og-image.png`;

  setMeta("property", "og:url", window.location.href);
  setMeta("property", "og:image", image);
  setMeta("name", "twitter:image", image);

  const description = document.head.querySelector<HTMLMetaElement>(
    'meta[property="og:description"]',
  );
  if (description) setMeta("property", "og:description", OG_DESCRIPTION);
}

applyRuntimeMetadata();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <TooltipProvider>
      <RouterProvider router={router} />
      <Toaster richColors position="top-right" />
    </TooltipProvider>
  </StrictMode>,
);

const splash = document.getElementById("splash-loader");
const configError = (window as unknown as { __API_CONFIG_ERROR__?: boolean }).__API_CONFIG_ERROR__;
if (splash && !configError) {
  splash.classList.add("hidden");
  setTimeout(() => splash.remove(), 500);
}
