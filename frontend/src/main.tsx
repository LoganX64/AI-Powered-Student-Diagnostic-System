import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import router from "../routes/routes";
import { TooltipProvider } from "./components/ui/tooltip";
import { Toaster } from "./components/ui/sonner";
import "./index.css";

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
