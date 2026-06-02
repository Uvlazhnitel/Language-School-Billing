import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = env.VITE_DEV_API_TARGET?.trim() || "http://127.0.0.1:8080";

  return {
    plugins: [react()],
    server: {
      proxy: {
        "/api": apiTarget,
        "/healthz": apiTarget,
      },
    },
    test: {
      environment: "node",
    },
  };
});
