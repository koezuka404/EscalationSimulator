import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const proxyTarget = process.env.API_PROXY_TARGET ?? "http://localhost:8083";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api": { target: proxyTarget, changeOrigin: true },
      "/admin/customers": {
        target: proxyTarget,
        changeOrigin: true,
        bypass(req) {
          if (req.headers.accept?.includes("text/html")) return "/index.html";
        },
      },
      "/ws": { target: proxyTarget, ws: true, changeOrigin: true },
    },
  },
});
