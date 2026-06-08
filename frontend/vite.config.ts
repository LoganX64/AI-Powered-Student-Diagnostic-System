import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path/win32";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const backendUrl = env.VITE_BACKEND_URL;
  const port = parseInt(env.VITE_PORT);

  if (!env.VITE_PORT) {
    throw new Error("VITE_PORT environment variable is required");
  }
  if (!backendUrl) {
    throw new Error("VITE_BACKEND_URL environment variable is required");
  }

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      port: port,
      host: true,
      proxy: {
        "/student": backendUrl,
        "/admin": backendUrl,
        "/coach": backendUrl,
        "/auth": backendUrl,
      },
    },
  };
});
