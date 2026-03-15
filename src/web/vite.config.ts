import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    port: 3000,
    proxy: {
      "/auth": { target: "http://localhost:8090", changeOrigin: true },
      "/projects": { target: "http://localhost:8090", changeOrigin: true },
      "/webhooks": { target: "http://localhost:8090", changeOrigin: true },
      "/healthz": { target: "http://localhost:8090", changeOrigin: true },
    },
  },
});
